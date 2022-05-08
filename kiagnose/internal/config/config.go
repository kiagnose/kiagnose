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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/client-go/kubernetes"

	"github.com/kiagnose/kiagnose/kiagnose/internal/rbac"
)

type Config struct {
	Image        string
	Timeout      time.Duration
	EnvVars      []corev1.EnvVar
	ClusterRoles []*rbacv1.ClusterRole
	Roles        []*rbacv1.Role
}

func ReadFromConfigMap(client kubernetes.Interface, configMap *corev1.ConfigMap) (*Config, error) {
	parser := newConfigMapParser(configMap.Data)

	if err := parser.Parse(); err != nil {
		return nil, err
	}

	clusterRoles, err := rbac.GetClusterRoles(client, parser.ClusterRoleNames())
	if err != nil {
		return nil, err
	}

	roles, err := rbac.GetRoles(client, parser.RoleNames())
	if err != nil {
		return nil, err
	}

	return &Config{
		Image:        parser.Image(),
		Timeout:      parser.Timeout(),
		EnvVars:      paramsToEnvVars(parser.Params()),
		ClusterRoles: clusterRoles,
		Roles:        roles,
	}, nil
}

func paramsToEnvVars(params map[string]string) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	for k, v := range params {
		envVars = append(envVars, corev1.EnvVar{
			Name:  strings.ToUpper(k),
			Value: v,
		})
	}

	return envVars
}
