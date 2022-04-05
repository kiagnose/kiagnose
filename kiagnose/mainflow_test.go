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

package kiagnose

import (
	"os"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestReadConfigMapFullNameFromEnvShould(t *testing.T) {
	const (
		configMapNamespace = "kiagnose"
		configMapName      = "cm1"
	)

	t.Run("run successfully", func(t *testing.T) {
		os.Setenv(configMapNamespaceEnvVarName, configMapNamespace)
		os.Setenv(configMapNameEnvVarName, configMapName)
		defer os.Unsetenv(configMapNamespaceEnvVarName)
		defer os.Unsetenv(configMapNameEnvVarName)

		namespace, name, err := readConfigMapFullNameFromEnv()
		assert.NoError(t, err)
		assert.Equal(t, namespace, configMapNamespace)
		assert.Equal(t, name, configMapName)
	})

	t.Run("fail when ConfigMap's name environment variable is missing", func(t *testing.T) {
		os.Setenv(configMapNamespaceEnvVarName, configMapNamespace)
		defer os.Unsetenv(configMapNamespaceEnvVarName)

		namespace, name, err := readConfigMapFullNameFromEnv()
		assert.ErrorContains(t, err, "missing \"CONFIGMAP_NAME\" environment variable")
		assert.Empty(t, namespace)
		assert.Empty(t, name)
	})

	t.Run("fail when ConfigMap's namespace environment variable is missing", func(t *testing.T) {
		os.Setenv(configMapNameEnvVarName, configMapName)
		defer os.Unsetenv(configMapNameEnvVarName)

		namespace, name, err := readConfigMapFullNameFromEnv()
		assert.ErrorContains(t, err, "missing \"CONFIGMAP_NAMESPACE\" environment variable")
		assert.Empty(t, namespace)
		assert.Empty(t, name)
	})

	t.Run("fail when ConfigMap's namespace and name environment variables are missing", func(t *testing.T) {
		namespace, name, err := readConfigMapFullNameFromEnv()
		assert.ErrorContains(t, err, "missing \"CONFIGMAP_NAMESPACE\" environment variable")
		assert.Empty(t, namespace)
		assert.Empty(t, name)
	})
}
