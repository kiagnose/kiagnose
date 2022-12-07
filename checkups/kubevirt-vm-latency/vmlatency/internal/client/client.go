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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netattdefclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/typed/k8s.cni.cncf.io/v1"
)

type Client struct {
	kubecli.KubevirtClient
	netattdefclient.K8sCniCncfIoV1Interface
}

func New() (*Client, error) {
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	c, err := kubecli.GetKubevirtClientFromRESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	cniClient, err := netattdefclient.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &Client{c, cniClient}, nil
}

func (c *Client) GetVirtualMachineInstance(ctx context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	type resultWrapper struct {
		vmi *kvcorev1.VirtualMachineInstance
		err error
	}

	resultCh := make(chan resultWrapper, 1)

	go func() {
		vmi, err := c.KubevirtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
		resultCh <- resultWrapper{vmi, err}
	}()

	select {
	case result := <-resultCh:
		return result.vmi, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) CreateVirtualMachineInstance(
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Create(vmi)
}

func (c *Client) DeleteVirtualMachineInstance(namespace, name string) error {
	return c.KubevirtClient.VirtualMachineInstance(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *Client) SerialConsole(namespace, vmiName string, timeout time.Duration) (kubecli.StreamInterface, error) {
	return c.KubevirtClient.VirtualMachineInstance(namespace).SerialConsole(vmiName, &kubecli.SerialConsoleOptions{ConnectionTimeout: timeout})
}

func (c *Client) GetNetworkAttachmentDefinition(namespace, name string) (*netattdefv1.NetworkAttachmentDefinition, error) {
	return c.K8sCniCncfIoV1Interface.NetworkAttachmentDefinitions(namespace).Get(context.Background(), name, metav1.GetOptions{})
}
