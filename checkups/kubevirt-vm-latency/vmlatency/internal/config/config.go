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

	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"
)

const (
	NetworkNamespaceParamName              = "network_attachment_definition_namespace"
	NetworkNameParamName                   = "network_attachment_definition_name"
	SampleDurationSecondsParamName         = "sample_duration_seconds"
	SourceNodeNameParamName                = "source_node"
	TargetNodeNameParamName                = "target_node"
	DesiredMaxLatencyMillisecondsParamName = "max_desired_latency_milliseconds"
)

type Config struct {
	PodName                              string
	PodUID                               string
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

func New(baseConfig kconfig.Config) (Config, error) {
	if len(baseConfig.Params) == 0 {
		return Config{}, ErrInvalidParams
	}

	newConfig := Config{
		PodName:                              baseConfig.PodName,
		PodUID:                               baseConfig.PodUID,
		NetworkAttachmentDefinitionName:      baseConfig.Params[NetworkNameParamName],
		NetworkAttachmentDefinitionNamespace: baseConfig.Params[NetworkNamespaceParamName],
		TargetNodeName:                       baseConfig.Params[TargetNodeNameParamName],
		SourceNodeName:                       baseConfig.Params[SourceNodeNameParamName],
	}

	var err error
	sampleDuration := DefaultSampleDurationSeconds
	if value, exists := baseConfig.Params[SampleDurationSecondsParamName]; exists {
		if sampleDuration, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q parameter is invalid: %v", SampleDurationSecondsParamName, err)
		}
	}
	newConfig.SampleDurationSeconds = sampleDuration

	desiredMaxLatency := DefaultDesiredMaxLatencyMilliseconds
	if value, exists := baseConfig.Params[DesiredMaxLatencyMillisecondsParamName]; exists {
		if desiredMaxLatency, err = strconv.Atoi(value); err != nil {
			return Config{}, fmt.Errorf("%q parameter is invalid: %v", DesiredMaxLatencyMillisecondsParamName, err)
		}
	}
	newConfig.DesiredMaxLatencyMilliseconds = desiredMaxLatency

	err = newConfig.validate()
	if err != nil {
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
