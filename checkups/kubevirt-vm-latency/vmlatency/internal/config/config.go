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
	NetworkNamespaceParamName              = "NETWORK_ATTACHMENT_DEFINITION_NAMESPACE"
	NetworkNameParamName                   = "NETWORK_ATTACHMENT_DEFINITION_NAME"
	SampleDurationSecondsParamName         = "SAMPLE_DURATION_SECONDS"
	SourceNodeNameParamName                = "SOURCE_NODE"
	TargetNodeNameParamName                = "TARGET_NODE"
	DesiredMaxLatencyMillisecondsParamName = "MAX_DESIRED_LATENCY_MILLISECONDS"
)

type Config struct {
	NetworkAttachmentDefinitionName      string
	NetworkAttachmentDefinitionNamespace string
	TargetNodeName                       string
	SourceNodeName                       string
	SampleDurationSeconds                int
	DesiredMaxLatencyMilliseconds        int
}

var (
	ErrInvalidParams                          = errors.New("params is invalid")
	ErrInvalidNetworkName                     = fmt.Errorf("%q parameter is invalid", NetworkNameParamName)
	ErrInvalidNetworkNamespace                = fmt.Errorf("%q parameter is invalid", NetworkNamespaceParamName)
	ErrIllegalSourceAndTargetNodesCombination = errors.New("illegal source and target nodes combination")
)

const (
	DefaultSampleDurationSeconds         = 5
	DefaultDesiredMaxLatencyMilliseconds = math.MaxInt
)

func New(params map[string]string) (Config, error) {
	if len(params) == 0 {
		return Config{}, ErrInvalidParams
	}

	newConfig := Config{
		NetworkAttachmentDefinitionName:      params[NetworkNameParamName],
		NetworkAttachmentDefinitionNamespace: params[NetworkNamespaceParamName],
		TargetNodeName:                       params[TargetNodeNameParamName],
		SourceNodeName:                       params[SourceNodeNameParamName],
	}

	var err error
	sampleDuration := DefaultSampleDurationSeconds
	if value, exists := params[SampleDurationSecondsParamName]; exists {
		if sampleDuration, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q parameter is invalid: %v", SampleDurationSecondsParamName, err)
		}
	}
	newConfig.SampleDurationSeconds = sampleDuration

	desiredMaxLatency := DefaultDesiredMaxLatencyMilliseconds
	if value, exists := params[DesiredMaxLatencyMillisecondsParamName]; exists {
		if desiredMaxLatency, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q parameter is invalid: %v", DesiredMaxLatencyMillisecondsParamName, err)
		}
	}
	newConfig.DesiredMaxLatencyMilliseconds = desiredMaxLatency

	if err := newConfig.validate(); err != nil {
		return Config{}, err
	}

	return newConfig, nil
}

func (c Config) validate() error {
	if c.NetworkAttachmentDefinitionName == "" {
		return ErrInvalidNetworkName
	}

	if c.NetworkAttachmentDefinitionNamespace == "" {
		return ErrInvalidNetworkNamespace
	}

	if c.SourceNodeName == "" && c.TargetNodeName != "" || c.SourceNodeName != "" && c.TargetNodeName == "" {
		return ErrIllegalSourceAndTargetNodesCombination
	}

	return nil
}
