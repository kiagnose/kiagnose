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

package launcher_test

import (
	"context"
	"errors"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	k8scorev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/checkup"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/launcher"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/reporter"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

func TestLauncherShouldFail(t *testing.T) {
	testClient := newFakeClient()
	testCheckup := checkup.New(
		testClient,
		k8scorev1.NamespaceDefault,
		config.CheckupParameters{},
		&checkerStub{checkFailure: errorCheck},
	)
	testLauncher := launcher.New(testCheckup, reporterStub{})

	assert.ErrorContains(t, testLauncher.Run(), errorCheck.Error())
}

func TestLauncherShouldRunSuccessfully(t *testing.T) {
	testClient := newFakeClient()
	testCheckup := checkup.New(
		testClient,
		k8scorev1.NamespaceDefault,
		config.CheckupParameters{},
		&checkerStub{},
	)
	testLauncher := launcher.New(testCheckup, reporterStub{})

	assert.NoError(t, testLauncher.Run())
}

func TestLauncherShould(t *testing.T) {
	t.Run("run successfully", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{}, reporterStub{})
		assert.NoError(t, testLauncher.Run())
	})

	t.Run("fail when report is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{}, reporterStub{failReport: errorReport})
		assert.ErrorContains(t, testLauncher.Run(), errorReport.Error())
	})

	t.Run("fail when preflight is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failPreflight: errorPreflight}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorPreflight.Error())
	})

	t.Run("fail when preflight and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failPreflight: errorPreflight},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorPreflight.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when setup is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failSetup: errorSetup}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorSetup.Error())
	})

	t.Run("fail when setup and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failSetup: errorSetup},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failRun: errorRun}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorRun.Error())
	})

	t.Run("fail when teardown is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failTeardown: errorTeardown}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorTeardown.Error())
	})

	t.Run("fail when run and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when teardown and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failTeardown: errorTeardown},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run, teardown and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun, failTeardown: errorTeardown},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})
}

func TestLauncherShouldSuccessfullyProduceStatusResults(t *testing.T) {
	const testConfigMapName = "results"
	const sourceNodeName = "worker1"
	const targetNodeName = "worker2"
	testClient := newFakeClient()
	testReporter := reporter.New(testClient, k8scorev1.NamespaceDefault, testConfigMapName)
	testCheckup := checkup.New(
		testClient,
		k8scorev1.NamespaceDefault,
		config.CheckupParameters{
			SourceNodeName: sourceNodeName,
			TargetNodeName: targetNodeName,
		},
		&checkerStub{},
	)
	testLauncher := launcher.New(testCheckup, testReporter)

	assert.NoError(t, testLauncher.Run())

	expectedResults := status.Results{
		SourceNode: sourceNodeName,
		TargetNode: targetNodeName,
	}
	assert.Equal(t, testCheckup.Results(), expectedResults)
}

var (
	errorPreflight = errors.New("preflight check error")
	errorSetup     = errors.New("setup error")
	errorRun       = errors.New("run error")
	errorTeardown  = errors.New("teardown error")
	errorReport    = errors.New("report error")
)

type checkupStub struct {
	failPreflight error
	failSetup     error
	failRun       error
	failTeardown  error
}

func (s checkupStub) Preflight() error {
	return s.failPreflight
}

func (s checkupStub) Setup(_ context.Context) error {
	return s.failSetup
}

func (s checkupStub) Run() error {
	return s.failRun
}

func (s checkupStub) Teardown(_ context.Context) error {
	return s.failTeardown
}

func (s checkupStub) Results() status.Results {
	return status.Results{}
}

type reporterStub struct {
	failReport error
}

func (r reporterStub) Report(_ status.Status) error {
	return r.failReport
}

type fakeClient struct {
	vmiTracker         map[string]*kvcorev1.VirtualMachineInstance
	returnNetAttachDef *netattdefv1.NetworkAttachmentDefinition
}

// newFakeClient returns fakeClient that tracks VMIs.
// The VMI tracker acts as the cluster DB and keep records regarding
// the VMIs that were created or deleted.
func newFakeClient() *fakeClient {
	return &fakeClient{
		vmiTracker:         map[string]*kvcorev1.VirtualMachineInstance{},
		returnNetAttachDef: &netattdefv1.NetworkAttachmentDefinition{},
	}
}

func (c *fakeClient) GetVirtualMachineInstance(namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	vmi, exists := c.vmiTracker[vmiKey(namespace, name)]
	if !exists {
		return nil, k8serrors.NewNotFound(kvcorev1.Resource("VirtualMachineInstance"), "")
	}
	return vmi, nil
}

// CreateVirtualMachineInstance adds the given VMI to the VMI tracker,
// and adds VirtualMachineInstanceReady condition to the VMI status.
func (c *fakeClient) CreateVirtualMachineInstance(
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	vmi.Status.Conditions = append(vmi.Status.Conditions,
		kvcorev1.VirtualMachineInstanceCondition{
			Type:   kvcorev1.VirtualMachineInstanceAgentConnected,
			Status: k8scorev1.ConditionTrue,
		},
	)
	vmi.Status.NodeName = vmi.Spec.NodeSelector[k8scorev1.LabelHostname]
	c.vmiTracker[vmiKey(namespace, vmi.Name)] = vmi
	return vmi, nil
}

func (c *fakeClient) DeleteVirtualMachineInstance(namespace, name string) error {
	delete(c.vmiTracker, vmiKey(namespace, name))
	return nil
}

func (c *fakeClient) SerialConsole(namespace, vmiName string, timeout time.Duration) (kubecli.StreamInterface, error) {
	return nil, nil
}

func (c *fakeClient) GetNetworkAttachmentDefinition(_, _ string) (*netattdefv1.NetworkAttachmentDefinition, error) {
	return c.returnNetAttachDef, nil
}

func vmiKey(namespace, name string) string {
	return namespace + "/" + name
}

var errorCheck = errors.New("check failed")

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

func (c *fakeClient) UpdateConfigMap(_, _ string, _ map[string]string) error {
	return nil
}
