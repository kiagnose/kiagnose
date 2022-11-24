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

import "fmt"

type Environment struct {
	ConfigMapNamespace string
	ConfigMapName      string
}

const (
	ConfigMapNamespaceEnvVarName = "CONFIGMAP_NAMESPACE"
	ConfigMapNameEnvVarName      = "CONFIGMAP_NAME"
)

var (
	ErrMissingConfigMapNamespace = fmt.Errorf("missing required environment variable: %q", ConfigMapNamespaceEnvVarName)
	ErrMissingConfigMapName      = fmt.Errorf("missing required environment variable: %q", ConfigMapNameEnvVarName)
)

func NewEnvironment(rawEnv map[string]string) Environment {
	return Environment{
		ConfigMapNamespace: rawEnv[ConfigMapNamespaceEnvVarName],
		ConfigMapName:      rawEnv[ConfigMapNameEnvVarName],
	}
}

func (e Environment) Validate() error {
	if e.ConfigMapNamespace == "" {
		return ErrMissingConfigMapNamespace
	}

	if e.ConfigMapName == "" {
		return ErrMissingConfigMapName
	}

	return nil
}
