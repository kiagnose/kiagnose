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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/checkup"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/launcher"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/reporter"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

const testCheckupUID = "0123456789"

const (
	testNamespace = "target-ns"
	configMapName = "checkup1"
)

func TestLauncherShouldFail(t *testing.T) {
	testClient := newFakeClient()
	testCheckup := checkup.New(
		testClient,
		testCheckupUID,
		testNamespace,
		config.Config{},
		&checkerStub{checkFailure: errorCheck},
	)
	testLauncher := launcher.New(testCheckup, &reporterStub{})

	assert.ErrorContains(t, testLauncher.Run(), errorCheck.Error())
}

func TestLauncherShouldRunSuccessfully(t *testing.T) {
	testClient := newFakeClient()
	testCheckup := checkup.New(
		testClient,
		testCheckupUID,
		testNamespace,
		config.Config{},
		&checkerStub{},
	)
	testLauncher := launcher.New(testCheckup, &reporterStub{})

	assert.NoError(t, testLauncher.Run())
}

func TestLauncherShould(t *testing.T) {
	t.Run("run successfully", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{}, &reporterStub{})
		assert.NoError(t, testLauncher.Run())
	})

	t.Run("fail when report is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{}, &reporterStub{failReport: errorReport})
		assert.ErrorContains(t, testLauncher.Run(), errorReport.Error())
	})

	t.Run("fail when setup is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failSetup: errorSetup}, &reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorSetup.Error())
	})

	t.Run("fail when setup and 2nd report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failSetup: errorSetup},
			&reporterStub{failReport: errorReport, failOnSecondReport: true},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failRun: errorRun}, &reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorRun.Error())
	})

	t.Run("fail when teardown is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failTeardown: errorTeardown}, &reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorTeardown.Error())
	})

	t.Run("fail when run and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun},
			&reporterStub{failReport: errorReport, failOnSecondReport: true},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when teardown and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failTeardown: errorTeardown},
			&reporterStub{failReport: errorReport, failOnSecondReport: true},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run, teardown and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun, failTeardown: errorTeardown},
			&reporterStub{failReport: errorReport, failOnSecondReport: true},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})
}

func TestLauncherShouldSuccessfullyProduceStatusResults(t *testing.T) {
	const sourceNodeName = "worker1"
	const targetNodeName = "worker2"
	simpleFakeClient := fake.NewSimpleClientset(newConfigMap())
	testClient := newFakeClient()

	testReporter := reporter.New(simpleFakeClient, testNamespace, configMapName)
	testCheckup := checkup.New(
		testClient,
		testCheckupUID,
		testNamespace,
		config.Config{
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
	assert.Equal(t, expectedResults, testCheckup.Results())
}

var (
	errorSetup    = errors.New("setup error")
	errorRun      = errors.New("run error")
	errorTeardown = errors.New("teardown error")
	errorReport   = errors.New("report error")
)

type checkupStub struct {
	failSetup    error
	failRun      error
	failTeardown error
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
	reportCalls int
	failReport  error
	// The launcher calls the report twice: To mark the start timestamp and
	// then to update the checkup results.
	// Use this flag to cause the second report to fail.
	failOnSecondReport bool
}

func (r *reporterStub) Report(_ status.Status) error {
	r.reportCalls++
	if r.failOnSecondReport && r.reportCalls == 2 {
		return r.failReport
	} else if !r.failOnSecondReport {
		return r.failReport
	}
	return nil
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

func (c *fakeClient) GetVirtualMachineInstance(_ context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	vmi, exists := c.vmiTracker[vmiKey(namespace, name)]
	if !exists {
		return nil, k8serrors.NewNotFound(kvcorev1.Resource("VirtualMachineInstance"), "")
	}
	return vmi, nil
}

// CreateVirtualMachineInstance adds the given VMI to the VMI tracker,
// adds VirtualMachineInstanceAgentConnected condition to the VMI status and
// the node name according to node affinity rule with 'kubernetes.io/hostname' label selector.
func (c *fakeClient) CreateVirtualMachineInstance(
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	vmi.Status.Interfaces = append(vmi.Status.Interfaces, kvcorev1.VirtualMachineInstanceNetworkInterface{
		IP: "0.0.0.0",
	})

	if vmi.Spec.Affinity != nil && vmi.Spec.Affinity.NodeAffinity != nil {
		term := vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
		req := term.MatchExpressions[0]
		if req.Key == k8scorev1.LabelHostname {
			vmi.Status.NodeName = req.Values[0]
		}
	}

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

func newConfigMap() *k8scorev1.ConfigMap {
	return &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{},
	}
}
