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

const duplicatePingOutput = `
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

const pingOutputWithoutLatencyInfo = `
PING 1.1.1.1 (1.1.1.1) 56(84) bytes of data.
64 bytes from 1.1.1.1: icmp_seq=1 ttl=58 time=2.17 ms
64 bytes from 1.1.1.1: icmp_seq=2 ttl=58 time=1.98 ms
64 bytes from 1.1.1.1: icmp_seq=3 ttl=58 time=2.38 ms
64 bytes from 1.1.1.1: icmp_seq=4 ttl=58 time=2.11 ms
64 bytes from 1.1.1.1: icmp_seq=5 ttl=58 time=1.73 ms
^C
--- 1.1.1.1 ping statistics ---
5 packets transmitted, 5 received, 0% packet loss, time 4004ms
`

var pingResultsWithoutLatencyInfo = latency.Results{
	Transmitted: 5,
	Received:    5,
	Time:        time.Millisecond * 4004,
}

const pingOutputWithoutPacketsInfo = `
rtt min/avg/max/mdev = 1.732/2.074/2.382/0.214 ms
`

var pingResultsWithoutLacksPacketsInfo = latency.Results{
	Min:     time.Nanosecond * 1732000,
	Average: time.Nanosecond * 2074000,
	Max:     time.Nanosecond * 2382000,
}

const failingPingOutput = `
 PING 192.168.100.20 (192.168.100.20) 56(84) bytes of data.
 From 192.168.100.10 icmp_seq=1 Destination Host Unreachable
 From 192.168.100.10 icmp_seq=2 Destination Host Unreachable
 From 192.168.100.10 icmp_seq=3 Destination Host Unreachable

 --- 192.168.100.20 ping statistics ---
 3 packets transmitted, 0 received, +3 errors, 100% packet loss, time 2085ms
 `

var failingPingResults = latency.Results{
	Transmitted: 3,
	Received:    0,
	Time:        time.Millisecond * 2085,
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
		{
			description:     "failing ping output",
			pingOutput:      failingPingOutput,
			expectedResults: failingPingResults,
		},
		{
			description:     "empty string",
			pingOutput:      "",
			expectedResults: latency.Results{},
		},
		{
			description:     "invalid ping output",
			pingOutput:      "YmxhaGJsYWhibGFoCg==",
			expectedResults: latency.Results{},
		},
		{
			description:     "duplicated ping output",
			pingOutput:      duplicatePingOutput,
			expectedResults: successfulPingResults,
		},
		{
			description:     "ping output without packets info",
			pingOutput:      pingOutputWithoutLatencyInfo,
			expectedResults: pingResultsWithoutLatencyInfo,
		},
		{
			description:     "ping output without latency info",
			pingOutput:      pingOutputWithoutPacketsInfo,
			expectedResults: pingResultsWithoutLacksPacketsInfo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			assert.Equal(t, testCase.expectedResults, latency.ParsePingResults(testCase.pingOutput))
		})
	}
}
