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
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/kiagnose/config"
)

func TestConfigMapFullNameShouldSucceed(t *testing.T) {
	goodEnv := map[string]string{
		config.ConfigMapNamespaceEnvVarName: configMapNamespace,
		config.ConfigMapNameEnvVarName:      configMapName,
	}

	namespace, name, err := config.ConfigMapFullName(goodEnv)
	assert.NoError(t, err)
	assert.Equal(t, configMapNamespace, namespace)
	assert.Equal(t, configMapName, name)
}

func TestConfigMapFullNameShouldFail(t *testing.T) {
	type envVarsLoadingFailureTestCase struct {
		description           string
		envVars               map[string]string
		expectedErrorContains string
	}

	const expectedErrorPrefix = "missing required environment variable"

	failureTestCases := []envVarsLoadingFailureTestCase{
		{
			description:           "when ConfigMap's name environment variable is missing",
			envVars:               map[string]string{config.ConfigMapNamespaceEnvVarName: configMapNamespace},
			expectedErrorContains: expectedErrorPrefix,
		},
		{
			description:           "when ConfigMap's namespace environment variable is missing",
			envVars:               map[string]string{config.ConfigMapNameEnvVarName: configMapName},
			expectedErrorContains: expectedErrorPrefix,
		},
		{
			description:           "when both ConfigMap's environment variables are missing",
			envVars:               map[string]string{},
			expectedErrorContains: expectedErrorPrefix,
		},
	}

	for _, testCase := range failureTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, _, err := config.ConfigMapFullName(testCase.envVars)
			assert.ErrorContains(t, err, testCase.expectedErrorContains)
		})
	}
}
