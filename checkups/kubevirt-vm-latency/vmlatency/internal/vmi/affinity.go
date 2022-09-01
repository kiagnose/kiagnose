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
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvcorev1 "kubevirt.io/api/core/v1"
)

// WithAffinity adds the given affinity.
func WithAffinity(affinity *k8scorev1.Affinity) Option {
	return func(vmi *kvcorev1.VirtualMachineInstance) {
		if affinity != nil {
			vmi.Spec.Affinity = affinity
		}
	}
}

// NewNodeAffinity returns new node affinity with node selector of the given node name.
// Adding it to a VMI will make sure it will schedule on the given node name.
func NewNodeAffinity(nodeName string) *k8scorev1.NodeAffinity {
	req := k8scorev1.NodeSelectorRequirement{
		Key:      k8scorev1.LabelHostname,
		Operator: k8scorev1.NodeSelectorOpIn,
		Values:   []string{nodeName},
	}
	term := []k8scorev1.NodeSelectorTerm{
		{
			MatchExpressions: []k8scorev1.NodeSelectorRequirement{req},
		},
	}
	return &k8scorev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8scorev1.NodeSelector{
			NodeSelectorTerms: term,
		},
	}
}

// NewPodAntiAffinity returns new pod anti-affinity with label selector of the given label key and value.
// Adding it to a VMI will make sure it won't schedule on the same node as other VMIs with the given label.
func NewPodAntiAffinity(label Label) *k8scorev1.PodAntiAffinity {
	req := k8smetav1.LabelSelectorRequirement{
		Operator: k8smetav1.LabelSelectorOpIn,
		Key:      label.Key,
		Values:   []string{label.Value},
	}
	labelSelector := &k8smetav1.LabelSelector{
		MatchExpressions: []k8smetav1.LabelSelectorRequirement{req},
	}
	term := k8scorev1.PodAffinityTerm{
		TopologyKey:   k8scorev1.LabelHostname,
		LabelSelector: labelSelector,
	}
	return &k8scorev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []k8scorev1.PodAffinityTerm{term},
	}
}
