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
	failGetVmi    error
	failCreateVmi error
}

func (c clientStub) GetVirtualMachineInstance(_, _ string) (*kvcorev1.VirtualMachineInstance, error) {
	return nil, c.failGetVmi
}

func (c clientStub) CreateVirtualMachineInstance(_ string, _ *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	return nil, c.failCreateVmi
}
