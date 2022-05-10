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

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckupSpec defines a checkup to run
type CheckupSpec struct {
	// +kubebuilder:validation:Required
	Image string `json:"image"`
	// +kubebuilder:validation:Required
	Timeout metav1.Duration `json:"timeout"`
	// +kubebuilder:validation:XPreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Data             json.RawMessage `json:"data,omitempty"`
	ClusterRoleNames []string        `json:"clusterRoleNames,omitempty"`
	RoleNames        []string        `json:"roleNames,omitempty"`
}

// CheckupStatus defines the observed state of Checkup
type CheckupStatus struct {
	Conditions ConditionList `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=checkups,singular=checkup,shortName=ckup
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type",description="Status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].reason",description="Reason"
// +kubebuilder:storageversion
// +kubebuilder:subresource=status

// Checkup is the Schema for the kiagnose checkups API
type Checkup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CheckupSpec   `json:"spec,omitempty"`
	Status            CheckupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CheckupList contains a list of Checkup
type CheckupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Checkup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Checkup{}, &CheckupList{})
}
