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

	"k8s.io/apimachinery/pkg/types"

	kvcorev1 "kubevirt.io/api/core/v1"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/vmi"
)

type checkup struct {
	client    vmi.KubevirtVmisClient
	namespace string
	params    config.CheckupParameters
	results   status.Results
	sourceVM  *kvcorev1.VirtualMachineInstance
	targetVM  *kvcorev1.VirtualMachineInstance
}

func New(c vmi.KubevirtVmisClient, namespace string, params config.CheckupParameters) *checkup {
	return &checkup{
		client:    c,
		namespace: namespace,
		params:    params,
	}
}

func (c *checkup) Preflight() error {
	return nil
}

func (c *checkup) Setup(ctx context.Context) error {
	const (
		errMessagePrefix = "setup"

		defaultSetupTimeout = time.Minute * 10

		networkName   = "net0"
		sourceVmiName = "latency-check-source"
		sourceVmiMac  = "02:00:00:01:00:01"
		sourceVmiCidr = "192.168.100.10/24"
		targetVmiName = "latency-check-target"
		targetVmiMac  = "02:00:00:02:00:02"
		targetVmiCidr = "192.168.100.20/24"
	)

	netAttachDefNamespacedName := types.NamespacedName{
		Namespace: c.params.NetworkAttachmentDefinitionNamespace,
		Name:      c.params.NetworkAttachmentDefinitionName,
	}

	sourceVmi := newLatencyCheckVmi(
		sourceVmiName,
		c.params.SourceNodeName,
		networkName, netAttachDefNamespacedName,
		sourceVmiMac, sourceVmiCidr,
	)

	targetVmi := newLatencyCheckVmi(
		targetVmiName,
		c.params.TargetNodeName,
		networkName, netAttachDefNamespacedName,
		targetVmiMac, targetVmiCidr,
	)

	if err := vmi.Start(c.client, c.namespace, sourceVmi); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	if err := vmi.Start(c.client, c.namespace, targetVmi); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, defaultSetupTimeout)
	defer cancel()

	var err error
	if c.targetVM, err = vmi.WaitUntilReady(waitCtx, c.client, c.namespace, targetVmi.Name); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	if c.sourceVM, err = vmi.WaitUntilReady(waitCtx, c.client, c.namespace, sourceVmi.Name); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	return nil
}

func newLatencyCheckVmi(
	name,
	nodeName,
	networkName string, netAttachDef types.NamespacedName,
	macAddress, cidr string) *kvcorev1.VirtualMachineInstance {
	networkData, _ := vmi.NewNetworkData(
		vmi.WithEthernet(networkName,
			vmi.WithAddresses(cidr),
			vmi.WithMatchingMAC(macAddress),
		),
	)
	return vmi.NewFedora(name,
		vmi.WithNodeSelector(nodeName),
		vmi.WithMultusNetwork(networkName, netAttachDef.String()),
		vmi.WithInterface(
			vmi.NewInterface(networkName,
				vmi.WithMacAddress(macAddress),
				vmi.WithSriovBinding(),
			),
		),
		vmi.WithCloudInitNoCloudNetworkData(networkData),
	)
}

func (c *checkup) Run() error {
	const errMessagePrefix = "run"
	return fmt.Errorf("%s: not implemented", errMessagePrefix)
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
