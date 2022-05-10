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

package reporter

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

type configMapUpdater interface {
	UpdateConfigMap(string, string, map[string]string) error
}

type reporter struct {
	client             configMapUpdater
	configMapName      string
	configMapNamespace string
}

func New(c configMapUpdater, configMapNamespace, configMapName string) *reporter {
	return &reporter{
		client:             c,
		configMapNamespace: configMapNamespace,
		configMapName:      configMapName,
	}
}

func (r *reporter) Report(s status.Status) error {
	data := formatStatus(s)

	if raw, err := json.MarshalIndent(data, "", " "); err == nil {
		log.Printf("reporting status:\n%s\n", string(raw))
	}

	return r.client.UpdateConfigMap(r.configMapNamespace, r.configMapName, data)
}

func formatStatus(s status.Status) map[string]string {
	const (
		succeededKey                 = "status.succeeded"
		failureReasonKey             = "status.failureReason"
		resultMinLatencyKey          = "status.result.minLatencyNanoSec"
		resultAvgLatencyKey          = "status.result.avgLatencyNanoSec"
		resultMaxLatencyKey          = "status.result.maxLatencyNanoSec"
		resultMeasurementDurationKey = "status.result.measurementDurationSec"
	)
	data := map[string]string{}

	data[succeededKey] = strconv.FormatBool(len(s.FailureReason) == 0)
	data[failureReasonKey] = strings.Join(s.FailureReason, ", ")

	var emptyResults status.Results
	if s.Results != emptyResults {
		const base = 10
		data[resultMinLatencyKey] = strconv.FormatInt(s.Results.MinLatency.Nanoseconds(), base)
		data[resultAvgLatencyKey] = strconv.FormatInt(s.Results.AvgLatency.Nanoseconds(), base)
		data[resultMaxLatencyKey] = strconv.FormatInt(s.Results.MaxLatency.Nanoseconds(), base)
		data[resultMeasurementDurationKey] = strconv.FormatInt(int64(s.Results.MeasurementDuration.Seconds()), base)
	}

	return data
}
