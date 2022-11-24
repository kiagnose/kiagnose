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
	"time"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kiagnose/kiagnose/kiagnose/config"
	"github.com/kiagnose/kiagnose/kiagnose/types"
)

const (
	configMapNamespace = "target-ns"
	configMapName      = "cm1"
	configMapUID       = "0123456789"

	timeoutValue = "1m"
	param1Key    = "message1"
	param1Value  = "message1 value"
	param2Key    = "message2"
	param2Value  = "message2 value"
)

var validRawEnv = map[string]string{
	config.ConfigMapNamespaceEnvVarName: configMapNamespace,
	config.ConfigMapNameEnvVarName:      configMapName,
}

func TestReadShouldSucceed(t *testing.T) {
	type readTestCase struct {
		description    string
		rawEnv         map[string]string
		configMapData  map[string]string
		expectedConfig config.Config
	}

	testCases := []readTestCase{
		{
			description: "when supplied with required parameters only",
			rawEnv:      validRawEnv,
			configMapData: map[string]string{
				types.TimeoutKey: timeoutValue,
			},
			expectedConfig: config.Config{
				ConfigMapNamespace: configMapNamespace,
				ConfigMapName:      configMapName,
				UID:                configMapUID,
				Timeout:            stringToDurationMustParse(timeoutValue),
				Params:             map[string]string{},
			},
		},
		{
			description: "when supplied with all parameters",
			rawEnv:      validRawEnv,
			configMapData: map[string]string{
				types.TimeoutKey:                     timeoutValue,
				types.ParamNameKeyPrefix + param1Key: param1Value,
				types.ParamNameKeyPrefix + param2Key: param2Value,
			},
			expectedConfig: config.Config{
				ConfigMapNamespace: configMapNamespace,
				ConfigMapName:      configMapName,
				UID:                configMapUID,
				Timeout:            stringToDurationMustParse(timeoutValue),
				Params: map[string]string{
					param1Key: param1Value,
					param2Key: param2Value,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(newConfigMap(configMapNamespace, configMapName, testCase.configMapData))

			actualConfig, err := config.Read(fakeClient, testCase.rawEnv)
			assert.NoError(t, err)

			assert.Equal(t, testCase.expectedConfig, actualConfig)
		})
	}
}

func TestEnvironmentReadShouldFail(t *testing.T) {
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
			fakeClient := fake.NewSimpleClientset()

			_, err := config.Read(fakeClient, testCase.rawEnv)
			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestConfigMapReadShouldFail(t *testing.T) {
	t.Run("when ConfigMap doesn't exist", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		_, err := config.Read(fakeClient, validRawEnv)
		assert.ErrorContains(t, err, "not found")
	})

	type readFailureTestCase struct {
		description   string
		rawEnv        map[string]string
		configMapData map[string]string
		expectedError string
	}

	const emptyParamName = ""

	failureTestCases := []readFailureTestCase{
		{
			description: "when ConfigMap is already in use",
			rawEnv:      validRawEnv,
			configMapData: map[string]string{
				types.TimeoutKey:        timeoutValue,
				types.StartTimestampKey: time.Now().Format(time.RFC3339),
			},
			expectedError: config.ErrConfigMapIsAlreadyInUse.Error(),
		},
		{
			description: "when ConfigMap is already in use (startTimestamp exists but empty)",
			rawEnv:      validRawEnv,
			configMapData: map[string]string{
				types.TimeoutKey:        timeoutValue,
				types.StartTimestampKey: "",
			},
			expectedError: config.ErrConfigMapIsAlreadyInUse.Error(),
		},
		{
			description:   "when timout field is missing",
			rawEnv:        validRawEnv,
			configMapData: map[string]string{},
			expectedError: config.ErrTimeoutFieldIsMissing.Error(),
		},
		{
			description: "when timout field is illegal",
			rawEnv:      validRawEnv,
			configMapData: map[string]string{
				types.TimeoutKey: "illegalValue",
			},
			expectedError: config.ErrTimeoutFieldIsIllegal.Error(),
		},
		{
			description: "when ConfigMap Data is nil", rawEnv: validRawEnv, configMapData: nil, expectedError: config.ErrConfigMapDataIsNil.Error()},
		{
			description: "when param name is empty",
			rawEnv:      validRawEnv,
			configMapData: map[string]string{
				types.TimeoutKey: timeoutValue,
				types.ParamNameKeyPrefix + emptyParamName: "some value",
			},
			expectedError: config.ErrParamNameIsIllegal.Error(),
		},
	}

	for _, testCase := range failureTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(newConfigMap(configMapNamespace, configMapName, testCase.configMapData))

			_, err := config.Read(fakeClient, testCase.rawEnv)
			assert.ErrorContains(t, err, testCase.expectedError)
		})
	}
}

func newConfigMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       configMapUID,
		},
		Data: data,
	}
}

func stringToDurationMustParse(rawDuration string) time.Duration {
	duration, err := time.ParseDuration(rawDuration)
	if err != nil {
		panic("Bad duration")
	}

	return duration
}
