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

	"k8s.io/client-go/kubernetes"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
	kreporter "github.com/kiagnose/kiagnose/kiagnose/reporter"
)

type reporter struct {
	kreporter.Reporter
}

func New(c kubernetes.Interface, configMapNamespace, configMapName string) *reporter {
	r := kreporter.New(c, configMapNamespace, configMapName)
	return &reporter{*r}
}

func (r *reporter) Report(s status.Status) error {
	if !r.HasData() {
		return r.Reporter.Report(s.Status)
	}

	// TODO: Update the base reporter to drop the `Succeeded` field.
	s.Succeeded = len(s.FailureReason) == 0

	data := formatResults(s)
	if raw, err := json.MarshalIndent(data, "", " "); err == nil {
		log.Printf("reporting status:\n%s\n", string(raw))
	}

	s.Status.Results = data
	return r.Reporter.Report(s.Status)
}

func formatResults(s status.Status) map[string]string {
	const (
		resultMinLatencyKey          = "minLatencyNanoSec"
		resultAvgLatencyKey          = "avgLatencyNanoSec"
		resultMaxLatencyKey          = "maxLatencyNanoSec"
		resultMeasurementDurationKey = "measurementDurationSec"
		resultSourceNode             = "sourceNode"
		resultTargetNode             = "targetNode"
	)
	data := map[string]string{}

	var emptyResults status.Results
	if s.Results != emptyResults {
		const base = 10
		data[resultMinLatencyKey] = strconv.FormatInt(s.Results.MinLatency.Nanoseconds(), base)
		data[resultAvgLatencyKey] = strconv.FormatInt(s.Results.AvgLatency.Nanoseconds(), base)
		data[resultMaxLatencyKey] = strconv.FormatInt(s.Results.MaxLatency.Nanoseconds(), base)
		data[resultMeasurementDurationKey] = strconv.FormatInt(int64(s.Results.MeasurementDuration.Seconds()), base)
		data[resultSourceNode] = s.Results.SourceNode
		data[resultTargetNode] = s.Results.TargetNode
	}

	return data
}
