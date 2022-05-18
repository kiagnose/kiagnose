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

func TestParsePingResults(t *testing.T) {
	const (
		pingOutput = `
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
	)
	expectedResults := latency.Results{
		Transmitted: 5,
		Received:    5,
	}
	var err error
	if expectedResults.Time, err = time.ParseDuration("4004ms"); err != nil {
		panic(err)
	}
	if expectedResults.Min, err = time.ParseDuration("1.732ms"); err != nil {
		panic(err)
	}
	if expectedResults.Average, err = time.ParseDuration("2.074ms"); err != nil {
		panic(err)
	}
	if expectedResults.Max, err = time.ParseDuration("2.382ms"); err != nil {
		panic(err)
	}

	assert.Equal(t, latency.ParsePingResults(pingOutput), expectedResults)
}
