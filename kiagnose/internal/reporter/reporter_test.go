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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
	"github.com/kiagnose/kiagnose/kiagnose/internal/reporter"
)

const (
	configMapNamespace = "kiagnose"
	configMapName      = "checkup1"

	testImageValue   = "mycheckup:v0.1.0"
	testTimeoutValue = "1m"

	startTimestamp      = 1650882937
	completionTimestamp = 1651063860
)

var checkupRawSpec = map[string]string{config.ImageKey: testImageValue, config.TimeoutKey: testTimeoutValue}

func TestCheckupSuccessScenario(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(newConfigMap(checkupRawSpec))

	reportee := &reporteeStub{}
	expectedReport := map[string]string{}

	reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName, reportee)

	t.Run("Check report after Setup() success", func(t *testing.T) {
		reportee.failureReason = ""
		reportee.startTimestamp = time.Unix(startTimestamp, 0)

		expectedReport[reporter.StartTimestampKey] = reportee.startTimestamp.Format(time.RFC3339)
		expectedReport[reporter.FailureReasonKey] = reportee.failureReason

		assert.NoError(t, reporterUnderTest.Report())
		assertReport(t, fakeClient, checkupRawSpec, expectedReport)
	})

	t.Run("check report after Run() and Teardown() success", func(t *testing.T) {
		reportee.succeeded = strconv.FormatBool(true)
		reportee.failureReason = ""
		reportee.completionTimestamp = time.Unix(completionTimestamp, 0)

		expectedReport[reporter.SucceededKey] = reportee.succeeded
		expectedReport[reporter.FailureReasonKey] = reportee.failureReason
		expectedReport[reporter.CompletionTimestampKey] = reportee.completionTimestamp.Format(time.RFC3339)

		assert.NoError(t, reporterUnderTest.Report())
		assertReport(t, fakeClient, checkupRawSpec, expectedReport)
	})
}

func TestSetupFailure(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(newConfigMap(checkupRawSpec))

	reportee := &reporteeStub{}
	expectedReport := map[string]string{}

	reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName, reportee)

	t.Run("Check report after Setup() failure", func(t *testing.T) {
		reportee.succeeded = strconv.FormatBool(false)
		reportee.failureReason = "setup: some error"
		reportee.startTimestamp = time.Unix(startTimestamp, 0)

		expectedReport[reporter.SucceededKey] = reportee.succeeded
		expectedReport[reporter.FailureReasonKey] = reportee.failureReason
		expectedReport[reporter.StartTimestampKey] = reportee.startTimestamp.Format(time.RFC3339)

		assert.NoError(t, reporterUnderTest.Report())
		assertReport(t, fakeClient, checkupRawSpec, expectedReport)
	})

	t.Run("Check report after Teardown() success", func(t *testing.T) {
		reportee.completionTimestamp = time.Unix(completionTimestamp, 0)

		expectedReport[reporter.CompletionTimestampKey] = reportee.completionTimestamp.Format(time.RFC3339)

		assert.NoError(t, reporterUnderTest.Report())
		assertReport(t, fakeClient, checkupRawSpec, expectedReport)
	})
}

func TestTeardownFailure(t *testing.T) {
	type failureTestCase struct {
		description   string
		failureReason string
	}

	testCases := []failureTestCase{
		{
			description:   "Check report after Run() failure and Teardown() success",
			failureReason: "run: some error",
		},
		{
			description:   "Check report after Teardown() failure",
			failureReason: "teardown: some error",
		},
	}

	for _, testCase := range testCases {
		fakeClient := fake.NewSimpleClientset(newConfigMap(checkupRawSpec))

		reportee := &reporteeStub{
			startTimestamp: time.Unix(startTimestamp, 0),
		}
		expectedReport := map[string]string{
			reporter.StartTimestampKey: reportee.startTimestamp.Format(time.RFC3339),
		}

		reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName, reportee)

		t.Run(testCase.description, func(t *testing.T) {
			reportee.succeeded = strconv.FormatBool(false)
			reportee.failureReason = testCase.failureReason
			reportee.completionTimestamp = time.Unix(completionTimestamp, 0)

			expectedReport[reporter.SucceededKey] = reportee.succeeded
			expectedReport[reporter.FailureReasonKey] = reportee.failureReason
			expectedReport[reporter.CompletionTimestampKey] = reportee.completionTimestamp.Format(time.RFC3339)

			assert.NoError(t, reporterUnderTest.Report())
			assertReport(t, fakeClient, checkupRawSpec, expectedReport)
		})
	}
}

func TestReportFailure(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(newConfigMap(nil))

	reportee := &reporteeStub{}

	reporterUnderTest := reporter.New(fakeClient, configMapNamespace, configMapName, reportee)
	assert.ErrorIs(t, reporterUnderTest.Report(), reporter.ErrConfigMapDataIsNil)
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

func concatenateMaps(map1, map2 map[string]string) map[string]string {
	resultMap := map[string]string{}

	for k, v := range map1 {
		resultMap[k] = v
	}

	for k, v := range map2 {
		resultMap[k] = v
	}

	return resultMap
}

func assertReport(t *testing.T, fakeClient kubernetes.Interface, checkupRawSpec, expectedReport map[string]string) {
	configMap, err := fakeClient.CoreV1().ConfigMaps(configMapNamespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	assert.NoError(t, err)

	expectedData := concatenateMaps(checkupRawSpec, expectedReport)
	assert.Equal(t, expectedData, configMap.Data)
}

type reporteeStub struct {
	succeeded           string
	failureReason       string
	startTimestamp      time.Time
	completionTimestamp time.Time
}

func (rs *reporteeStub) Succeeded() string {
	return rs.succeeded
}

func (rs *reporteeStub) FailureReason() string {
	return rs.failureReason
}

func (rs *reporteeStub) StartTimestamp() time.Time {
	return rs.startTimestamp
}

func (rs *reporteeStub) CompletionTimestamp() time.Time {
	return rs.completionTimestamp
}
