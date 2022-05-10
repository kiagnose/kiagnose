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

	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
)

func TestCheckupKeyFromEnvShouldSucceed(t *testing.T) {
	goodEnv := map[string]string{
		config.CheckupNamespaceEnvVarName: checkupKey.Namespace,
		config.CheckupNameEnvVarName:      checkupKey.Name,
	}

	obtainedKey, err := config.CheckupKeyFromEnv(goodEnv)
	assert.NoError(t, err)
	assert.Equal(t, obtainedKey, checkupKey)
}

func TestCheckupKeyFromEnvShouldFail(t *testing.T) {
	type envVarsLoadingFailureTestCase struct {
		description           string
		envVars               map[string]string
		expectedErrorContains string
	}

	const expectedErrorPrefix = "missing required environment variable"

	failureTestCases := []envVarsLoadingFailureTestCase{
		{
			description:           "when Checkup's name environment variable is missing",
			envVars:               map[string]string{config.CheckupNamespaceEnvVarName: checkupKey.Namespace},
			expectedErrorContains: expectedErrorPrefix,
		},
		{
			description:           "when Checkup's namespace environment variable is missing",
			envVars:               map[string]string{config.CheckupNameEnvVarName: checkupKey.Name},
			expectedErrorContains: expectedErrorPrefix,
		},
		{
			description:           "when both Checkup's environment variables are missing",
			envVars:               map[string]string{},
			expectedErrorContains: expectedErrorPrefix,
		},
	}

	for _, testCase := range failureTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.CheckupKeyFromEnv(testCase.envVars)
			assert.ErrorContains(t, err, testCase.expectedErrorContains)
		})
	}
}
