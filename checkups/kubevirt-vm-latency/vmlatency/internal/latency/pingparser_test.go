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
	const pingOutput = `
PING 192.168.100.20 (192.168.100.20): 56 data bytes
64 bytes from 192.168.100.20: seq=0 ttl=64 time=0.314 ms
64 bytes from 192.168.100.20: seq=1 ttl=64 time=0.340 ms
64 bytes from 192.168.100.20: seq=2 ttl=64 time=0.461 ms
64 bytes from 192.168.100.20: seq=3 ttl=64 time=0.332 ms
64 bytes from 192.168.100.20: seq=4 ttl=64 time=0.395 ms

--- 192.168.100.20 ping statistics ---
5 packets transmitted, 5 packets received, 0% packet loss
round-trip min/avg/max = 0.314/0.368/0.461 ms
`
	expectedResults := latency.Results{
		Transmitted: 5,
		Received:    5,
		Time:        time.Duration(0),
	}

	var err error
	expectedResults.Min, err = time.ParseDuration("0.314ms")
	assert.NoError(t, err)

	expectedResults.Average, err = time.ParseDuration("0.368ms")
	assert.NoError(t, err)

	expectedResults.Max, err = time.ParseDuration("0.461ms")
	assert.NoError(t, err)

	actualResults, err := latency.ParsePingResults(pingOutput)
	assert.NoError(t, err)

	assert.Equal(t, expectedResults, actualResults)
}
