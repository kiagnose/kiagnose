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

package config_test

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
	"github.com/kiagnose/kiagnose/kiagnose/types"
)

const (
	configMapNamespace = "target-ns"
	configMapName      = "cm1"

	imageName               = "registry:5000/echo-checkup:latest"
	timeoutValue            = "1m"
	serviceAccountNameValue = "test-sa"
	param1Key               = "message1"
	param1Value             = "message1 value"
	param2Key               = "message2"
	param2Value             = "message2 value"
)

var (
	clusterRoleNamesList = []string{"cluster_role1", "cluster_role2"}
	roleNamesList        = []string{"default/role1", "default/role2"}
)

func TestReadFromConfigMapShouldSucceed(t *testing.T) {
	type loadTestCase struct {
		description    string
		clusterRoles   []*rbacv1.ClusterRole
		roles          []*rbacv1.Role
		configMapData  map[string]string
		expectedConfig *config.Config
	}

	testCases := []loadTestCase{
		{
			description: "when supplied with required parameters only",
			configMapData: map[string]string{
				types.ImageKey:              imageName,
				types.TimeoutKey:            timeoutValue,
				types.ServiceAccountNameKey: serviceAccountNameValue,
			},
			expectedConfig: &config.Config{
				Image:              imageName,
				Timeout:            stringToDurationMustParse(timeoutValue),
				ServiceAccountName: serviceAccountNameValue,
			},
		},
		{
			description:  "when supplied with all parameters",
			clusterRoles: expectedClusterRoles(),
			roles:        expectedRoles(),
			configMapData: map[string]string{
				types.ImageKey:                       imageName,
				types.TimeoutKey:                     timeoutValue,
				types.ServiceAccountNameKey:          serviceAccountNameValue,
				types.ParamNameKeyPrefix + param1Key: param1Value,
				types.ParamNameKeyPrefix + param2Key: param2Value,
				types.ClusterRolesKey:                strings.Join(clusterRoleNamesList, "\n"),
				types.RolesKey:                       strings.Join(roleNamesList, "\n"),
			},
			expectedConfig: &config.Config{
				Image:              imageName,
				Timeout:            stringToDurationMustParse(timeoutValue),
				ServiceAccountName: serviceAccountNameValue,
				EnvVars:            expectedEnvVars(param1Key, param1Value, param2Key, param2Value),
				ClusterRoles:       expectedClusterRoles(),
				Roles:              expectedRoles(),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := newFakeClientWithObjects(configMapNamespace, configMapName, testCase.configMapData, testCase.clusterRoles, testCase.roles)

			actualConfig, err := config.ReadFromConfigMap(fakeClient, configMapNamespace, configMapName)
			assert.NoError(t, err)

			sort.Slice(actualConfig.EnvVars, func(i, j int) bool {
				return actualConfig.EnvVars[i].Name < actualConfig.EnvVars[j].Name
			})

			assert.Equal(t, testCase.expectedConfig, actualConfig)
		})
	}
}

func TestReadFromConfigMapShouldFail(t *testing.T) {
	t.Run("when ConfigMap doesn't exist", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		_, err := config.ReadFromConfigMap(fakeClient, configMapNamespace, configMapName)
		assert.ErrorContains(t, err, "not found")
	})

	type loadFailureTestCase struct {
		description   string
		configMapData map[string]string
		expectedError string
	}

	const emptyParamName = ""

	failureTestCases := []loadFailureTestCase{
		{
			description: "when ConfigMap is already in use",
			configMapData: map[string]string{
				types.ImageKey:          imageName,
				types.TimeoutKey:        timeoutValue,
				types.StartTimestampKey: time.Now().Format(time.RFC3339),
			},
			expectedError: config.ErrConfigMapIsAlreadyInUse.Error(),
		},
		{
			description: "when ConfigMap is already in use (startTimestamp exists but empty)",
			configMapData: map[string]string{
				types.ImageKey:          imageName,
				types.TimeoutKey:        timeoutValue,
				types.StartTimestampKey: "",
			},
			expectedError: config.ErrConfigMapIsAlreadyInUse.Error(),
		},
		{
			description:   "when image field is missing",
			configMapData: map[string]string{types.TimeoutKey: timeoutValue},
			expectedError: config.ErrImageFieldIsMissing.Error(),
		},
		{
			description:   "when image field value is empty",
			configMapData: map[string]string{types.ImageKey: "", types.TimeoutKey: timeoutValue},
			expectedError: config.ErrImageFieldIsIllegal.Error(),
		},
		{
			description:   "when timout field is missing",
			configMapData: map[string]string{types.ImageKey: imageName},
			expectedError: config.ErrTimeoutFieldIsMissing.Error(),
		},
		{
			description:   "when timout field is illegal",
			configMapData: map[string]string{types.ImageKey: imageName, types.TimeoutKey: "illegalValue"},
			expectedError: config.ErrTimeoutFieldIsIllegal.Error(),
		},
		{
			description:   "when ConfigMap Data is nil",
			configMapData: nil,
			expectedError: config.ErrConfigMapDataIsNil.Error(),
		},
		{
			description:   "when ClusterRole doesn't exist",
			configMapData: map[string]string{types.ImageKey: imageName, types.TimeoutKey: timeoutValue, types.ClusterRolesKey: "NA\n"},
			expectedError: "clusterroles.rbac.authorization.k8s.io",
		},
		{
			description:   "when Role doesn't exist",
			configMapData: map[string]string{types.ImageKey: imageName, types.TimeoutKey: timeoutValue, types.RolesKey: "default/role999\n"},
			expectedError: "roles.rbac.authorization.k8s.io",
		},
		{
			description:   "when Role name is illegal",
			configMapData: map[string]string{types.ImageKey: imageName, types.TimeoutKey: timeoutValue, types.RolesKey: "illegal name\n"},
			expectedError: "role name",
		},
		{
			description: "when param name is empty",
			configMapData: map[string]string{
				types.ImageKey:   imageName,
				types.TimeoutKey: timeoutValue,
				types.ParamNameKeyPrefix + emptyParamName: "some value",
			},
			expectedError: config.ErrParamNameIsIllegal.Error(),
		},
	}

	for _, testCase := range failureTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(newConfigMap(configMapNamespace, configMapName, testCase.configMapData))

			_, err := config.ReadFromConfigMap(fakeClient, configMapNamespace, configMapName)
			assert.ErrorContains(t, err, testCase.expectedError)
		})
	}
}

func newFakeClientWithObjects(namespace, name string,
	configMapData map[string]string, clusterRoles []*rbacv1.ClusterRole, roles []*rbacv1.Role) *fake.Clientset {
	client := fake.NewSimpleClientset(newConfigMap(namespace, name, configMapData))

	for _, clusterRole := range clusterRoles {
		_, err := client.RbacV1().ClusterRoles().Create(context.Background(), clusterRole, metav1.CreateOptions{})
		if err != nil {
			panic("failed to create ClusterRole")
		}
	}

	for _, role := range roles {
		_, err := client.RbacV1().Roles(role.Namespace).Create(context.Background(), role, metav1.CreateOptions{})
		if err != nil {
			panic("failed to create Role")
		}
	}

	return client
}

func newConfigMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func expectedEnvVars(param1Key, param1Value, param2Key, param2Value string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: strings.ToUpper(param1Key), Value: param1Value},
		{Name: strings.ToUpper(param2Key), Value: param2Value},
	}
}

func expectedClusterRoles() []*rbacv1.ClusterRole {
	return []*rbacv1.ClusterRole{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "ClusterRole", APIVersion: rbacv1.GroupName},
			ObjectMeta: metav1.ObjectMeta{Name: "cluster_role1"},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "ClusterRole", APIVersion: rbacv1.GroupName},
			ObjectMeta: metav1.ObjectMeta{Name: "cluster_role2"},
		},
	}
}

func expectedRoles() []*rbacv1.Role {
	return []*rbacv1.Role{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Role", APIVersion: rbacv1.GroupName},
			ObjectMeta: metav1.ObjectMeta{Name: "role1", Namespace: "default"},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Role", APIVersion: rbacv1.GroupName},
			ObjectMeta: metav1.ObjectMeta{Name: "role2", Namespace: "default"},
		},
	}
}

func stringToDurationMustParse(rawDuration string) time.Duration {
	duration, err := time.ParseDuration(rawDuration)
	if err != nil {
		panic("Bad duration")
	}

	return duration
}
