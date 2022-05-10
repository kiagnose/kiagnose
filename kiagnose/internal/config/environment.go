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
	"fmt"

	"k8s.io/apimachinery/pkg/types"
)

const (
	CheckupNamespaceEnvVarName = "CHECKUP_NAMESPACE"
	CheckupNameEnvVarName      = "CHECKUP_NAME"
)

func CheckupKeyFromEnv(env map[string]string) (key types.NamespacedName, err error) {
	const envVarErr = "missing required environment variable"

	var exists bool
	key.Namespace, exists = env[CheckupNamespaceEnvVarName]
	if !exists {
		return types.NamespacedName{}, fmt.Errorf("%s: %q", envVarErr, CheckupNamespaceEnvVarName)
	}

	key.Name, exists = env[CheckupNameEnvVarName]
	if !exists {
		return types.NamespacedName{}, fmt.Errorf("%s: %q", envVarErr, CheckupNameEnvVarName)
	}

	return key, nil
}
