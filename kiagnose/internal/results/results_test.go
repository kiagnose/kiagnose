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

package results_test

import (
	"strconv"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/kiagnose/internal/results"
)

const (
	configMapNamespace = "kiagnose"
	configMapName      = "checkup1"
)

func TestReadResultsShouldSucceedWhen(t *testing.T) {
	type successTestCase struct {
		description     string
		input           map[string]string
		expectedResults results.Results
	}

	const (
		emptyFailureReason = ""
		testFailureReason  = "some reason"
	)

	testCases := []successTestCase{
		{
			description: "checkup succeeds",
			input: map[string]string{
				results.SucceededKey:     strconv.FormatBool(true),
				results.FailureReasonKey: emptyFailureReason,
			},
			expectedResults: results.Results{
				Succeeded:     true,
				FailureReason: emptyFailureReason,
			},
		},
		{
			description: "checkup fails",
			input: map[string]string{
				results.SucceededKey:     strconv.FormatBool(false),
				results.FailureReasonKey: testFailureReason,
			},
			expectedResults: results.Results{
				Succeeded:     false,
				FailureReason: testFailureReason,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(newConfigMap(testCase.input))

			actualResults, err := results.ReadFromConfigMap(fakeClient.CoreV1(), configMapNamespace, configMapName)
			assert.NoError(t, err)

			assert.Equal(t, testCase.expectedResults, actualResults)
		})
	}
}

func TestReadResultsShouldFailWhen(t *testing.T) {
	t.Run("failing to fetch results", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		_, err := results.ReadFromConfigMap(fakeClient.CoreV1(), configMapNamespace, configMapName)

		assert.ErrorContains(t, err, "not found")
	})

	type failureTestCase struct {
		description   string
		input         map[string]string
		expectedError error
	}

	testCases := []failureTestCase{
		{
			description:   "results are fetched with nil Data",
			input:         nil,
			expectedError: results.ErrConfigMapDataIsNil,
		},
		{
			description:   "succeeded field is missing",
			input:         map[string]string{},
			expectedError: results.ErrSucceededFieldMissing,
		},
		{
			description: "succeeded field is illegal",
			input: map[string]string{
				results.SucceededKey: "not a boolean",
			},
			expectedError: results.ErrSucceededFieldIllegal,
		},
		{
			description: "failureReason field is missing",
			input: map[string]string{
				results.SucceededKey: strconv.FormatBool(false),
			},
			expectedError: results.ErrFailureReasonFieldMissing,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(newConfigMap(testCase.input))

			_, err := results.ReadFromConfigMap(fakeClient.CoreV1(), configMapNamespace, configMapName)

			assert.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func newConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		Data: data,
	}
}
