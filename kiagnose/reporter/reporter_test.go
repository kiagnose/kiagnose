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

package reporter_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kiagnose/kiagnose/kiagnose/configmap"
	"github.com/kiagnose/kiagnose/kiagnose/reporter"
	"github.com/kiagnose/kiagnose/kiagnose/status"
	"github.com/kiagnose/kiagnose/kiagnose/types"
)

const (
	configMapNamespace = "kiagnose"
	configMapName      = "checkup1"
)

func TestReportShouldSucceed(t *testing.T) {
	type successTestCase struct {
		description   string
		succeeded     bool
		failureReason []string
	}

	var (
		fakeClient        *fake.Clientset
		reporterUnderTest *reporter.Reporter
		checkupStatus     status.Status
	)

	setup := func() {
		fakeClient = fake.NewSimpleClientset(newConfigMap(checkupSpecData()))
		reporterUnderTest = reporter.New(fakeClient, configMapNamespace, configMapName)

		checkupStatus = status.Status{StartTimestamp: time.Now()}
	}

	t.Run("on initial report", func(t *testing.T) {
		setup()

		assert.NoError(t, reporterUnderTest.Report(checkupStatus))

		expectedReportData := map[string]string{
			types.StartTimestampKey: timestamp(checkupStatus.StartTimestamp),
		}

		assert.Equal(t,
			mergeMaps(checkupSpecData(), expectedReportData),
			getCheckupData(t, fakeClient, configMapNamespace, configMapName),
		)
	})

	testCases := []successTestCase{
		{
			description:   "on checkup successful completion",
			succeeded:     true,
			failureReason: nil,
		},
		{
			description:   "on checkup failed completion",
			succeeded:     false,
			failureReason: []string{"some reason"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			setup()

			assert.NoError(t, reporterUnderTest.Report(checkupStatus))

			checkupStatus.Succeeded = testCase.succeeded
			checkupStatus.FailureReason = testCase.failureReason
			checkupStatus.CompletionTimestamp = checkupStatus.StartTimestamp.Add(time.Minute)

			assert.NoError(t, reporterUnderTest.Report(checkupStatus))

			expectedReportData := map[string]string{
				types.StartTimestampKey:      timestamp(checkupStatus.StartTimestamp),
				types.SucceededKey:           strconv.FormatBool(checkupStatus.Succeeded),
				types.FailureReasonKey:       strings.Join(checkupStatus.FailureReason, ","),
				types.CompletionTimestampKey: timestamp(checkupStatus.CompletionTimestamp),
			}

			assert.Equal(t,
				mergeMaps(checkupSpecData(), expectedReportData),
				getCheckupData(t, fakeClient, configMapNamespace, configMapName),
			)
		})
	}
}

func TestReportShouldFail(t *testing.T) {
	t.Run("when checkup spec is fetched with nil Data", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newConfigMap(nil))
		reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName)

		checkupStatus := status.Status{}

		assert.ErrorIs(t, reporterUnderTest.Report(checkupStatus), reporter.ErrConfigMapDataIsNil)
	})

	t.Run("when checkup spec fails to be fetched for the first time", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName)

		checkupStatus := status.Status{}

		assert.ErrorContains(t, reporterUnderTest.Report(checkupStatus), "not found")
	})

	t.Run("when checkup status fails to be updated", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newConfigMap(checkupSpecData()))
		reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName)

		checkupStatus := status.Status{}

		assert.NoError(t, reporterUnderTest.Report(checkupStatus))

		injectFailureToAccessCheckupData(t, fakeClient, configMapNamespace, configMapName)

		assert.ErrorContains(t, reporterUnderTest.Report(checkupStatus), "not found")
	})
}

func checkupSpecData() map[string]string {
	const (
		testImageValue   = "mycheckup:v0.1.0"
		testTimeoutValue = "1m"
	)

	return map[string]string{types.ImageKey: testImageValue, types.TimeoutKey: testTimeoutValue}
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

func mergeMaps(map1, map2 map[string]string) map[string]string {
	resultMap := map[string]string{}

	for k, v := range map1 {
		resultMap[k] = v
	}

	for k, v := range map2 {
		resultMap[k] = v
	}

	return resultMap
}

func timestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

func getCheckupData(t *testing.T, client kubernetes.Interface, configMapNamespace, configMapName string) map[string]string {
	configMap, err := configmap.Get(client, configMapNamespace, configMapName)
	assert.NoError(t, err)

	return configMap.Data
}

func injectFailureToAccessCheckupData(t *testing.T, client kubernetes.Interface, configMapNamespace, configMapName string) {
	assert.NoError(t, client.CoreV1().ConfigMaps(configMapNamespace).Delete(context.Background(), configMapName, metav1.DeleteOptions{}))
}
