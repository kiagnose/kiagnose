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

package config_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
)

type configCreateTestCases struct {
	description    string
	params         map[string]string
	expectedConfig config.Config
}

const (
	testPodName                       = "my-pod"
	testPodUID                        = "0123456789-0123456789"
	testNamespace                     = "default"
	testNetAttachDefName              = "blue-net"
	testDesiredMaxLatencyMilliseconds = 100
	testSampleDurationSeconds         = 60
	testSourceNodeName                = "worker1"
	testTargetNodeName                = "worker2"
)

func TestCreateConfigFromParamsShould(t *testing.T) {
	testCases := []configCreateTestCases{
		{
			description: "set default sample duration when parameter is missing",
			params: map[string]string{
				config.NetworkNameParamName:                   testNetAttachDefName,
				config.NetworkNamespaceParamName:              testNamespace,
				config.DesiredMaxLatencyMillisecondsParamName: fmt.Sprintf("%d", testDesiredMaxLatencyMilliseconds),
			},
			expectedConfig: config.Config{
				PodName:                              testPodName,
				PodUID:                               testPodUID,
				SampleDurationSeconds:                config.DefaultSampleDurationSeconds,
				NetworkAttachmentDefinitionName:      testNetAttachDefName,
				NetworkAttachmentDefinitionNamespace: testNamespace,
				DesiredMaxLatency:                    testDesiredMaxLatencyMilliseconds * time.Millisecond,
			},
		},
		{
			description: "set default desired max latency when parameter is missing",
			params: map[string]string{
				config.NetworkNameParamName:           testNetAttachDefName,
				config.NetworkNamespaceParamName:      testNamespace,
				config.SampleDurationSecondsParamName: fmt.Sprintf("%d", testSampleDurationSeconds),
			},
			expectedConfig: config.Config{
				PodName:                              testPodName,
				PodUID:                               testPodUID,
				DesiredMaxLatency:                    config.DefaultDesiredMaxLatencyMilliseconds,
				NetworkAttachmentDefinitionName:      testNetAttachDefName,
				NetworkAttachmentDefinitionNamespace: testNamespace,
				SampleDurationSeconds:                testSampleDurationSeconds,
			},
		},
		{
			description: "set source and target nodes when both are specified",
			params: map[string]string{
				config.NetworkNameParamName:           testNetAttachDefName,
				config.NetworkNamespaceParamName:      testNamespace,
				config.SampleDurationSecondsParamName: fmt.Sprintf("%d", testSampleDurationSeconds),
				config.SourceNodeNameParamName:        testSourceNodeName,
				config.TargetNodeNameParamName:        testTargetNodeName,
			},
			expectedConfig: config.Config{
				PodName:                              testPodName,
				PodUID:                               testPodUID,
				DesiredMaxLatency:                    config.DefaultDesiredMaxLatencyMilliseconds,
				NetworkAttachmentDefinitionName:      testNetAttachDefName,
				NetworkAttachmentDefinitionNamespace: testNamespace,
				SampleDurationSeconds:                testSampleDurationSeconds,
				SourceNodeName:                       testSourceNodeName,
				TargetNodeName:                       testTargetNodeName,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{
				PodName: testPodName,
				PodUID:  testPodUID,
				Params:  testCase.params,
			}
			testConfig, err := config.New(baseConfig)
			assert.NoError(t, err)
			assert.Equal(t, testConfig, testCase.expectedConfig)
		})
	}
}

func TestCreateConfigShouldPreferNonDeprecatedParameters(t *testing.T) {
	testCases := []configCreateTestCases{
		{
			description: "when both deprecated and non-deprecated params are specified",
			params: map[string]string{
				config.NetworkNameDeprecatedParamName:                   testNetAttachDefName + "999",
				config.NetworkNamespaceDeprecatedParamName:              testNamespace + "999",
				config.TargetNodeNameDeprecatedParamName:                testSourceNodeName + "999",
				config.SourceNodeNameDeprecatedParamName:                testTargetNodeName + "999",
				config.DesiredMaxLatencyMillisecondsDeprecatedParamName: fmt.Sprint(testDesiredMaxLatencyMilliseconds + 999),
				config.SampleDurationSecondsDeprecatedParamName:         fmt.Sprint(testSampleDurationSeconds + 999),
				config.NetworkNameParamName:                             testNetAttachDefName,
				config.NetworkNamespaceParamName:                        testNamespace,
				config.TargetNodeNameParamName:                          testSourceNodeName,
				config.SourceNodeNameParamName:                          testTargetNodeName,
				config.DesiredMaxLatencyMillisecondsParamName:           fmt.Sprint(testDesiredMaxLatencyMilliseconds),
				config.SampleDurationSecondsParamName:                   fmt.Sprint(testSampleDurationSeconds),
			},
			expectedConfig: config.Config{
				PodName:                              testPodName,
				PodUID:                               testPodUID,
				NetworkAttachmentDefinitionName:      testNetAttachDefName,
				NetworkAttachmentDefinitionNamespace: testNamespace,
				TargetNodeName:                       testSourceNodeName,
				SourceNodeName:                       testTargetNodeName,
				SampleDurationSeconds:                testSampleDurationSeconds,
				DesiredMaxLatency:                    testDesiredMaxLatencyMilliseconds * time.Millisecond,
			},
		},
		{
			description: "fallback to deprecated parameters when new form is missing",
			params: map[string]string{
				config.NetworkNameDeprecatedParamName:                   testNetAttachDefName,
				config.NetworkNamespaceDeprecatedParamName:              testNamespace,
				config.TargetNodeNameDeprecatedParamName:                testSourceNodeName,
				config.SourceNodeNameDeprecatedParamName:                testTargetNodeName,
				config.DesiredMaxLatencyMillisecondsDeprecatedParamName: fmt.Sprint(testDesiredMaxLatencyMilliseconds),
				config.SampleDurationSecondsDeprecatedParamName:         fmt.Sprint(testSampleDurationSeconds),
			},
			expectedConfig: config.Config{
				PodName:                              testPodName,
				PodUID:                               testPodUID,
				NetworkAttachmentDefinitionName:      testNetAttachDefName,
				NetworkAttachmentDefinitionNamespace: testNamespace,
				TargetNodeName:                       testSourceNodeName,
				SourceNodeName:                       testTargetNodeName,
				SampleDurationSeconds:                testSampleDurationSeconds,
				DesiredMaxLatency:                    testDesiredMaxLatencyMilliseconds * time.Millisecond,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{
				PodName: testPodName,
				PodUID:  testPodUID,
				Params:  testCase.params,
			}
			testConfig, err := config.New(baseConfig)
			assert.NoError(t, err)
			assert.Equal(t, testConfig, testCase.expectedConfig)
		})
	}
}

type configCreateFallingTestCases struct {
	description   string
	expectedError error
	params        map[string]string
}

func TestCreateConfigFromParamsShouldFailWhen(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "params is nil",
			expectedError: config.ErrInvalidParams,
			params:        nil,
		},
		{
			description:   "params is empty",
			expectedError: config.ErrInvalidParams,
			params:        map[string]string{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{Params: testCase.params}
			_, err := config.New(baseConfig)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigFromParamsShouldFailWhenMandatoryParamsAreMissing(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "network name parameter is missing",
			expectedError: config.ErrInvalidNetworkName,
			params: map[string]string{
				config.NetworkNamespaceParamName: testNamespace,
			},
		},
		{
			description:   "network namespace parameter is missing",
			expectedError: config.ErrInvalidNetworkNamespace,
			params: map[string]string{
				config.NetworkNameParamName: testNetAttachDefName,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{Params: testCase.params}
			_, err := config.New(baseConfig)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigFromParamsShouldFailWhenMandatoryParamsAreInvalid(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "network name parameter value is not valid",
			expectedError: config.ErrInvalidNetworkName,
			params: map[string]string{
				config.NetworkNameParamName:      "",
				config.NetworkNamespaceParamName: testNamespace,
			},
		},
		{
			description:   "network namespace parameter value is not valid",
			expectedError: config.ErrInvalidNetworkNamespace,
			params: map[string]string{
				config.NetworkNameParamName:      testNetAttachDefName,
				config.NetworkNamespaceParamName: "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{Params: testCase.params}
			_, err := config.New(baseConfig)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigFromParamsShouldFailWhenNodeNames(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "source node name is set but target node name isn't",
			expectedError: config.ErrIllegalSourceAndTargetNodesCombination,
			params: map[string]string{
				config.NetworkNameParamName:      testNetAttachDefName,
				config.NetworkNamespaceParamName: testNamespace,
				config.SourceNodeNameParamName:   testSourceNodeName,
			},
		},
		{
			description:   "target node name is set but source node name isn't",
			expectedError: config.ErrIllegalSourceAndTargetNodesCombination,
			params: map[string]string{
				config.NetworkNameParamName:      testNetAttachDefName,
				config.NetworkNamespaceParamName: testNamespace,
				config.TargetNodeNameParamName:   testTargetNodeName,
			},
		},
		{
			description:   "source node name is empty",
			expectedError: config.ErrIllegalSourceAndTargetNodesCombination,
			params: map[string]string{
				config.NetworkNameParamName:      testNetAttachDefName,
				config.NetworkNamespaceParamName: testNamespace,
				config.SourceNodeNameParamName:   "",
				config.TargetNodeNameParamName:   testTargetNodeName,
			},
		},
		{
			description:   "target node name is empty",
			expectedError: config.ErrIllegalSourceAndTargetNodesCombination,
			params: map[string]string{
				config.NetworkNameParamName:      testNetAttachDefName,
				config.NetworkNamespaceParamName: testNamespace,
				config.SourceNodeNameParamName:   testSourceNodeName,
				config.TargetNodeNameParamName:   "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{Params: testCase.params}
			_, err := config.New(baseConfig)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigShouldFailWhenIntegerParamsAreInvalid(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "sample duration is not valid integer",
			expectedError: strconv.ErrSyntax,
			params: map[string]string{
				config.NetworkNameParamName:           testNetAttachDefName,
				config.NetworkNamespaceParamName:      testNamespace,
				config.SampleDurationSecondsParamName: "3rr0r",
			},
		},
		{
			description:   "desired max latency is too big",
			expectedError: strconv.ErrRange,
			params: map[string]string{
				config.NetworkNameParamName:                   testNetAttachDefName,
				config.NetworkNamespaceParamName:              testNamespace,
				config.DesiredMaxLatencyMillisecondsParamName: "39213801928309128309",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			baseConfig := kconfig.Config{Params: testCase.params}
			_, err := config.New(baseConfig)
			assert.ErrorContains(t, err, testCase.expectedError.Error())
		})
	}
}
