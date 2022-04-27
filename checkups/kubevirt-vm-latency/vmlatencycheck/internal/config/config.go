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

type Config struct {
	ResultsConfigMapName          string
	ResultsConfigMapNamespace     string
	NetworkName                   string
	NetworkNamespace              string
	TargetNodeName                string
	SourceNodeName                string
	SampleDurationSeconds         int
	DesiredMaxLatencyMilliseconds int
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

func NewFromEnv(env map[string]string) (*Config, error) {
	if env == nil {
		return nil, ErrInvalidEnv
	}

	resultsConfigMapName, exists := env[ResultsConfigMapNameEnvVarName]
	if !exists {
		return nil, ErrResultsConfigMapNameMissing
	}
	if resultsConfigMapName == "" {
		return nil, ErrInvalidResultsConfigMapName
	}

	resultsConfigMapNamespace, exists := env[ResultsConfigMapNamespaceEnvVarName]
	if !exists {
		return nil, ErrResultsConfigMapNamespaceMissing
	}
	if resultsConfigMapNamespace == "" {
		return nil, ErrInvalidResultsConfigMapNamespace
	}

	networkName, exists := env[NetworkNameEnvVarName]
	if !exists {
		return nil, ErrNetworkNameMissing
	}
	if networkName == "" {
		return nil, ErrInvalidNetworkName
	}

	networkNamespace, exists := env[NetworkNamespaceEnvVarName]
	if !exists {
		return nil, ErrNetworkNamespaceMissing
	}
	if networkNamespace == "" {
		return nil, ErrInvalidNetworkNamespace
	}

	sampleDuration, err := strconv.Atoi(env[SampleDurationSecondsEnvVarName])
	if err != nil {
		sampleDuration = DefaultSampleDurationSeconds
	}

	desiredMaxLatencyMilliseconds, err := strconv.Atoi(env[DesiredMaxLatencyMillisecondsEnvVarName])
	if err != nil {
		desiredMaxLatencyMilliseconds = DefaultDesiredMaxLatencyMilliseconds
	}

	return &Config{
		ResultsConfigMapName:          resultsConfigMapName,
		ResultsConfigMapNamespace:     resultsConfigMapNamespace,
		NetworkName:                   networkName,
		NetworkNamespace:              networkNamespace,
		TargetNodeName:                env[TargetNodeNameEnvVarName],
		SourceNodeName:                env[SourceNodeNameEnvVarName],
		SampleDurationSeconds:         sampleDuration,
		DesiredMaxLatencyMilliseconds: desiredMaxLatencyMilliseconds,
	}, nil
}
