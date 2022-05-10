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
	"encoding/json"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	kiagnosev1alpha1 "github.com/kiagnose/kiagnose/api/v1alpha1"
	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
)

const (
	imageName = "registry:5000/echo-checkup:latest"
)

var (
	timeoutValue = metav1.Duration{Duration: time.Minute}
	checkupKey   = types.NamespacedName{
		Namespace: "kiagnose",
		Name:      "cm1",
	}
	marshaledData        = json.RawMessage(`{"message1": "value1", "message2": "value2"}`)
	unmarshaledData      = map[string]json.RawMessage{"message1": json.RawMessage(`"value1"`), "message2": json.RawMessage(`"value2"`)}
	clusterRoleNamesList = []string{"cluster_role1", "cluster_role2"}
	roleNamesList        = []string{"default/role1", "default/role2"}
)

func TestReadFromCRShouldSucceed(t *testing.T) {
	type loadTestCase struct {
		description    string
		clusterRoles   []*rbacv1.ClusterRole
		roles          []*rbacv1.Role
		cr             kiagnosev1alpha1.Checkup
		expectedConfig *config.Config
	}

	testCases := []loadTestCase{
		{
			description: "when supplied with required parameters only",
			cr: kiagnosev1alpha1.Checkup{
				Spec: kiagnosev1alpha1.CheckupSpec{
					Image:   imageName,
					Timeout: timeoutValue,
				},
			},
			expectedConfig: &config.Config{
				Image:   imageName,
				Timeout: timeoutValue.Duration,
				Data:    map[string]json.RawMessage{},
			},
		},
		{
			description:  "when supplied with all parameters and data as YAML",
			clusterRoles: expectedClusterRoles(),
			roles:        expectedRoles(),
			cr: kiagnosev1alpha1.Checkup{
				Spec: kiagnosev1alpha1.CheckupSpec{
					Image:            imageName,
					Timeout:          timeoutValue,
					Data:             marshaledData,
					ClusterRoleNames: clusterRoleNamesList,
					RoleNames:        roleNamesList,
				},
			},
			expectedConfig: &config.Config{
				Image:        imageName,
				Timeout:      timeoutValue.Duration,
				Data:         unmarshaledData,
				ClusterRoles: expectedClusterRoles(),
				Roles:        expectedRoles(),
			},
		},
		{
			description:  "when supplied with all parameters and data as JSON",
			clusterRoles: expectedClusterRoles(),
			roles:        expectedRoles(),
			cr: kiagnosev1alpha1.Checkup{
				Spec: kiagnosev1alpha1.CheckupSpec{
					Image:            imageName,
					Timeout:          timeoutValue,
					Data:             marshaledData,
					ClusterRoleNames: clusterRoleNamesList,
					RoleNames:        roleNamesList,
				},
			},
			expectedConfig: &config.Config{
				Image:        imageName,
				Timeout:      timeoutValue.Duration,
				Data:         unmarshaledData,
				ClusterRoles: expectedClusterRoles(),
				Roles:        expectedRoles(),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			k8sClient, crClient := newFakeClientsWithObjects(t, checkupKey, &testCase.cr, testCase.clusterRoles, testCase.roles)
			actualConfig, err := config.ReadFromCR(k8sClient, crClient, checkupKey)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedConfig, actualConfig)
		})
	}
}

func TestReadFromCRShouldFail(t *testing.T) {
	t.Run("when ConfigMap doesn't exist", func(t *testing.T) {
		k8sClient, crClient := newFakeClientsWithObjects(t,
			types.NamespacedName{Namespace: "foo", Name: "bar"}, &kiagnosev1alpha1.Checkup{}, nil, nil)
		_, err := config.ReadFromCR(k8sClient, crClient, checkupKey)
		assert.ErrorContains(t, err, "not found")
	})

	type loadFailureTestCase struct {
		description   string
		cr            kiagnosev1alpha1.Checkup
		expectedError string
	}

	failureTestCases := []loadFailureTestCase{
		{
			description: "when ClusterRole doesn't exist",
			cr: kiagnosev1alpha1.Checkup{Spec: kiagnosev1alpha1.CheckupSpec{
				Image: imageName, Timeout: timeoutValue, ClusterRoleNames: []string{"NA"},
			}},
			expectedError: "clusterroles.rbac.authorization.k8s.io",
		},
		{
			description: "when Role doesn't exist",
			cr: kiagnosev1alpha1.Checkup{Spec: kiagnosev1alpha1.CheckupSpec{
				Image: imageName, Timeout: timeoutValue, RoleNames: []string{"default/role999"},
			}},
			expectedError: "roles.rbac.authorization.k8s.io",
		},
		{
			description: "when Role name is illegal",
			cr: kiagnosev1alpha1.Checkup{Spec: kiagnosev1alpha1.CheckupSpec{
				Image: imageName, Timeout: timeoutValue, RoleNames: []string{"illegal name"},
			}},
			expectedError: "role name",
		},
	}

	for _, testCase := range failureTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			k8sClient, crClient := newFakeClientsWithObjects(t, checkupKey, &testCase.cr, nil, nil)

			_, err := config.ReadFromCR(k8sClient, crClient, checkupKey)
			assert.ErrorContains(t, err, testCase.expectedError)
		})
	}
}

func newFakeClientsWithObjects(t *testing.T, key types.NamespacedName,
	checkup *kiagnosev1alpha1.Checkup, clusterRoles []*rbacv1.ClusterRole, roles []*rbacv1.Role) (*k8sfake.Clientset, client.Client) {
	k8sObjects := []runtime.Object{}
	for _, role := range roles {
		k8sObjects = append(k8sObjects, role)
	}
	for _, clusterRole := range clusterRoles {
		k8sObjects = append(k8sObjects, clusterRole)
	}
	k8sClient := k8sfake.NewSimpleClientset(k8sObjects...)

	scheme := runtime.NewScheme()
	err := kiagnosev1alpha1.AddToScheme(scheme)
	assert.NoError(t, err)
	checkup.Namespace = key.Namespace
	checkup.Name = key.Name
	crClient := crfake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(checkup).Build()
	return k8sClient, crClient
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
