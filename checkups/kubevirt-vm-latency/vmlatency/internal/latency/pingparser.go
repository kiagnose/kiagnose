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

package latency

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Results struct {
	Min         time.Duration
	Max         time.Duration
	Average     time.Duration
	Time        time.Duration
	Transmitted int
	Received    int
}

func ParsePingResults(rawPingOutput string) Results {
	statistics, err := parseStatistics(rawPingOutput)
	if err != nil {
		log.Printf("ping-parser: failed to parse ping statistics: %v\n%s\n", err, rawPingOutput)
	}

	latencyResults, err := parseLatencyResults(rawPingOutput)
	if err != nil {
		log.Printf("ping-parser: failed to parse ping pingParser results: %v\n%s\n", err, rawPingOutput)
	}

	return Results{
		Min:         latencyResults.min,
		Max:         latencyResults.max,
		Average:     latencyResults.average,
		Time:        statistics.time,
		Transmitted: statistics.transmitted,
		Received:    statistics.received,
	}
}

const millisecondsUnit = "ms"

type pingLatencyResults struct {
	min     time.Duration
	max     time.Duration
	average time.Duration
}

func parseLatencyResults(rawPingOutput string) (pingLatencyResults, error) {
	const (
		minCaptureGroup = "min"
		avgCaptureGroup = "avg"
		maxCaptureGroup = "max"
	)
	var pattern = `(round-trip|rtt)\s+min/avg/max(/mdev)?\s+=\s+` +
		fmt.Sprintf(`(?P<%s>[\d.]+)/`, minCaptureGroup) +
		fmt.Sprintf(`(?P<%s>[\d.]+)/`, avgCaptureGroup) +
		fmt.Sprintf(`(?P<%s>[\d.]+)`, maxCaptureGroup) +
		`(/[\d.]+)?` +
		`\s+ms`

	matchesByCaptureGroup, err := findStringByNamedCaptureGroup(rawPingOutput, pattern)
	if err != nil {
		return pingLatencyResults{}, err
	}

	var pingLatency pingLatencyResults
	min, exists := matchesByCaptureGroup[minCaptureGroup]
	if !exists {
		return pingLatencyResults{}, errors.New("failed to parse 'min': not match found")
	}
	if pingLatency.min, err = time.ParseDuration(strings.TrimSpace(min) + millisecondsUnit); err != nil {
		return pingLatencyResults{}, fmt.Errorf("failed to parse 'min':%v", err)
	}

	average, exists := matchesByCaptureGroup[avgCaptureGroup]
	if !exists {
		return pingLatencyResults{}, errors.New("failed to parse 'avg': not match found")
	}
	if pingLatency.average, err = time.ParseDuration(strings.TrimSpace(average) + millisecondsUnit); err != nil {
		return pingLatencyResults{}, fmt.Errorf("failed to parse 'avg':%v", err)
	}

	max, exists := matchesByCaptureGroup[maxCaptureGroup]
	if !exists {
		return pingLatencyResults{}, errors.New("failed to parse 'max': not match found")
	}
	if pingLatency.max, err = time.ParseDuration(strings.TrimSpace(max) + millisecondsUnit); err != nil {
		return pingLatencyResults{}, fmt.Errorf("failed to parse 'max':%v", err)
	}

	return pingLatency, nil
}

type pingStatistics struct {
	transmitted int
	received    int
	time        time.Duration
}

func parseStatistics(rawPingOutput string) (pingStatistics, error) {
	const (
		transmittedCaptureGroup = "transmitted"
		receivedCaptureGroup    = "received"
		timeCaptureGroup        = "time"
	)

	var pattern = fmt.Sprintf(`(?P<%s>\d+) packets transmitted,\s+`, transmittedCaptureGroup) +
		fmt.Sprintf(`(?P<%s>\d+) received,\s+`, receivedCaptureGroup) +
		`(\+\d+ errors,\s+)?` +
		`\d+% packet loss,\s+` +
		fmt.Sprintf(`time (?P<%s>\d+)`, timeCaptureGroup)

	matchesByCaptureGroup, err := findStringByNamedCaptureGroup(rawPingOutput, pattern)
	if err != nil {
		return pingStatistics{}, err
	}

	var pingStats pingStatistics
	transmitted, exists := matchesByCaptureGroup[transmittedCaptureGroup]
	if !exists {
		return pingStatistics{}, errors.New("failed to parse 'packets transmitted': not match found")
	}
	if pingStats.transmitted, err = strconv.Atoi(strings.TrimSpace(transmitted)); err != nil {
		return pingStatistics{}, fmt.Errorf("failed to parse 'packets transmitted': %v", err)
	}

	received, exists := matchesByCaptureGroup[receivedCaptureGroup]
	if !exists {
		return pingStatistics{}, errors.New("failed to parse 'packets received': not match found")
	}
	if pingStats.received, err = strconv.Atoi(strings.TrimSpace(received)); err != nil {
		return pingStatistics{}, fmt.Errorf("failed to parse 'packets received': %v", err)
	}

	if pingDuration, exists := matchesByCaptureGroup[timeCaptureGroup]; exists {
		pingStats.time, _ = time.ParseDuration(strings.TrimSpace(pingDuration) + millisecondsUnit)
	}

	return pingStats, nil
}

// findStringByNamedCaptureGroup expects input string and a regular-expression with named capture groups.
// It returns a map with the matched strings (value) by each capture group name (key).
// In case no match found it returns an error.
//
// Example
//
// Code:
//   matches, err := findStringByNamedCaptureGroup( "blah world blah",  "blah (?P<hello>[a-z]+) blah")
//   fmt.Printf("%v\n", matches["hello"])
// Output:
//   "world"
func findStringByNamedCaptureGroup(rawPingOutput, pattern string) (map[string]string, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := r.FindStringSubmatch(rawPingOutput)
	if len(matches) < 1 {
		return nil, errors.New("no match found")
	}
	names := r.SubexpNames()

	lookupMatchByGroupName := make(map[string]string, len(matches))
	for i, match := range matches {
		lookupMatchByGroupName[names[i]] = match
	}

	return lookupMatchByGroupName, nil
}
