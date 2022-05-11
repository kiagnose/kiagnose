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
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	k8scorev1 "k8s.io/api/core/v1"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/checkup"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
)

const (
	testNamespace             = "default"
	testNetAttachDefName      = "blue-net"
	testSampleDurationSeconds = 1
	testTimeout               = time.Nanosecond
)

func TestCheckupSetupShouldFailWhen(t *testing.T) {
	t.Run("failed to create a VM", func(t *testing.T) {
		expectedError := errors.New("vmi create test error")
		testCheckup := checkup.New(
			clientStub{failCreateVmi: expectedError},
			testNamespace,
			newTestsCheckupParameters())

		assert.NoError(t, testCheckup.Preflight())
		assert.ErrorContains(t, testCheckup.Setup(context.Background()), expectedError.Error())
	})

	t.Run("VMs were not ready before timeout expiration", func(t *testing.T) {
		expectedError := errors.New("timed out")
		testCheckup := checkup.New(
			clientStub{failGetVmi: expectedError},
			testNamespace,
			newTestsCheckupParameters())

		testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		assert.NoError(t, testCheckup.Preflight())
		assert.ErrorContains(t, testCheckup.Setup(testCtx), expectedError.Error())
	})
}

func TestCheckupTeardownShouldFailWhen(t *testing.T) {
	t.Run("failed to delete a VM", func(t *testing.T) {
		testClient := &clientStub{}
		testCheckup := checkup.New(
			testClient,
			testNamespace,
			newTestsCheckupParameters())
		testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		assert.NoError(t, testCheckup.Preflight())

		testClient.returnVmi = newTestReadyVmi()

		assert.NoError(t, testCheckup.Setup(testCtx))

		expectedErr := errors.New("delete vmi test error")
		testClient.failDeleteVmi = expectedErr

		assert.ErrorContains(t, testCheckup.Teardown(testCtx), expectedErr.Error())
	})

	t.Run("VMs were not disposed before timeout expiration", func(t *testing.T) {
		testClient := &clientStub{}
		testCheckup := checkup.New(
			testClient,
			testNamespace,
			newTestsCheckupParameters())

		assert.NoError(t, testCheckup.Preflight())

		testClient.returnVmi = newTestReadyVmi()

		assert.NoError(t, testCheckup.Setup(context.Background()))

		testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		assert.ErrorContains(t, testCheckup.Teardown(testCtx), "timed out")
	})
}

func newTestReadyVmi() *kvcorev1.VirtualMachineInstance {
	return &kvcorev1.VirtualMachineInstance{Status: kvcorev1.VirtualMachineInstanceStatus{
		Conditions: []kvcorev1.VirtualMachineInstanceCondition{
			{
				Type:   kvcorev1.VirtualMachineInstanceAgentConnected,
				Status: k8scorev1.ConditionTrue,
			},
		},
	}}
}

func newTestsCheckupParameters() config.CheckupParameters {
	return config.CheckupParameters{
		NetworkAttachmentDefinitionName:      testNetAttachDefName,
		NetworkAttachmentDefinitionNamespace: testNamespace,
		TargetNodeName:                       "",
		SourceNodeName:                       "",
		SampleDurationSeconds:                testSampleDurationSeconds,
		DesiredMaxLatencyMilliseconds:        0,
	}
}

type clientStub struct {
	returnVmi     *kvcorev1.VirtualMachineInstance
	failGetVmi    error
	failCreateVmi error
	failDeleteVmi error
}

func (c clientStub) GetVirtualMachineInstance(_, _ string) (*kvcorev1.VirtualMachineInstance, error) {
	return c.returnVmi, c.failGetVmi
}

func (c clientStub) CreateVirtualMachineInstance(_ string, _ *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	return nil, c.failCreateVmi
}

func (c clientStub) DeleteVirtualMachineInstance(_, _ string) error {
	return c.failDeleteVmi
}
