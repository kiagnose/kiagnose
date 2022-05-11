/*
 *
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

// Package vmi source: https://github.com/kubevirt/kubevirt/blob/v0.53.0/tests/libnet
package vmi

import (
	"fmt"

	"sigs.k8s.io/yaml"
)

type cloudInitNetworkData struct {
	Version   int                           `json:"version"`
	Ethernets map[string]cloudInitInterface `json:"ethernets,omitempty"`
}

type cloudInitInterface struct {
	name           string
	AcceptRA       *bool                `json:"accept-ra,omitempty"`
	Addresses      []string             `json:"addresses,omitempty"`
	DHCP4          *bool                `json:"dhcp4,omitempty"`
	DHCP6          *bool                `json:"dhcp6,omitempty"`
	DHCPIdentifier string               `json:"dhcp-identifier,omitempty"` // "duid" or  "mac"
	Gateway4       string               `json:"gateway4,omitempty"`
	Gateway6       string               `json:"gateway6,omitempty"`
	Nameservers    cloudInitNameservers `json:"nameservers,omitempty"`
	MACAddress     string               `json:"macaddress,omitempty"`
	Match          cloudInitMatch       `json:"match,omitempty"`
	MTU            int                  `json:"mtu,omitempty"`
	Routes         []cloudInitRoute     `json:"routes,omitempty"`
	SetName        string               `json:"set-name,omitempty"`
}

type cloudInitNameservers struct {
	Search    []string `json:"search,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
}

type cloudInitMatch struct {
	Name       string `json:"name,omitempty"`
	MACAddress string `json:"macaddress,omitempty"`
	Driver     string `json:"driver,omitempty"`
}

type cloudInitRoute struct {
	From   string `json:"from,omitempty"`
	OnLink *bool  `json:"on-link,omitempty"`
	Scope  string `json:"scope,omitempty"`
	Table  *int   `json:"table,omitempty"`
	To     string `json:"to,omitempty"`
	Type   string `json:"type,omitempty"`
	Via    string `json:"via,omitempty"`
	Metric *int   `json:"metric,omitempty"`
}

type networkDataOption func(*cloudInitNetworkData) error
type networkDataInterfaceOption func(*cloudInitInterface) error

func NewNetworkData(options ...networkDataOption) (string, error) {
	networkData := cloudInitNetworkData{
		Version: 2,
	}

	for _, option := range options {
		err := option(&networkData)
		if err != nil {
			return "", fmt.Errorf("failed defining network data when running options: %w", err)
		}
	}

	nd, err := yaml.Marshal(&networkData)
	if err != nil {
		return "", err
	}

	return string(nd), nil
}

func WithEthernet(name string, options ...networkDataInterfaceOption) networkDataOption {
	return func(networkData *cloudInitNetworkData) error {
		if networkData.Ethernets == nil {
			networkData.Ethernets = map[string]cloudInitInterface{}
		}

		networkDataInterface := cloudInitInterface{name: name}

		for _, option := range options {
			err := option(&networkDataInterface)
			if err != nil {
				return fmt.Errorf("failed defining network data ethernet device when running options: %w", err)
			}
		}

		networkData.Ethernets[name] = networkDataInterface
		return nil
	}
}

func WithAddresses(addresses ...string) networkDataInterfaceOption {
	return func(networkDataInterface *cloudInitInterface) error {
		networkDataInterface.Addresses = append(networkDataInterface.Addresses, addresses...)
		return nil
	}
}

func WithMatchingMAC(macAddress string) networkDataInterfaceOption {
	return func(networkDataInterface *cloudInitInterface) error {
		networkDataInterface.Match = cloudInitMatch{
			MACAddress: macAddress,
		}
		networkDataInterface.SetName = networkDataInterface.name
		return nil
	}
}
