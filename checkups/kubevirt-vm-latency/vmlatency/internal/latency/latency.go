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
	"strings"
	"time"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/console"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
	kubevmi "github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/vmi"
)

type Latency struct {
	client kubevmi.KubevirtVmisClient
}

func New(client kubevmi.KubevirtVmisClient) Latency {
	return Latency{client: client}
}

func (l Latency) Check(
	sourceVMI, targetVMI *kvcorev1.VirtualMachineInstance,
	networkInterfaceName string,
	sampleTime time.Duration) (status.Results, error) {
	sourceVMIConsole := console.NewConsole(l.client, sourceVMI)

	if err := sourceVMIConsole.LoginToFedora(); err != nil {
		return status.Results{}, fmt.Errorf("failed to run check: %v", err)
	}

	targetIPAddress, err := kubevmi.NetworkIPAddress(l.client, targetVMI.Namespace, targetVMI.Name, networkInterfaceName)
	if err != nil {
		return status.Results{}, err
	}

	start := time.Now()
	res, err := sourceVMIConsole.RunCommand(composePingCommand(targetIPAddress, sampleTime), sampleTime+time.Minute)
	if err != nil {
		return status.Results{}, err
	}
	measurementDuration := time.Since(start)

	results := ParsePingResults(res)
	results.MeasurementDuration = measurementDuration

	return results, nil
}

func composePingCommand(ipAddress string, timeout time.Duration) string {
	const (
		pingBinaryName  = "ping"
		pingTimeoutFlag = "-w"
	)
	sampleTimeSeconds := fmt.Sprintf("%d", int(timeout.Seconds()))

	return strings.Join([]string{pingBinaryName, ipAddress, pingTimeoutFlag, sampleTimeSeconds}, " ")
}
