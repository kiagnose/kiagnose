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
		errMessagePrefix   = "ping parser"
		millisecondsSuffix = "ms"
	)

	const (
		totalPacketLoss = "100% packet loss"

		statisticsPattern = `(\d+)\s+packets transmitted,\s+(\d+)\s+packets received,\s+(\d+)%\s+packet loss\s+` +
			`round-trip min/avg/max = (\d+\.\d+)/(\d+\.\d+)/(\d+\.\d+) ms`
		expectedElements = 7
	)

	var (
		results Results
		err     error
	)

	if strings.Contains(pingResult, totalPacketLoss) {
		return Results{}, fmt.Errorf("%s: no connectivity - 100%% packet loss", errMessagePrefix)
	}

	matches := regexp.MustCompile(statisticsPattern).FindStringSubmatch(pingResult)

	if len(matches) != expectedElements {
		return Results{}, fmt.Errorf("%s: input does not match regex", errMessagePrefix)
	}

	results.Transmitted, err = strconv.Atoi(matches[1])
	if err != nil {
		return Results{}, fmt.Errorf("%s: failed to parse 'packets transmitted': %v", errMessagePrefix, err)
	}

	results.Received, err = strconv.Atoi(matches[2])
	if err != nil {
		log.Printf("%s: failed to parse 'packets received': %v", errMessagePrefix, err)
	}

	results.Min, err = time.ParseDuration(matches[4] + millisecondsSuffix)
	if err != nil {
		return Results{}, fmt.Errorf("%s: failed to parse 'min': %v", errMessagePrefix, err)
	}

	results.Average, err = time.ParseDuration(matches[5] + millisecondsSuffix)
	if err != nil {
		return Results{}, fmt.Errorf("%s: failed to parse 'avg': %v", errMessagePrefix, err)
	}

	results.Max, err = time.ParseDuration(matches[6] + millisecondsSuffix)
	if err != nil {
		return Results{}, fmt.Errorf("%s: failed to parse 'max': %v", errMessagePrefix, err)
	}

	return results, nil
}
