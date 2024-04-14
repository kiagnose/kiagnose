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

func ParsePingResults(pingResult string) (Results, error) {
	const (
		errMessagePrefix = "ping parser"

		millisecondsSuffix = "ms"
	)
	var results Results
	var err error

	p := regexp.MustCompile(`(\d+)\s*\S*packets\s*transmitted,\s*(\d+)\s*(packets )?received,\s*(\d+)%\s*packet\s*loss(, time (\d+))?`)
	statisticsPatternMatches := p.FindAllStringSubmatch(pingResult, -1)
	for _, item := range statisticsPatternMatches {
		if results.Transmitted, err = strconv.Atoi(strings.TrimSpace(item[1])); err != nil {
			log.Printf("%s: failed to parse 'time': %v", errMessagePrefix, err)
		}

		if results.Received, err = strconv.Atoi(strings.TrimSpace(item[2])); err != nil {
			log.Printf("%s: failed to parse 'time': %v", errMessagePrefix, err)
		}

		if results.Time, err = time.ParseDuration(fmt.Sprintf("%s%s", strings.TrimSpace(item[6]), millisecondsSuffix)); err != nil {
			log.Printf("%s: failed to parse 'time': %v", errMessagePrefix, err)
		}
	}

	latencyPattern := regexp.MustCompile(`(round-trip|rtt)\s+\S+\s*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)(/[0-9.]+)?\s*ms`)
	latencyPatternMatches := latencyPattern.FindAllStringSubmatch(pingResult, -1)
	for _, item := range latencyPatternMatches {
		if results.Min, err = time.ParseDuration(fmt.Sprintf("%s%s", strings.TrimSpace(item[2]), millisecondsSuffix)); err != nil {
			log.Printf("%s: failed to parse 'min': %v", errMessagePrefix, err)
		}

		if results.Average, err = time.ParseDuration(fmt.Sprintf("%s%s", strings.TrimSpace(item[3]), millisecondsSuffix)); err != nil {
			log.Printf("%s: failed to parse 'avg': %v", errMessagePrefix, err)
		}

		if results.Max, err = time.ParseDuration(fmt.Sprintf("%s%s", strings.TrimSpace(item[4]), millisecondsSuffix)); err != nil {
			log.Printf("%s: failed to parse 'max': %v", errMessagePrefix, err)
		}
	}

	return results, nil
}
