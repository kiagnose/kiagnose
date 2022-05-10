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

package config

import (
	"context"
	"encoding/json"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kiagnosev1alpha1 "github.com/kiagnose/kiagnose/api/v1alpha1"
	"github.com/kiagnose/kiagnose/kiagnose/internal/rbac"
)

type Config struct {
	Image        string
	Timeout      time.Duration
	Data         map[string]json.RawMessage
	ClusterRoles []*rbacv1.ClusterRole
	Roles        []*rbacv1.Role
}

func ReadFromCR(k8sClient kubernetes.Interface, crClient client.Client, key types.NamespacedName) (*Config, error) {
	checkup := &kiagnosev1alpha1.Checkup{}
	if err := crClient.Get(context.Background(), key, checkup); err != nil {
		return nil, err
	}

	clusterRoles, err := rbac.GetClusterRoles(k8sClient, checkup.Spec.ClusterRoleNames)
	if err != nil {
		return nil, err
	}

	roles, err := rbac.GetRoles(k8sClient, checkup.Spec.RoleNames)
	if err != nil {
		return nil, err
	}

	data := map[string]json.RawMessage{}
	if len(checkup.Spec.Data) > 0 {
		err = json.Unmarshal(checkup.Spec.Data, &data)
		if err != nil {
			return nil, err
		}
	}
	return &Config{
		Image:        checkup.Spec.Image,
		Timeout:      checkup.Spec.Timeout.Duration,
		Data:         data,
		ClusterRoles: clusterRoles,
		Roles:        roles,
	}, nil
}
