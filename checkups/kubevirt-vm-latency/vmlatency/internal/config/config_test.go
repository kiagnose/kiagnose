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

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
)

type configCreateTestCases struct {
	description    string
	env            map[string]string
	expectedConfig config.Config
}

const (
	testNamespace                     = "default"
	testResultConfigMapName           = "results"
	testNetAttachDefName              = "blue-net"
	testDesiredMaxLatencyMilliseconds = 100
	testSampleDurationSeconds         = 60
	testSourceNodeName                = "worker1"
	testTargetNodeName                = "worker2"
)

func TestCreateConfigFromEnvShould(t *testing.T) {
	testCases := []configCreateTestCases{
		{
			description: "set default sample duration when env var is missing",
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:          testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName:     testNamespace,
				config.NetworkNameEnvVarName:                   testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:              testNamespace,
				config.DesiredMaxLatencyMillisecondsEnvVarName: fmt.Sprintf("%d", testDesiredMaxLatencyMilliseconds),
			},
			expectedConfig: config.Config{
				CheckupParameters: config.CheckupParameters{
					SampleDurationSeconds:                config.DefaultSampleDurationSeconds,
					NetworkAttachmentDefinitionName:      testNetAttachDefName,
					NetworkAttachmentDefinitionNamespace: testNamespace,
					DesiredMaxLatencyMilliseconds:        testDesiredMaxLatencyMilliseconds,
				},
				ResultsConfigMapName:      testResultConfigMapName,
				ResultsConfigMapNamespace: testNamespace,
			},
		},
		{
			description: "set default desired max latency when env var is missing",
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SampleDurationSecondsEnvVarName:     fmt.Sprintf("%d", testSampleDurationSeconds),
			},
			expectedConfig: config.Config{
				CheckupParameters: config.CheckupParameters{
					DesiredMaxLatencyMilliseconds:        config.DefaultDesiredMaxLatencyMilliseconds,
					NetworkAttachmentDefinitionName:      testNetAttachDefName,
					NetworkAttachmentDefinitionNamespace: testNamespace,
					SampleDurationSeconds:                testSampleDurationSeconds,
				},
				ResultsConfigMapName:      testResultConfigMapName,
				ResultsConfigMapNamespace: testNamespace,
			},
		},
		{
			description: "set source and target nodes when both are specified",
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SampleDurationSecondsEnvVarName:     fmt.Sprintf("%d", testSampleDurationSeconds),
				config.SourceNodeNameEnvVarName:            testSourceNodeName,
				config.TargetNodeNameEnvVarName:            testTargetNodeName,
			},
			expectedConfig: config.Config{
				CheckupParameters: config.CheckupParameters{
					DesiredMaxLatencyMilliseconds:        config.DefaultDesiredMaxLatencyMilliseconds,
					NetworkAttachmentDefinitionName:      testNetAttachDefName,
					NetworkAttachmentDefinitionNamespace: testNamespace,
					SampleDurationSeconds:                testSampleDurationSeconds,
					SourceNodeName:                       testSourceNodeName,
					TargetNodeName:                       testTargetNodeName,
				},
				ResultsConfigMapName:      testResultConfigMapName,
				ResultsConfigMapNamespace: testNamespace,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testConfig, err := config.New(testCase.env)
			assert.NoError(t, err)
			assert.Equal(t, testConfig, testCase.expectedConfig)
		})
	}
}

type configCreateFallingTestCases struct {
	description   string
	expectedError error
	env           map[string]string
}

func TestCreateConfigFromEnvShouldFailWhen(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "env is nil",
			expectedError: config.ErrInvalidEnv,
			env:           nil,
		},
		{
			description:   "env is empty",
			expectedError: config.ErrInvalidEnv,
			env:           map[string]string{},
		},
		{
			description:   "results ConfigMap name env var value is not valid",
			expectedError: config.ErrInvalidResultsConfigMapName,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "",
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
			},
		},
		{
			description:   "results ConfigMap namespace env var value is not valid",
			expectedError: config.ErrInvalidResultsConfigMapNamespace,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: "",
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
			},
		},
		{
			description:   "network name env var value is not valid",
			expectedError: config.ErrInvalidNetworkName,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               "",
				config.NetworkNamespaceEnvVarName:          testNamespace,
			},
		},
		{
			description:   "network namespace env var value is not valid",
			expectedError: config.ErrInvalidNetworkNamespace,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.New(testCase.env)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigFromEnvShouldFailWhenMandatoryEnvVarsAreMissing(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "results ConfigMap name env var is missing",
			expectedError: config.ErrResultsConfigMapNameMissing,
			env: map[string]string{
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
			},
		},
		{
			description:   "results ConfigMap namespace env var is missing",
			expectedError: config.ErrResultsConfigMapNamespaceMissing,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName: testResultConfigMapName,
				config.NetworkNameEnvVarName:          testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:     testNamespace,
			},
		},
		{
			description:   "network name env var is missing",
			expectedError: config.ErrNetworkNameMissing,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNamespaceEnvVarName:          testNamespace,
			},
		},
		{
			description:   "network namespace env var is missing",
			expectedError: config.ErrNetworkNamespaceMissing,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.New(testCase.env)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigFromEnvShouldFailWhenNodeNames(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "source node name is set but target node name isn't",
			expectedError: config.ErrTargetNodeNameMissing,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SourceNodeNameEnvVarName:            testSourceNodeName,
			},
		},
		{
			description:   "target node name is set but source node name isn't",
			expectedError: config.ErrSourceNodeNameMissing,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.TargetNodeNameEnvVarName:            testTargetNodeName,
			},
		},
		{
			description:   "source node name is empty",
			expectedError: config.ErrInvalidSourceNodeName,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SourceNodeNameEnvVarName:            "",
				config.TargetNodeNameEnvVarName:            testTargetNodeName,
			},
		},
		{
			description:   "target node name is empty",
			expectedError: config.ErrInvalidTargetNodeName,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SourceNodeNameEnvVarName:            testSourceNodeName,
				config.TargetNodeNameEnvVarName:            "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.New(testCase.env)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigShouldFailWhenIntegerEnvVarsAreInvalid(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description:   "sample duration is not valid integer",
			expectedError: strconv.ErrSyntax,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SampleDurationSecondsEnvVarName:     "3rr0r",
			},
		},
		{
			description:   "desired max latency is too big",
			expectedError: strconv.ErrRange,
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:          testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName:     testNamespace,
				config.NetworkNameEnvVarName:                   testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:              testNamespace,
				config.DesiredMaxLatencyMillisecondsEnvVarName: "39213801928309128309",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.New(testCase.env)
			assert.ErrorContains(t, err, testCase.expectedError.Error())
		})
	}
}
