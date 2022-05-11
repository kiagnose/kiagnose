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

package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

type Client struct{ kubecli.KubevirtClient }

func New() (*Client, error) {
	c, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	return &Client{c}, nil
}

func (c *Client) UpdateConfigMap(namespace, name string, date map[string]string) error {
	cm, err := c.KubevirtClient.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cm.Data = date

	if _, err := c.KubevirtClient.CoreV1().ConfigMaps(namespace).Update(context.Background(), cm, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) GetVirtualMachineInstance(namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
}

func (c *Client) CreateVirtualMachineInstance(
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Create(vmi)
}
