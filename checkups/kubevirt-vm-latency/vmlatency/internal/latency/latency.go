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

	var err error

	sourceVMIConsole := console.NewConsole(l.client, sourceVMI)

	if err = sourceVMIConsole.LoginToAlpine(); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	const runCommandGracePeriod = time.Minute * 1
	targetIPAddress := targetVMI.Status.Interfaces[0].IP
	start := time.Now()
	res, err := sourceVMIConsole.RunCommand(composePingCommand(targetIPAddress, sampleTime), sampleTime+runCommandGracePeriod)
	pingTime := time.Since(start)
	if err != nil {
		return err
	}

	l.results, err = ParsePingResults(res)
	if err != nil {
		return err
	}

	if l.results.Time == 0 {
		l.results.Time = pingTime
	}

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
