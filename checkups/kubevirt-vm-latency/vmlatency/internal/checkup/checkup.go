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

package checkup

import (
	"context"
	"fmt"
	"strings"
	"time"

	k8scorev1 "k8s.io/api/core/v1"

	kvcorev1 "kubevirt.io/api/core/v1"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/vmi"
)

type checker interface {
	Check(sourceVMI, targetVMI *kvcorev1.VirtualMachineInstance, sampleTime time.Duration) error
	MinLatency() time.Duration
	AverageLatency() time.Duration
	MaxLatency() time.Duration
	CheckDuration() time.Duration
}

type checkup struct {
	client    vmi.KubevirtVmisClient
	namespace string
	params    config.CheckupParameters
	results   status.Results
	sourceVM  *kvcorev1.VirtualMachineInstance
	targetVM  *kvcorev1.VirtualMachineInstance
	checker   checker
}

func New(c vmi.KubevirtVmisClient, namespace string, params config.CheckupParameters, checker checker) *checkup {
	return &checkup{
		client:    c,
		namespace: namespace,
		params:    params,
		checker:   checker,
	}
}

func (c *checkup) Preflight() error {
	return nil
}

const (
	SourceVmiName       = "latency-check-source"
	TargetVmiName       = "latency-check-target"
	LabelLatencyCheckVM = "latency-check-vm"
)

func (c *checkup) Setup(ctx context.Context) error {
	const (
		errMessagePrefix = "setup"

		defaultSetupTimeout = time.Minute * 10

		sourceVmiMac  = "02:00:00:01:00:01"
		sourceVmiCidr = "192.168.100.10/24"
		targetVmiMac  = "02:00:00:02:00:02"
		targetVmiCidr = "192.168.100.20/24"
	)

	netAttachDef, err := c.client.GetNetworkAttachmentDefinition(
		c.params.NetworkAttachmentDefinitionNamespace,
		c.params.NetworkAttachmentDefinitionName)
	if err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	sourceVmi := newLatencyCheckVmi(SourceVmiName, c.params.SourceNodeName, netAttachDef, sourceVmiMac, sourceVmiCidr)
	targetVmi := newLatencyCheckVmi(TargetVmiName, c.params.TargetNodeName, netAttachDef, targetVmiMac, targetVmiCidr)

	if err = vmi.Start(c.client, c.namespace, sourceVmi); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}
	if err = vmi.Start(c.client, c.namespace, targetVmi); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, defaultSetupTimeout)
	defer cancel()
	if c.targetVM, err = vmi.WaitForStatusIPAddress(waitCtx, c.client, c.namespace, targetVmi.Name); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	if c.sourceVM, err = vmi.WaitForStatusIPAddress(waitCtx, c.client, c.namespace, sourceVmi.Name); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	c.results.TargetNode = c.targetVM.Status.NodeName
	c.results.SourceNode = c.sourceVM.Status.NodeName
	return nil
}

func newLatencyCheckVmi(
	name,
	nodeName string,
	netAttachDef *netattdefv1.NetworkAttachmentDefinition,
	macAddress, cidr string) *kvcorev1.VirtualMachineInstance {
	const networkName = "net0"

	vmLabel := vmi.Label{Key: LabelLatencyCheckVM, Value: ""}
	var affinity *k8scorev1.Affinity
	if nodeName != "" {
		affinity = &k8scorev1.Affinity{NodeAffinity: vmi.NewNodeAffinity(nodeName)}
	} else {
		affinity = &k8scorev1.Affinity{PodAntiAffinity: vmi.NewPodAntiAffinity(vmLabel)}
	}

	return vmi.NewAlpine(name,
		vmi.WithLabels(vmLabel),
		vmi.WithAffinity(affinity),
		vmi.WithMultusNetwork(networkName, netAttachDef.Namespace+"/"+netAttachDef.Name),
		vmi.WithInterface(
			networkName,
			vmi.WithMacAddress(macAddress),
			vmi.WithBindingFromNetAttachDef(netAttachDef),
		),
		vmi.WithCloudInitNoCloudNetworkData(
			vmi.WithEthernet(
				networkName,
				vmi.WithAddresses(cidr),
				vmi.WithMatchingMAC(macAddress),
			),
		),
	)
}

func (c *checkup) Run() error {
	sampleDuration := time.Duration(c.params.SampleDurationSeconds) * time.Second
	if err := c.checker.Check(c.sourceVM, c.targetVM, sampleDuration); err != nil {
		return fmt.Errorf("run: %v", err)
	}

	c.results.MinLatency = c.checker.MinLatency()
	c.results.AvgLatency = c.checker.AverageLatency()
	c.results.MaxLatency = c.checker.MaxLatency()
	c.results.MeasurementDuration = c.checker.CheckDuration()

	actualMaxLatency := c.results.MaxLatency.Milliseconds()
	maxLatencyDesired := int64(c.params.DesiredMaxLatencyMilliseconds)
	if actualMaxLatency > maxLatencyDesired {
		return fmt.Errorf("run : actual max latency (%d) is greater then desired (%d)", actualMaxLatency, maxLatencyDesired)
	}

	return nil
}

func (c *checkup) Teardown(waitCtx context.Context) error {
	const (
		errMessagePrefix = "teardown"

		defaultTeardownTimeout = time.Minute * 3
	)

	var teardownErrors []string
	if err := vmi.Delete(c.client, c.namespace, c.sourceVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.sourceVM.Name, err))
	}

	if err := vmi.Delete(c.client, c.namespace, c.targetVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.targetVM.Name, err))
	}

	waitCtx, cancel := context.WithTimeout(waitCtx, defaultTeardownTimeout)
	defer cancel()

	if err := vmi.WaitForVmiDispose(waitCtx, c.client, c.namespace, c.sourceVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.sourceVM.Name, err))
	}

	if err := vmi.WaitForVmiDispose(waitCtx, c.client, c.namespace, c.targetVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.targetVM.Name, err))
	}

	if len(teardownErrors) > 0 {
		return fmt.Errorf("%s: %v", errMessagePrefix, strings.Join(teardownErrors, ", "))
	}

	return nil
}

func (c *checkup) Results() status.Results {
	return c.results
}
