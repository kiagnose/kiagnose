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

func TestEnvironmentValidationShouldSucceed(t *testing.T) {
	goodEnv := map[string]string{
		config.ConfigMapNamespaceEnvVarName: configMapNamespace,
		config.ConfigMapNameEnvVarName:      configMapName,
	}

	actualEnvironment := config.NewEnvironment(goodEnv)

	expectedEnvironment := config.Environment{
		ConfigMapNamespace: configMapNamespace,
		ConfigMapName:      configMapName,
	}

	assert.Equal(t, expectedEnvironment, actualEnvironment)

	assert.NoError(t, actualEnvironment.Validate())
}

func TestValidateEnvironmentShouldFail(t *testing.T) {
	type validationErrorTestCase struct {
		description   string
		rawEnv        map[string]string
		expectedError error
	}

	failureTestCases := []validationErrorTestCase{
		{
			description:   "when ConfigMap's name environment variable is missing",
			rawEnv:        map[string]string{config.ConfigMapNamespaceEnvVarName: configMapNamespace},
			expectedError: config.ErrMissingConfigMapName,
		},
		{
			description:   "when ConfigMap's namespace environment variable is missing",
			rawEnv:        map[string]string{config.ConfigMapNameEnvVarName: configMapName},
			expectedError: config.ErrMissingConfigMapNamespace,
		},
		{
			description:   "when both ConfigMap's environment variables are missing",
			rawEnv:        map[string]string{},
			expectedError: config.ErrMissingConfigMapNamespace,
		},
	}

	for _, testCase := range failureTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			environment := config.NewEnvironment(testCase.rawEnv)
			assert.ErrorIs(t,
				environment.Validate(),
				testCase.expectedError,
			)
		})
	}
}
