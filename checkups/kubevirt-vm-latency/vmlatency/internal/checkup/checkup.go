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
	"log"
	"strings"
	"time"

	k8scorev1 "k8s.io/api/core/v1"
	k8srand "k8s.io/apimachinery/pkg/util/rand"

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
	uid       string
	namespace string
	params    config.Config
	results   status.Results
	sourceVM  *kvcorev1.VirtualMachineInstance
	targetVM  *kvcorev1.VirtualMachineInstance
	checker   checker
}

func New(c vmi.KubevirtVmisClient, uid, namespace string, params config.Config, checker checker) *checkup {
	return &checkup{
		client:    c,
		uid:       uid,
		namespace: namespace,
		params:    params,
		checker:   checker,
	}
}

const (
	SourceVMINamePrefix  = "latency-check-source"
	TargetVMINamePrefix  = "latency-check-target"
	LabelLatencyCheckUID = "latency-check/uid"
)

func (c *checkup) Setup(ctx context.Context) (setupErr error) {
	const errMessagePrefix = "setup"

	netAttachDef, err := c.client.GetNetworkAttachmentDefinition(
		ctx,
		c.params.NetworkAttachmentDefinitionNamespace,
		c.params.NetworkAttachmentDefinitionName)
	if err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	sourceVMIName := randomizeName(SourceVMINamePrefix)
	targetVMIName := randomizeName(TargetVMINamePrefix)

	sourceVmi := newLatencyCheckVmi(c.uid, sourceVMIName, c.params.SourceNodeName, c.params.PodName, c.params.PodUID, netAttachDef)
	targetVmi := newLatencyCheckVmi(c.uid, targetVMIName, c.params.TargetNodeName, c.params.PodName, c.params.PodUID, netAttachDef)

	if err = vmi.Start(ctx, c.client, c.namespace, sourceVmi); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}
	defer func() {
		if setupErr != nil {
			c.cleanupVMI(sourceVmi.Name)
		}
	}()

	if err = vmi.Start(ctx, c.client, c.namespace, targetVmi); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}
	defer func() {
		if setupErr != nil {
			c.cleanupVMI(targetVmi.Name)
		}
	}()

	if c.targetVM, err = vmi.WaitForStatusIPAddress(ctx, c.client, c.namespace, targetVmi.Name); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	if c.sourceVM, err = vmi.WaitForStatusIPAddress(ctx, c.client, c.namespace, sourceVmi.Name); err != nil {
		return fmt.Errorf("%s: %v", errMessagePrefix, err)
	}

	c.results.TargetNode = c.targetVM.Status.NodeName
	c.results.SourceNode = c.sourceVM.Status.NodeName
	return nil
}

func (c *checkup) cleanupVMI(vmiName string) {
	const setupCleanupTimeout = 30 * time.Second

	log.Printf("setup failed, cleanup VMI '%s/%s'", c.namespace, vmiName)
	delCtx, cancel := context.WithTimeout(context.Background(), setupCleanupTimeout)
	defer cancel()
	_ = vmi.Delete(delCtx, c.client, c.namespace, vmiName)
	if derr := vmi.WaitForVmiDispose(delCtx, c.client, c.namespace, vmiName); derr != nil {
		log.Printf("Failed to cleanup VMI '%s/%s': %v", c.namespace, vmiName, derr)
	}
}

func newLatencyCheckVmi(
	uid, name, nodeName, ownerName, ownerUID string,
	netAttachDef *netattdefv1.NetworkAttachmentDefinition) *kvcorev1.VirtualMachineInstance {
	const networkName = "net0"

	vmLabel := vmi.Label{Key: LabelLatencyCheckUID, Value: uid}
	var affinity *k8scorev1.Affinity
	if nodeName != "" {
		affinity = &k8scorev1.Affinity{NodeAffinity: vmi.NewNodeAffinity(nodeName)}
	} else {
		affinity = &k8scorev1.Affinity{PodAntiAffinity: vmi.NewPodAntiAffinity(vmLabel)}
	}

	macAddress := vmi.RandomMACAddress()
	return vmi.NewAlpine(name,
		vmi.WithOwnerReference(ownerName, ownerUID),
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
				vmi.WithAddresses(vmi.RandomIPAddress()),
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

func (c *checkup) Teardown(ctx context.Context) error {
	const errMessagePrefix = "teardown"

	var teardownErrors []string
	if err := vmi.Delete(ctx, c.client, c.namespace, c.sourceVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.sourceVM.Name, err))
	}

	if err := vmi.Delete(ctx, c.client, c.namespace, c.targetVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.targetVM.Name, err))
	}

	if err := vmi.WaitForVmiDispose(ctx, c.client, c.namespace, c.sourceVM.Name); err != nil {
		teardownErrors = append(teardownErrors, fmt.Sprintf("'%s/%s': %v", c.namespace, c.sourceVM.Name, err))
	}

	if err := vmi.WaitForVmiDispose(ctx, c.client, c.namespace, c.targetVM.Name); err != nil {
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

func randomizeName(prefix string) string {
	const randomStringLen = 5

	return fmt.Sprintf("%s-%s", prefix, k8srand.String(randomStringLen))
}
