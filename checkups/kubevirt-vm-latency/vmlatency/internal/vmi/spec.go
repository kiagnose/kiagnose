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

// Package vmi source https://github.com/kubevirt/kubevirt/tree/v0.53.0/tests/libvmi
package vmi

import (
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvcorev1 "kubevirt.io/api/core/v1"
)

// Option represents an action that enables an option.
type Option func(vmi *kvcorev1.VirtualMachineInstance)

// newBaseVmi instantiates a new VMI configuration,
// building its properties based on the specified With* options.
func newBaseVmi(name string, opts ...Option) *kvcorev1.VirtualMachineInstance {
	vmi := &kvcorev1.VirtualMachineInstance{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       kvcorev1.VirtualMachineInstanceGroupVersionKind.Kind,
			APIVersion: kvcorev1.GroupVersion.String(),
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: name,
		},
		Spec: kvcorev1.VirtualMachineInstanceSpec{},
	}

	for _, f := range opts {
		f(vmi)
	}

	return vmi
}

// withTerminationGracePeriodSecond sets TerminationGracePeriodSecond.
func withTerminationGracePeriodSecond(duration int64) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		vmi.Spec.TerminationGracePeriodSeconds = &duration
	}
}

type Label struct {
	Key, Value string
}

// WithLabels adds the given labels.
func WithLabels(labels ...Label) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		if vmi.ObjectMeta.Labels == nil {
			vmi.ObjectMeta.Labels = map[string]string{}
		}

		for _, label := range labels {
			vmi.ObjectMeta.Labels[label.Key] = label.Value
		}
	}
}

// withResourceMemory specifies the vmi memory resource.
func withResourceMemory(value string) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		vmi.Spec.Domain.Resources.Requests = k8scorev1.ResourceList{
			k8scorev1.ResourceMemory: resource.MustParse(value),
		}
	}
}

// withRng adds 'rng' to the vmi devices.
func withRng() Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Rng = &kvcorev1.Rng{}
	}
}

// WithMultusNetwork adds network with Multus network source.
func WithMultusNetwork(networkName, netAttachDefName string) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		vmi.Spec.Networks = append(vmi.Spec.Networks, multusNetwork(networkName, netAttachDefName))
	}
}

func multusNetwork(name, netAttachDefName string) kvcorev1.Network {
	return kvcorev1.Network{
		Name: name,
		NetworkSource: kvcorev1.NetworkSource{
			Multus: &kvcorev1.MultusNetwork{
				NetworkName: netAttachDefName,
			},
		},
	}
}

// WithCloudInitNoCloudNetworkData adds cloud-init no-cloud network data with the given options.
func WithCloudInitNoCloudNetworkData(opts ...networkDataOption) Option {
	networkData, _ := NewNetworkData(opts...)

	return func(vmi *kvcorev1.VirtualMachineInstance) {
		const volumeName = "cloudinit"
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, newCloudinitVolumeWithNetworkData(volumeName, networkData))
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, newDisk(volumeName))
	}
}

func newCloudinitVolumeWithNetworkData(name, networkData string) kvcorev1.Volume {
	return kvcorev1.Volume{
		Name: name,
		VolumeSource: kvcorev1.VolumeSource{
			CloudInitNoCloud: &kvcorev1.CloudInitNoCloudSource{
				NetworkData: networkData,
			},
		},
	}
}

func newDisk(name string) kvcorev1.Disk {
	return kvcorev1.Disk{
		Name: name,
		DiskDevice: kvcorev1.DiskDevice{
			Disk: &kvcorev1.DiskTarget{
				Bus: kvcorev1.DiskBusVirtio,
			},
		},
	}
}

func withContainerDiskImage(name string) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		diskName := "disk0"
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, newDisk(diskName))
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, newContainerDiskVolume(diskName, name))
	}
}

func newContainerDiskVolume(name, image string) kvcorev1.Volume {
	return kvcorev1.Volume{
		Name: name,
		VolumeSource: kvcorev1.VolumeSource{
			ContainerDisk: &kvcorev1.ContainerDiskSource{
				Image: image,
			},
		},
	}
}

// WithInterface adds an interface.
func WithInterface(iface kvcorev1.Interface) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, iface)
	}
}

// interfaceOption represents an action that enables an option on a VMI interface.
type interfaceOption func(vmi *kvcorev1.Interface)

// NewInterface instantiates a new VMI interface configuration,
// building its properties based on the specified With* options.
func NewInterface(name string, opts ...interfaceOption) kvcorev1.Interface {
	iface := &kvcorev1.Interface{Name: name}

	for _, f := range opts {
		f(iface)
	}

	return *iface
}

// WithMacAddress set the interface with custom MAC address.
func WithMacAddress(macAddress string) interfaceOption {
	return func(iface *kvcorev1.Interface) {
		iface.MacAddress = macAddress
	}
}

// WithSriovBinding set the interface with SR-IOV binding method.
func WithSriovBinding() interfaceOption {
	return func(iface *kvcorev1.Interface) {
		iface.InterfaceBindingMethod = kvcorev1.InterfaceBindingMethod{
			SRIOV: &kvcorev1.InterfaceSRIOV{},
		}
	}
}

// WithBridgeBinding set the interface with bridge binding method.
func WithBridgeBinding() interfaceOption {
	return func(iface *kvcorev1.Interface) {
		iface.InterfaceBindingMethod = kvcorev1.InterfaceBindingMethod{
			Bridge: &kvcorev1.InterfaceBridge{},
		}
	}
}

func NewAlpine(name string, opts ...Option) *kvcorev1.VirtualMachineInstance {
	const (
		memory                                     = "128Mi"
		defaultTerminationGracePeriodSeconds int64 = 5

		containerDiskImage = "quay.io/kubevirtci/alpine-with-test-tooling-container-disk" +
			"@sha256:a40c4a7bb9644098740ad5f8aa64040b0a64bb84cc4e3b42d633bb752ab4b9ce"
	)
	latencyCheckOpts := []Option{
		withContainerDiskImage(containerDiskImage),
		withTerminationGracePeriodSecond(defaultTerminationGracePeriodSeconds),
		withResourceMemory(memory),
		withRng(),
	}
	latencyCheckOpts = append(latencyCheckOpts, opts...)

	return newBaseVmi(name, latencyCheckOpts...)
}
