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
	"fmt"
	"strconv"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kiagnose/kiagnose/kiagnose/configmap"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/reporter"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

const (
	testNamespace     = "default"
	testConfigMapName = "results"
)

func TestReportShouldRunSuccessfullyWhen(t *testing.T) {
	t.Run("status is initialized", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newConfigMap())

		testReporter := reporter.New(fakeClient, testNamespace, testConfigMapName)

		assert.NoError(t, testReporter.Report(status.Status{}))
	})
}

func TestReportShouldFailWhen(t *testing.T) {
	t.Run("failed to update results ConfigMap", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		testReporter := reporter.New(fakeClient, testNamespace, testConfigMapName)

		assert.ErrorContains(t, testReporter.Report(status.Status{}), "not found")
	})
}

func TestReportShouldSuccessfullyConvertResultValues(t *testing.T) {
	const (
		someFailureReason      = "some reason"
		someOtherFailureReason = "some other reason"
	)

	t.Run("on checkup successful completion", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newConfigMap())
		testReporter := reporter.New(fakeClient, testNamespace, testConfigMapName)

		var checkupStatus status.Status
		checkupStatus.StartTimestamp = time.Now()
		assert.NoError(t, testReporter.Report(checkupStatus))

		checkupStatus.FailureReason = []string{}
		checkupStatus.CompletionTimestamp = time.Now()
		checkupStatus.Results = status.Results{
			MinLatency:          1 * time.Minute,
			AvgLatency:          2 * time.Minute,
			MeasurementDuration: 3 * time.Minute,
			MaxLatency:          4 * time.Minute,
			TargetNode:          "a",
			SourceNode:          "b",
		}

		assert.NoError(t, testReporter.Report(checkupStatus))

		expectedReportData := map[string]string{
			"status.result.minLatencyNanoSec":      fmt.Sprint(checkupStatus.MinLatency.Nanoseconds()),
			"status.result.maxLatencyNanoSec":      fmt.Sprint(checkupStatus.MaxLatency.Nanoseconds()),
			"status.result.avgLatencyNanoSec":      fmt.Sprint(checkupStatus.AvgLatency.Nanoseconds()),
			"status.result.measurementDurationSec": fmt.Sprint(checkupStatus.MeasurementDuration.Seconds()),
			"status.result.targetNode":             checkupStatus.TargetNode,
			"status.result.sourceNode":             checkupStatus.SourceNode,
			"status.startTimestamp":                timestamp(checkupStatus.StartTimestamp),
			"status.completionTimestamp":           timestamp(checkupStatus.CompletionTimestamp),
			"status.succeeded":                     strconv.FormatBool(true),
			"status.failureReason":                 "",
		}

		assert.Equal(t, expectedReportData, getCheckupData(t, fakeClient, testNamespace, testConfigMapName))
	})

	t.Run("on checkup failure", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newConfigMap())
		testReporter := reporter.New(fakeClient, testNamespace, testConfigMapName)

		var checkupStatus status.Status
		checkupStatus.StartTimestamp = time.Now()
		assert.NoError(t, testReporter.Report(checkupStatus))

		checkupStatus.FailureReason = []string{someFailureReason}
		checkupStatus.CompletionTimestamp = time.Now()
		assert.NoError(t, testReporter.Report(checkupStatus))

		expectedReportData := map[string]string{
			"status.startTimestamp":      timestamp(checkupStatus.StartTimestamp),
			"status.completionTimestamp": timestamp(checkupStatus.CompletionTimestamp),
			"status.succeeded":           strconv.FormatBool(false),
			"status.failureReason":       someFailureReason,
		}

		assert.Equal(t, expectedReportData, getCheckupData(t, fakeClient, testNamespace, testConfigMapName))
	})

	t.Run("on checkup multiple failures", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newConfigMap())
		testReporter := reporter.New(fakeClient, testNamespace, testConfigMapName)

		var checkupStatus status.Status
		checkupStatus.StartTimestamp = time.Now()
		checkupStatus.CompletionTimestamp = time.Now()
		assert.NoError(t, testReporter.Report(checkupStatus))

		checkupStatus.FailureReason = []string{someFailureReason, someOtherFailureReason}
		assert.NoError(t, testReporter.Report(checkupStatus))

		expectedReportData := map[string]string{
			"status.startTimestamp":      timestamp(checkupStatus.StartTimestamp),
			"status.completionTimestamp": timestamp(checkupStatus.CompletionTimestamp),
			"status.succeeded":           strconv.FormatBool(false),
			"status.failureReason":       someFailureReason + "," + someOtherFailureReason,
		}

		assert.Equal(t, expectedReportData, getCheckupData(t, fakeClient, testNamespace, testConfigMapName))
	})
}

func newConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		Data: map[string]string{},
	}
}

func getCheckupData(t *testing.T, client kubernetes.Interface, configMapNamespace, configMapName string) map[string]string {
	configMap, err := configmap.Get(client, configMapNamespace, configMapName)
	assert.NoError(t, err)

	return configMap.Data
}

func timestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}
