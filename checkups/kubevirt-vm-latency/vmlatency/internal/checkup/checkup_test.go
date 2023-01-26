/*
 * This file is part of the kiagnose project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package checkup_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/checkup"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/vmi"
)

const (
	testPodName               = "my-pod"
	testPodUID                = "0123456789-0123456789"
	testCheckupUID            = "0123456789"
	testNamespace             = "default"
	testNetAttachDefName      = "blue-net"
	testSampleDurationSeconds = 1
	testTimeout               = time.Nanosecond
)

func TestCheckupSetupShouldFailWhen(t *testing.T) {
	t.Run("NetworkAttachmentDefinition does not exist", func(t *testing.T) {
		expectedError := errors.New("get netAttachDef test error")
		testClient := newTestClient()
		testClient.failGetNetAttachDef = expectedError
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, newTestsCheckupParameters(), &checkerStub{})

		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedError.Error())
	})

	t.Run("failed to create a VM", func(t *testing.T) {
		expectedError := errors.New("vmi create test error")
		testClient := newTestClient()
		testClient.failCreateVmi = expectedError
		testClient.returnNetAttachDef = &netattdefv1.NetworkAttachmentDefinition{}
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, newTestsCheckupParameters(), &checkerStub{})

		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedError.Error())
	})

	t.Run("VMs were not ready before timeout expiration", func(t *testing.T) {
		expectedError := errors.New("timed out")
		testClient := newTestClient()
		testClient.failGetVmi = expectedError
		testClient.returnNetAttachDef = &netattdefv1.NetworkAttachmentDefinition{}
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, newTestsCheckupParameters(), &checkerStub{})

		testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		assert.ErrorContains(t, testCheckup.Setup(testCtx), expectedError.Error())
		assert.Len(t, testClient.createdVmis, 0)
	})
}

func TestCheckupTeardownShouldFailWhen(t *testing.T) {
	t.Run("failed to delete a VM", func(t *testing.T) {
		testClient := newTestClient()
		testClient.returnNetAttachDef = &netattdefv1.NetworkAttachmentDefinition{}
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, newTestsCheckupParameters(), &checkerStub{})

		testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		assert.NoError(t, testCheckup.Setup(testCtx))

		expectedErr := errors.New("delete vmi test error")
		testClient.failDeleteVmi = expectedErr

		assert.ErrorContains(t, testCheckup.Teardown(testCtx), expectedErr.Error())
	})

	t.Run("VMs were not disposed before timeout expiration", func(t *testing.T) {
		testClient := newTestClient()
		testClient.returnNetAttachDef = &netattdefv1.NetworkAttachmentDefinition{}
		testClient.skipDeletion = true
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, newTestsCheckupParameters(), &checkerStub{})

		assert.NoError(t, testCheckup.Setup(context.Background()))

		testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		assert.ErrorContains(t, testCheckup.Teardown(testCtx), "timed out")
	})
}

type checkupSetupCreateVmiTestCase struct {
	description                string
	netAttachDef               *netattdefv1.NetworkAttachmentDefinition
	expectedIfaceBindingMethod kvcorev1.InterfaceBindingMethod
}

func TestCheckupSetupShouldCreateVMsWith(t *testing.T) {
	const (
		bridgeCniPluginName = "bridge"
		sriovCniPluginName  = "sriov"
	)
	testsCases := []checkupSetupCreateVmiTestCase{
		{
			"bridge interface when NetAttachDef is not using  SR-IOV CNI",
			newTestNetAttachDef(bridgeCniPluginName),
			kvcorev1.InterfaceBindingMethod{Bridge: &kvcorev1.InterfaceBridge{}},
		},
		{
			"SR-IOV interface when NetAttachDef uses SR-IOV CNI",
			newTestNetAttachDef(sriovCniPluginName),
			kvcorev1.InterfaceBindingMethod{SRIOV: &kvcorev1.InterfaceSRIOV{}},
		},
	}
	for _, testCase := range testsCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newTestClient()
			testClient.returnNetAttachDef = testCase.netAttachDef
			testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, newTestsCheckupParameters(), &checkerStub{})

			assert.NoError(t, testCheckup.Setup(context.Background()))
			assert.Len(t, testClient.createdVmis, 2)
			for _, createVMI := range testClient.createdVmis {
				assert.Len(t, createVMI.Spec.Domain.Devices.Interfaces, 1)
				assert.Equal(t, testCase.expectedIfaceBindingMethod, createVMI.Spec.Domain.Devices.Interfaces[0].InterfaceBindingMethod)
			}
		})
	}
}

func TestCheckupSetupShouldCreateVMsWithPodAntiAffinity(t *testing.T) {
	t.Run("when source and target nodes names are not specified", func(t *testing.T) {
		testClient := newTestClient()
		testClient.returnNetAttachDef = newTestNetAttachDef("")
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, config.Config{}, &checkerStub{})

		assert.NoError(t, testCheckup.Setup(context.Background()))

		sourceVMIName := testClient.SourceVMIName()
		targetVMIName := testClient.TargetVMIName()

		assertVmiPodAntiAffinityExist(t, testClient, sourceVMIName)
		assertVmiPodAntiAffinityExist(t, testClient, targetVMIName)
		assertVmiNodeAffinityNotExist(t, testClient, sourceVMIName)
		assertVmiNodeAffinityNotExist(t, testClient, targetVMIName)
	})
}

func TestCheckupSetupShouldCreateVMsWithNodeAffinity(t *testing.T) {
	t.Run("when both source and target node names are specified", func(t *testing.T) {
		const (
			testSourceNode = "source.example.com"
			testTargetNode = "target.example.com"
		)
		testClient := newTestClient()
		testClient.returnNetAttachDef = newTestNetAttachDef("blah")
		testCheckupParams := config.Config{SourceNodeName: testSourceNode, TargetNodeName: testTargetNode}
		testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, testCheckupParams, &checkerStub{})

		assert.NoError(t, testCheckup.Setup(context.Background()))

		sourceVMIName := testClient.SourceVMIName()
		targetVMIName := testClient.TargetVMIName()

		assertVmiNodeAffinityExist(t, testClient, sourceVMIName, testSourceNode)
		assertVmiNodeAffinityExist(t, testClient, targetVMIName, testTargetNode)
		assertVmiPodAntiAffinityNotExist(t, testClient, sourceVMIName)
		assertVmiPodAntiAffinityNotExist(t, testClient, targetVMIName)
	})
}

func TestCheckupSetupShouldCreateOwnerReference(t *testing.T) {
	testClient := newTestClient()
	testClient.returnNetAttachDef = newTestNetAttachDef("blah")
	testCheckupParams := config.Config{
		PodName:                              testPodName,
		PodUID:                               testPodUID,
		NetworkAttachmentDefinitionName:      testNetAttachDefName,
		NetworkAttachmentDefinitionNamespace: testNamespace,
	}
	testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, testCheckupParams, &checkerStub{})

	assert.NoError(t, testCheckup.Setup(context.Background()))

	sourceVMIName := testClient.SourceVMIName()
	targetVMIName := testClient.TargetVMIName()

	sourceVMI, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, sourceVMIName)
	assert.NoError(t, err)

	targetVMI, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, targetVMIName)
	assert.NoError(t, err)

	assertOwnerReferenceExists(t, sourceVMI, testPodName, testPodUID)
	assertOwnerReferenceExists(t, targetVMI, testPodName, testPodUID)
}

func TestCheckupSetupShouldNotCreateOwnerReferenceWhenPodUIDIsEmpty(t *testing.T) {
	testClient := newTestClient()
	testClient.returnNetAttachDef = newTestNetAttachDef("blah")

	const emptyPodUID = ""

	testCheckupParams := config.Config{
		PodName:                              testPodName,
		PodUID:                               emptyPodUID,
		NetworkAttachmentDefinitionName:      testNetAttachDefName,
		NetworkAttachmentDefinitionNamespace: testNamespace,
	}

	testCheckup := checkup.New(testClient, testCheckupUID, testNamespace, testCheckupParams, &checkerStub{})

	assert.NoError(t, testCheckup.Setup(context.Background()))

	sourceVMIName := testClient.SourceVMIName()
	targetVMIName := testClient.TargetVMIName()

	sourceVMI, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, sourceVMIName)
	assert.NoError(t, err)

	targetVMI, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, targetVMIName)
	assert.NoError(t, err)

	assertOwnerReferenceDoesNotExists(t, sourceVMI, testPodName, emptyPodUID)
	assertOwnerReferenceDoesNotExists(t, targetVMI, testPodName, emptyPodUID)
}

func assertVmiPodAntiAffinityExist(t *testing.T, testClient *clientStub, vmiName string) {
	actualVmi, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, vmiName)
	assert.NoError(t, err)
	assert.NotNil(t, actualVmi.Spec.Affinity.PodAntiAffinity)
	expectedPodAntiAffinity := vmi.NewPodAntiAffinity(vmi.Label{Key: checkup.LabelLatencyCheckUID, Value: testCheckupUID})
	assert.Equal(t, expectedPodAntiAffinity, actualVmi.Spec.Affinity.PodAntiAffinity)
}

func assertVmiPodAntiAffinityNotExist(t *testing.T, testClient *clientStub, name string) {
	actualVmi, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, name)
	assert.NoError(t, err)

	assert.Nil(t, actualVmi.Spec.Affinity.PodAntiAffinity)
}

func assertVmiNodeAffinityExist(t *testing.T, testClient *clientStub, vmiName, nodeName string) {
	actualVmi, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, vmiName)
	assert.NoError(t, err)

	actualVmiNodeAffinity := actualVmi.Spec.Affinity.NodeAffinity
	assert.NotNil(t, actualVmiNodeAffinity)

	expectedNodeAffinity := vmi.NewNodeAffinity(nodeName)
	assert.Equal(t, expectedNodeAffinity, actualVmiNodeAffinity)
}

func assertVmiNodeAffinityNotExist(t *testing.T, testClient *clientStub, name string) {
	actualVmi, err := testClient.GetVirtualMachineInstance(context.Background(), testNamespace, name)
	assert.NoError(t, err)

	assert.Nil(t, actualVmi.Spec.Affinity.NodeAffinity)
}

func assertOwnerReferenceExists(t *testing.T, vmiUnderTest *kvcorev1.VirtualMachineInstance, ownerName, ownerUID string) {
	assert.Contains(t, vmiUnderTest.OwnerReferences, metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       ownerName,
		UID:        types.UID(ownerUID),
	})
}

func assertOwnerReferenceDoesNotExists(t *testing.T, vmiUnderTest *kvcorev1.VirtualMachineInstance, ownerName, ownerUID string) {
	assert.NotContains(t, vmiUnderTest.OwnerReferences, metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       ownerName,
		UID:        types.UID(ownerUID),
	})
}

func newTestNetAttachDef(cniPluginName string) *netattdefv1.NetworkAttachmentDefinition {
	return &netattdefv1.NetworkAttachmentDefinition{
		Spec: netattdefv1.NetworkAttachmentDefinitionSpec{
			Config: fmt.Sprintf("{\"type\": %q}", cniPluginName),
		},
	}
}

func newTestsCheckupParameters() config.Config {
	return config.Config{
		NetworkAttachmentDefinitionName:      testNetAttachDefName,
		NetworkAttachmentDefinitionNamespace: testNamespace,
		TargetNodeName:                       "",
		SourceNodeName:                       "",
		SampleDurationSeconds:                testSampleDurationSeconds,
		DesiredMaxLatency:                    0,
	}
}

func newTestClient() *clientStub {
	return &clientStub{createdVmis: map[string]*kvcorev1.VirtualMachineInstance{}}
}

type clientStub struct {
	createdVmis map[string]*kvcorev1.VirtualMachineInstance

	returnNetAttachDef *netattdefv1.NetworkAttachmentDefinition

	failGetNetAttachDef error
	failGetVmi          error
	failCreateVmi       error
	failDeleteVmi       error

	skipDeletion bool
}

func (c *clientStub) GetVirtualMachineInstance(_ context.Context, _, name string) (*kvcorev1.VirtualMachineInstance, error) {
	v, exists := c.createdVmis[name]
	if !exists {
		return nil, k8serrors.NewNotFound(schema.GroupResource{}, "")
	}
	return v, c.failGetVmi
}

func (c *clientStub) CreateVirtualMachineInstance(
	_ context.Context,
	_ string,
	v *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	if c.failCreateVmi != nil {
		return nil, c.failCreateVmi
	}

	v.Status.Interfaces = append(v.Status.Interfaces, kvcorev1.VirtualMachineInstanceNetworkInterface{
		IP: "0.0.0.0",
	})

	c.createdVmis[v.Name] = v

	return v, nil
}

func (c *clientStub) DeleteVirtualMachineInstance(_ context.Context, _, vmiName string) error {
	if !c.skipDeletion {
		delete(c.createdVmis, vmiName)
	}
	return c.failDeleteVmi
}

func (c *clientStub) SerialConsole(_, _ string, _ time.Duration) (kubecli.StreamInterface, error) {
	return nil, nil
}

func (c *clientStub) GetNetworkAttachmentDefinition(_ context.Context, _, _ string) (*netattdefv1.NetworkAttachmentDefinition, error) {
	return c.returnNetAttachDef, c.failGetNetAttachDef
}

func (c *clientStub) SourceVMIName() string {
	for vmiName := range c.createdVmis {
		if strings.HasPrefix(vmiName, checkup.SourceVMINamePrefix) {
			return vmiName
		}
	}

	return ""
}

func (c *clientStub) TargetVMIName() string {
	for vmiName := range c.createdVmis {
		if strings.HasPrefix(vmiName, checkup.TargetVMINamePrefix) {
			return vmiName
		}
	}

	return ""
}

type checkerStub struct {
	checkFailure error
}

func (c *checkerStub) Check(_, _ *kvcorev1.VirtualMachineInstance, _ time.Duration) error {
	return c.checkFailure
}

func (c *checkerStub) MinLatency() time.Duration {
	return 0
}

func (c *checkerStub) AverageLatency() time.Duration {
	return 0
}

func (c *checkerStub) MaxLatency() time.Duration {
	return 0
}

func (c *checkerStub) CheckDuration() time.Duration {
	return 0
}
