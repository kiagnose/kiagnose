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

package vmi

import (
	"context"
	"fmt"
	"log"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

type KubevirtVmisClient interface {
	GetVirtualMachineInstance(ctx context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error)
	CreateVirtualMachineInstance(
		ctx context.Context,
		namespace string,
		vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error)
	DeleteVirtualMachineInstance(ctx context.Context, namespace, name string) error
	SerialConsole(namespace, vmiName string, timeout time.Duration) (kubecli.StreamInterface, error)
	GetNetworkAttachmentDefinition(ctx context.Context, namespace, name string) (*netattdefv1.NetworkAttachmentDefinition, error)
}

func Start(ctx context.Context, c KubevirtVmisClient, namespace string, vmi *kvcorev1.VirtualMachineInstance) error {
	log.Printf("starting VMI %s/%s..", namespace, vmi.Name)
	if _, err := c.CreateVirtualMachineInstance(ctx, namespace, vmi); err != nil {
		return fmt.Errorf("failed to start VMI %s/%s: %v", vmi.Namespace, vmi.Name, err)
	}
	return nil
}

func WaitForStatusIPAddress(ctx context.Context, c KubevirtVmisClient, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	log.Printf("waiting for VMI %s/%s IP address to appear on status..\n", namespace, name)
	var updatedVMI *kvcorev1.VirtualMachineInstance

	conditionFn := func(ctx context.Context) (bool, error) {
		var err error
		updatedVMI, err = c.GetVirtualMachineInstance(ctx, namespace, name)
		if err != nil {
			return false, nil
		}
		return vmiIPAddressExists(updatedVMI), nil
	}
	const interval = time.Second * 5
	if err := wait.PollImmediateUntilWithContext(ctx, interval, conditionFn); err != nil {
		return nil, fmt.Errorf("failed to wait for VMI '%s/%s' IP address to appear on status: %v", namespace, name, err)
	}

	return updatedVMI, nil
}

func vmiIPAddressExists(vmi *kvcorev1.VirtualMachineInstance) bool {
	return len(vmi.Status.Interfaces) > 0 && vmi.Status.Interfaces[0].IP != ""
}

func Delete(ctx context.Context, c KubevirtVmisClient, namespace, name string) error {
	log.Printf("deleting VMI %s/%s..\n", namespace, name)

	if err := c.DeleteVirtualMachineInstance(ctx, namespace, name); err != nil {
		return fmt.Errorf("failed to delete VMI %s/%s: %v", namespace, name, err)
	}
	return nil
}

func WaitForVmiDispose(ctx context.Context, c KubevirtVmisClient, namespace, name string) error {
	log.Printf("waiting for VMI %s/%s to dispose..\n", namespace, name)

	conditionFn := func(ctx context.Context) (bool, error) {
		_, err := c.GetVirtualMachineInstance(ctx, namespace, name)
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	}
	const interval = time.Second * 5
	if err := wait.PollImmediateUntilWithContext(ctx, interval, conditionFn); err != nil {
		return fmt.Errorf("failed to wait for VMI %s/%s to dispose: %v", namespace, name, err)
	}

	return nil
}
