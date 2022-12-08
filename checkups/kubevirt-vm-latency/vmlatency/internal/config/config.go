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
	"log"
	"math"
	"strconv"

	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"
)

const (
	NetworkNamespaceParamName              = "networkAttachmentDefinitionNamespace"
	NetworkNameParamName                   = "networkAttachmentDefinitionName"
	SourceNodeNameParamName                = "sourceNode"
	TargetNodeNameParamName                = "targetNode"
	SampleDurationSecondsParamName         = "sampleDurationSeconds"
	DesiredMaxLatencyMillisecondsParamName = "maxDesiredLatencyMilliseconds"
)

// Deprecated
const (
	NetworkNamespaceDeprecatedParamName              = "network_attachment_definition_namespace"
	NetworkNameDeprecatedParamName                   = "network_attachment_definition_name"
	SampleDurationSecondsDeprecatedParamName         = "sample_duration_seconds"
	SourceNodeNameDeprecatedParamName                = "source_node"
	TargetNodeNameDeprecatedParamName                = "target_node"
	DesiredMaxLatencyMillisecondsDeprecatedParamName = "max_desired_latency_milliseconds"
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
		SampleDurationSeconds:                DefaultSampleDurationSeconds,
		DesiredMaxLatencyMilliseconds:        DefaultDesiredMaxLatencyMilliseconds,
		NetworkAttachmentDefinitionNamespace: readConfig(baseConfig.Params, NetworkNamespaceParamName, NetworkNamespaceDeprecatedParamName),
		NetworkAttachmentDefinitionName:      readConfig(baseConfig.Params, NetworkNameParamName, NetworkNameDeprecatedParamName),
		SourceNodeName:                       readConfig(baseConfig.Params, SourceNodeNameParamName, SourceNodeNameDeprecatedParamName),
		TargetNodeName:                       readConfig(baseConfig.Params, TargetNodeNameParamName, TargetNodeNameDeprecatedParamName),
	}

	var err error
	if v := readConfig(baseConfig.Params, SampleDurationSecondsParamName, SampleDurationSecondsDeprecatedParamName); v != "" {
		if newConfig.SampleDurationSeconds, err = strconv.Atoi(v); err != nil {
			return Config{}, fmt.Errorf("%q parameter is invalid: %v", SampleDurationSecondsDeprecatedParamName, err)
		}
	}

	if v := readConfig(baseConfig.Params, DesiredMaxLatencyMillisecondsParamName, DesiredMaxLatencyMillisecondsDeprecatedParamName); v != "" {
		if newConfig.DesiredMaxLatencyMilliseconds, err = strconv.Atoi(v); err != nil {
			return Config{}, fmt.Errorf("%q parameter is invalid: %v", DesiredMaxLatencyMillisecondsParamName, err)
		}
	}

	err = newConfig.validate()
	if err != nil {
		return Config{}, err
	}

	return newConfig, nil
}

func readConfig(config map[string]string, paramName, paramDeprecatedName string) string {
	if value, exists := config[paramName]; exists {
		return value
	} else if value, exists := config[paramDeprecatedName]; exists {
		log.Printf("warning: %q parameter is DEPRECATED, please use the new form: %q", paramDeprecatedName, paramName)
		return value
	}
	return ""
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
