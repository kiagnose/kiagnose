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
	CheckupNameEnvVarName                   = "CHECKUP_NAME"
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
	CheckupName               string
	ResultsConfigMapName      string
	ResultsConfigMapNamespace string
	CheckupParameters
}

var (
	ErrInvalidEnv                             = errors.New("environment is invalid")
	ErrInvalidCheckupName                     = fmt.Errorf("%q environment variable is invalid", CheckupNameEnvVarName)
	ErrInvalidResultsConfigMapName            = fmt.Errorf("%q environment variable is invalid", ResultsConfigMapNameEnvVarName)
	ErrInvalidResultsConfigMapNamespace       = fmt.Errorf("%q environment variable is invalid", ResultsConfigMapNamespaceEnvVarName)
	ErrInvalidNetworkName                     = fmt.Errorf("%q environment variable is invalid", NetworkNameEnvVarName)
	ErrInvalidNetworkNamespace                = fmt.Errorf("%q environment variable is invalid", NetworkNamespaceEnvVarName)
	ErrIllegalSourceAndTargetNodesCombination = errors.New("illegal source and target nodes combination")
)

const (
	DefaultSampleDurationSeconds         = 5
	DefaultDesiredMaxLatencyMilliseconds = math.MaxInt
)

func New(env map[string]string) (Config, error) {
	if len(env) == 0 {
		return Config{}, ErrInvalidEnv
	}

	newConfig := Config{
		CheckupName:               env[CheckupNameEnvVarName],
		ResultsConfigMapName:      env[ResultsConfigMapNameEnvVarName],
		ResultsConfigMapNamespace: env[ResultsConfigMapNamespaceEnvVarName],
		CheckupParameters: CheckupParameters{
			NetworkAttachmentDefinitionName:      env[NetworkNameEnvVarName],
			NetworkAttachmentDefinitionNamespace: env[NetworkNamespaceEnvVarName],
			TargetNodeName:                       env[TargetNodeNameEnvVarName],
			SourceNodeName:                       env[SourceNodeNameEnvVarName],
		},
	}

	var err error
	sampleDuration := DefaultSampleDurationSeconds
	if value, exists := env[SampleDurationSecondsEnvVarName]; exists {
		if sampleDuration, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q environment variable is invalid: %v", SampleDurationSecondsEnvVarName, err)
		}
	}
	newConfig.SampleDurationSeconds = sampleDuration

	desiredMaxLatency := DefaultDesiredMaxLatencyMilliseconds
	if value, exists := env[DesiredMaxLatencyMillisecondsEnvVarName]; exists {
		if desiredMaxLatency, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q environment variable is invalid: %v", DesiredMaxLatencyMillisecondsEnvVarName, err)
		}
	}
	newConfig.DesiredMaxLatencyMilliseconds = desiredMaxLatency

	if err := newConfig.validate(); err != nil {
		return Config{}, err
	}

	return newConfig, nil
}

func (c Config) validate() error {
	if c.CheckupName == "" {
		return ErrInvalidCheckupName
	}

	if c.ResultsConfigMapName == "" {
		return ErrInvalidResultsConfigMapName
	}

	if c.ResultsConfigMapNamespace == "" {
		return ErrInvalidResultsConfigMapNamespace
	}

	if c.NetworkAttachmentDefinitionName == "" {
		return ErrInvalidNetworkName
	}

	if c.NetworkAttachmentDefinitionNamespace == "" {
		return ErrInvalidNetworkNamespace
	}

	if c.SourceNodeName == "" && c.TargetNodeName != "" {
		return ErrIllegalSourceAndTargetNodesCombination
	} else if c.SourceNodeName != "" && c.TargetNodeName == "" {
		return ErrIllegalSourceAndTargetNodesCombination
	}

	return nil
}
