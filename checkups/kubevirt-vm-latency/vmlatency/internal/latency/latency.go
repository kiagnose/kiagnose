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
	"context"
	"fmt"
	"strings"
	"time"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/console"
	kubevmi "github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/vmi"
)

type Latency struct {
	client  kubevmi.KubevirtVmisClient
	results Results
}

func New(client kubevmi.KubevirtVmisClient) *Latency {
	return &Latency{client: client}
}

func (l *Latency) MinLatency() time.Duration {
	return l.results.Min
}

func (l *Latency) AverageLatency() time.Duration {
	return l.results.Average
}

func (l *Latency) MaxLatency() time.Duration {
	return l.results.Max
}

func (l *Latency) CheckDuration() time.Duration {
	return l.results.Time
}

func (l *Latency) Check(sourceVMI, targetVMI *kvcorev1.VirtualMachineInstance, sampleTime time.Duration) error {
	const errMessagePrefix = "failed to run check"
	sourceVMIConsole := console.NewConsole(l.client, sourceVMI)

	if err := sourceVMIConsole.LoginToFedora(); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	const waitForStatusIPAddressTimeout = time.Minute * 5
	ctx, cancel := context.WithTimeout(context.Background(), waitForStatusIPAddressTimeout)
	defer cancel()
	targetIPAddress, err := kubevmi.WaitForStatusIPAddress(ctx, l.client, targetVMI.Namespace, targetVMI.Name)
	if err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	const runCommandGracePeriod = time.Minute * 1
	res, err := sourceVMIConsole.RunCommand(composePingCommand(targetIPAddress, sampleTime), sampleTime+runCommandGracePeriod)
	if err != nil {
		return err
	}
	l.results = ParsePingResults(res)

	if l.results.Transmitted == 0 || l.results.Received == 0 {
		return fmt.Errorf("%s: failed due to connectivity issue: %d packets transmitted, %d packets received",
			errMessagePrefix, l.results.Transmitted, l.results.Received)
	}

	return nil
}

func composePingCommand(ipAddress string, timeout time.Duration) string {
	const (
		pingBinaryName  = "ping"
		pingTimeoutFlag = "-w"
	)
	sampleTimeSeconds := fmt.Sprintf("%d", int(timeout.Seconds()))

	return strings.Join([]string{pingBinaryName, ipAddress, pingTimeoutFlag, sampleTimeSeconds}, " ")
}
