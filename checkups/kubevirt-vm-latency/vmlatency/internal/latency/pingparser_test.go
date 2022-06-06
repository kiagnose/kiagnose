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

package latency_test

import (
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/latency"
)

const successfulPingOutput = `
PING 1.1.1.1 (1.1.1.1) 56(84) bytes of data.
64 bytes from 1.1.1.1: icmp_seq=1 ttl=58 time=2.17 ms
64 bytes from 1.1.1.1: icmp_seq=2 ttl=58 time=1.98 ms
64 bytes from 1.1.1.1: icmp_seq=3 ttl=58 time=2.38 ms
64 bytes from 1.1.1.1: icmp_seq=4 ttl=58 time=2.11 ms
64 bytes from 1.1.1.1: icmp_seq=5 ttl=58 time=1.73 ms
^C
--- 1.1.1.1 ping statistics ---
5 packets transmitted, 5 received, 0% packet loss, time 4004ms
rtt min/avg/max/mdev = 1.732/2.074/2.382/0.214 ms
`

var successfulPingResults = latency.Results{
	Min:         1732000 * time.Nanosecond,
	Average:     2074000 * time.Nanosecond,
	Max:         2382000 * time.Nanosecond,
	Time:        4004 * time.Millisecond,
	Transmitted: 5,
	Received:    5,
}

type pingParserTestCase struct {
	description     string
	pingOutput      string
	expectedResults latency.Results
}

func TestParsePingShouldSucceedGiven(t *testing.T) {
	testCases := []pingParserTestCase{
		{
			description:     "successful ping output",
			pingOutput:      successfulPingOutput,
			expectedResults: successfulPingResults,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			assert.Equal(t, testCase.expectedResults, latency.ParsePingResults(testCase.pingOutput))
		})
	}
}
