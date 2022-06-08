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

package netattachdef_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/netattachdef"
)

const bridgeWithTuningConfig = `
      {
        "cniVersion":"0.3.1",
        "name": "br10",
        "plugins": [
            {
                "type": "bridge",
                "bridge": "br10"
            },
			{
                "type": "tuning"
            }
        ]
      }
`

const cnvBridgeConfig = `
      {
        "cniVersion":"0.3.1",
        "name": "br10",
        "plugins": [
            {
                "type": "cnv-bridge",
                "bridge": "br10"
            }
        ]
      }
`

const sriovConfig = `
    {
      "cniVersion":"0.3.1",
      "name":"sriov-network",
      "type":"sriov",
      "vlan":0,
      "spoofchk":"on",
      "trust":"off",
      "vlanQoS":0,
      "link_state":"enable",
      "ipam":{}
    }
`

type testCase struct {
	description        string
	netAttachDefConfig string
	expectedCniPlugins []string
}

func TestListCNIPluginTypesShould(t *testing.T) {
	const (
		bridgeCniPluginName    = "bridge"
		cnvBridgeCniPluginName = "cnv-bridge"
		tuningCniPluginName    = "tuning"
		sriovCniPluginName     = "sriov"
	)
	testCases := []testCase{
		{
			"return nil when config is empty",
			"",
			nil,
		},
		{
			"succeed when config includes more then one CNI plugin",
			bridgeWithTuningConfig,
			[]string{bridgeCniPluginName, tuningCniPluginName},
		},
		{
			"succeed when config includes cnv-bridge CNI plugin",
			cnvBridgeConfig,
			[]string{cnvBridgeCniPluginName},
		},
		{
			"succeed when config includes SR-IOV CNI plugin",
			sriovConfig,
			[]string{sriovCniPluginName},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			assert.Equal(t, testCase.expectedCniPlugins, netattachdef.ListCNIPluginTypes(testCase.netAttachDefConfig))
		})
	}
}
