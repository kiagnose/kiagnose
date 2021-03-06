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

package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

const (
	ResultsConfigMapNamespaceEnvVarName     = "RESULT_CONFIGMAP_NAMESPACE"
	ResultsConfigMapNameEnvVarName          = "RESULT_CONFIGMAP_NAME"
	NetworkNamespaceEnvVarName              = "NETWORK_ATTACHMENT_DEFINITION_NAMESPACE"
	NetworkNameEnvVarName                   = "NETWORK_ATTACHMENT_DEFINITION_NAME"
	SampleDurationSecondsEnvVarName         = "SAMPLE_DURATION_SECONDS"
	SourceNodeNameEnvVarName                = "SOURCE_NODE"
	TargetNodeNameEnvVarName                = "TARGET_NODE"
	DesiredMaxLatencyMillisecondsEnvVarName = "MAX_DESIRED_LATENCY_MILLISECONDS"
)

type CheckupParameters struct {
	NetworkAttachmentDefinitionName      string
	NetworkAttachmentDefinitionNamespace string
	TargetNodeName                       string
	SourceNodeName                       string
	SampleDurationSeconds                int
	DesiredMaxLatencyMilliseconds        int
}

type Config struct {
	ResultsConfigMapName      string
	ResultsConfigMapNamespace string
	CheckupParameters
}

var (
	ErrInvalidEnv                       = errors.New("environment is invalid")
	ErrResultsConfigMapNameMissing      = fmt.Errorf("%q environment variable is missing", ResultsConfigMapNameEnvVarName)
	ErrInvalidResultsConfigMapName      = fmt.Errorf("%q environment variable is invalid", ResultsConfigMapNameEnvVarName)
	ErrResultsConfigMapNamespaceMissing = fmt.Errorf("%q environment variable is missing", ResultsConfigMapNamespaceEnvVarName)
	ErrInvalidResultsConfigMapNamespace = fmt.Errorf("%q environment variable is invalid", ResultsConfigMapNamespaceEnvVarName)
	ErrNetworkNameMissing               = fmt.Errorf("%q environment variable is missing", NetworkNameEnvVarName)
	ErrInvalidNetworkName               = fmt.Errorf("%q environment variable is invalid", NetworkNameEnvVarName)
	ErrNetworkNamespaceMissing          = fmt.Errorf("%q environment variable is missing", NetworkNamespaceEnvVarName)
	ErrInvalidNetworkNamespace          = fmt.Errorf("%q environment variable is invalid", NetworkNamespaceEnvVarName)
)

const (
	DefaultSampleDurationSeconds         = 5
	DefaultDesiredMaxLatencyMilliseconds = math.MaxInt
)

func New(env map[string]string) (Config, error) {
	if len(env) == 0 {
		return Config{}, ErrInvalidEnv
	}

	resultsConfigMapName, exists := env[ResultsConfigMapNameEnvVarName]
	if !exists {
		return Config{}, ErrResultsConfigMapNameMissing
	}
	if resultsConfigMapName == "" {
		return Config{}, ErrInvalidResultsConfigMapName
	}

	resultsConfigMapNamespace, exists := env[ResultsConfigMapNamespaceEnvVarName]
	if !exists {
		return Config{}, ErrResultsConfigMapNamespaceMissing
	}
	if resultsConfigMapNamespace == "" {
		return Config{}, ErrInvalidResultsConfigMapNamespace
	}

	networkName, exists := env[NetworkNameEnvVarName]
	if !exists {
		return Config{}, ErrNetworkNameMissing
	}
	if networkName == "" {
		return Config{}, ErrInvalidNetworkName
	}

	networkNamespace, exists := env[NetworkNamespaceEnvVarName]
	if !exists {
		return Config{}, ErrNetworkNamespaceMissing
	}
	if networkNamespace == "" {
		return Config{}, ErrInvalidNetworkNamespace
	}

	var err error
	sampleDuration := DefaultSampleDurationSeconds
	if value, exists := env[SampleDurationSecondsEnvVarName]; exists {
		if sampleDuration, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q environment variable is invalid: %v", SampleDurationSecondsEnvVarName, err)
		}
	}

	desiredMaxLatency := DefaultDesiredMaxLatencyMilliseconds
	if value, exists := env[DesiredMaxLatencyMillisecondsEnvVarName]; exists {
		if desiredMaxLatency, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q environment variable is invalid: %v", DesiredMaxLatencyMillisecondsEnvVarName, err)
		}
	}

	return Config{
		ResultsConfigMapName:      resultsConfigMapName,
		ResultsConfigMapNamespace: resultsConfigMapNamespace,
		CheckupParameters: CheckupParameters{
			NetworkAttachmentDefinitionName:      networkName,
			NetworkAttachmentDefinitionNamespace: networkNamespace,
			TargetNodeName:                       env[TargetNodeNameEnvVarName],
			SourceNodeName:                       env[SourceNodeNameEnvVarName],
			SampleDurationSeconds:                sampleDuration,
			DesiredMaxLatencyMilliseconds:        desiredMaxLatency,
		},
	}, nil
}
