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

	k8scorev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

type KubevirtVmisClient interface {
	GetVirtualMachineInstance(namespace, name string) (*kvcorev1.VirtualMachineInstance, error)
	CreateVirtualMachineInstance(namespace string, vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error)
	DeleteVirtualMachineInstance(namespace, name string) error
	SerialConsole(namespace, vmiName string, timeout time.Duration) (kubecli.StreamInterface, error)
}

func Start(c KubevirtVmisClient, namespace string, vmi *kvcorev1.VirtualMachineInstance) error {
	log.Printf("starting VMI %s/%s..", namespace, vmi.Name)
	if _, err := c.CreateVirtualMachineInstance(namespace, vmi); err != nil {
		return fmt.Errorf("failed to start VMI %s/%s: %v", vmi.Namespace, vmi.Name, err)
	}
	return nil
}

func WaitUntilReady(ctx context.Context, c KubevirtVmisClient, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	log.Printf("waiting for VMI %s/%s to be ready..\n", namespace, name)

	return waitForVmiCondition(ctx, c, namespace, name, kvcorev1.VirtualMachineInstanceAgentConnected)
}

func waitForVmiCondition(ctx context.Context, c KubevirtVmisClient, namespace, name string,
	conditionType kvcorev1.VirtualMachineInstanceConditionType) (*kvcorev1.VirtualMachineInstance, error) {
	var updatedVMI *kvcorev1.VirtualMachineInstance

	conditionFn := func(ctx context.Context) (bool, error) {
		var err error
		updatedVMI, err = c.GetVirtualMachineInstance(namespace, name)
		if err != nil {
			return false, nil
		}
		for _, condition := range updatedVMI.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8scorev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	}
	const interval = time.Second * 5
	if err := wait.PollImmediateUntilWithContext(ctx, interval, conditionFn); err != nil {
		return nil, fmt.Errorf("failed to wait for VMI '%s/%s' condition %q: %v", namespace, name, conditionType, err)
	}

	return updatedVMI, nil
}

func Delete(c KubevirtVmisClient, namespace, name string) error {
	log.Printf("deleting VMI %s/%s..\n", namespace, name)

	if err := c.DeleteVirtualMachineInstance(namespace, name); err != nil {
		return fmt.Errorf("failed to delete VMI %s/%s: %v", namespace, name, err)
	}
	return nil
}

func WaitForVmiDispose(ctx context.Context, c KubevirtVmisClient, namespace, name string) error {
	log.Printf("waiting for VMI %s/%s to dispose..\n", namespace, name)

	conditionFn := func(ctx context.Context) (bool, error) {
		_, err := c.GetVirtualMachineInstance(namespace, name)
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
