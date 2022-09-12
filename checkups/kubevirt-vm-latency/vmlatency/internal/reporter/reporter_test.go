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
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/reporter"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

const (
	testNamespace     = "default"
	testConfigMapName = "results"
)

func TestReportShouldRunSuccessfullyWhen(t *testing.T) {
	t.Run("status is initialized", func(t *testing.T) {
		testReporter := reporter.New(&configMapClientStub{}, testNamespace, testConfigMapName)

		assert.NoError(t, testReporter.Report(status.Status{}))
	})
}

func TestReportShouldFailWhen(t *testing.T) {
	t.Run("failed to update results ConfigMap", func(t *testing.T) {
		expectedErr := errors.New("update fail")
		testReporter := reporter.New(&configMapClientStub{failUpdateConfigMap: expectedErr}, testNamespace, testConfigMapName)

		assert.Equal(t, testReporter.Report(status.Status{}), expectedErr)
	})
}

func TestReportShouldSuccessfullyConvertResultValues(t *testing.T) {
	const (
		someFailureReason      = "some reason"
		someOtherFailureReason = "some other reason"
	)
	testClient := &configMapClientStub{}
	testReporter := reporter.New(testClient, testNamespace, testConfigMapName)

	t.Run("on checkup successful completion", func(t *testing.T) {
		checkupStatus := status.Status{
			FailureReason: []string{},
			Results: status.Results{
				MinLatency:          1 * time.Minute,
				AvgLatency:          2 * time.Minute,
				MeasurementDuration: 3 * time.Minute,
				MaxLatency:          4 * time.Minute,
				TargetNode:          "a",
				SourceNode:          "b",
			},
		}
		assert.NoError(t, testReporter.Report(checkupStatus))

		expectedReportData := map[string]string{
			"status.result.minLatencyNanoSec":      fmt.Sprint(checkupStatus.MinLatency.Nanoseconds()),
			"status.result.maxLatencyNanoSec":      fmt.Sprint(checkupStatus.MaxLatency.Nanoseconds()),
			"status.result.avgLatencyNanoSec":      fmt.Sprint(checkupStatus.AvgLatency.Nanoseconds()),
			"status.result.measurementDurationSec": fmt.Sprint(checkupStatus.MeasurementDuration.Seconds()),
			"status.result.targetNode":             checkupStatus.TargetNode,
			"status.result.sourceNode":             checkupStatus.SourceNode,
			"status.succeeded":                     strconv.FormatBool(true),
			"status.failureReason":                 "",
		}

		assert.Equal(t, expectedReportData, testClient.configMapData)
	})

	t.Run("on checkup failure", func(t *testing.T) {
		checkupStatus := status.Status{
			FailureReason: []string{someFailureReason},
		}
		assert.NoError(t, testReporter.Report(checkupStatus))

		expectedReportData := map[string]string{
			"status.succeeded":     strconv.FormatBool(false),
			"status.failureReason": someFailureReason,
		}

		assert.Equal(t, expectedReportData, testClient.configMapData)
	})

	t.Run("on checkup multiple failures", func(t *testing.T) {
		checkupStatus := status.Status{
			FailureReason: []string{someFailureReason, someOtherFailureReason},
		}
		assert.NoError(t, testReporter.Report(checkupStatus))

		expectedReportData := map[string]string{
			"status.succeeded":     strconv.FormatBool(false),
			"status.failureReason": someFailureReason + ", " + someOtherFailureReason,
		}

		assert.Equal(t, expectedReportData, testClient.configMapData)
	})
}

type configMapClientStub struct {
	failUpdateConfigMap error
	configMapData       map[string]string
}

func (c *configMapClientStub) UpdateConfigMap(_, _ string, data map[string]string) error {
	c.configMapData = data
	return c.failUpdateConfigMap
}
