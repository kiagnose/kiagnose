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

package rbac

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

func CreateServiceAccount(client corev1client.CoreV1Interface, sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	createdSa, err := client.ServiceAccounts(sa.Namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("ServiceAccount '%s/%s' successfully created", sa.Namespace, sa.Name)

	return createdSa, nil
}

// CreateClusterRoleBindings creates the given ClusterRoleBindings in the cluster.
// In case of failure it will delete and waits for the ClusterRoleBindings to dispose.
func CreateClusterRoleBindings(client rbacv1client.RbacV1Interface, clusterRoleBindings []*rbacv1.ClusterRoleBinding,
	timeout time.Duration) ([]*rbacv1.ClusterRoleBinding, error) {
	var createdClusterRoleBindings []*rbacv1.ClusterRoleBinding
	var createErr error
	for _, clusterRoleBinding := range clusterRoleBindings {
		createdClusterRoleBinding, err := createClusterRoleBinding(client, clusterRoleBinding)
		if err != nil {
			createErr = err
			break
		}
		createdClusterRoleBindings = append(createdClusterRoleBindings, createdClusterRoleBinding)
	}

	if createErr != nil {
		createErrMsg := fmt.Sprintf("failed for create ClusterRoleBindings: %v", createErr)
		if deleteErr := DeleteClusterRoleBindings(client, createdClusterRoleBindings, timeout); deleteErr != nil {
			return nil, fmt.Errorf("%s, clean up failed: %v", createErrMsg, deleteErr)
		}
		return nil, errors.New(createErrMsg)
	}

	return createdClusterRoleBindings, nil
}

func createClusterRoleBinding(c rbacv1client.RbacV1Interface, bindings *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	createdClusterRoleBinding, err := c.ClusterRoleBindings().Create(context.Background(), bindings, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("ClusterRoleBinding %q successfully created\n", createdClusterRoleBinding.Name)

	return createdClusterRoleBinding, nil
}

// DeleteClusterRoleBindings delete and waits for the given ClusterRoleBindings to dispose.
func DeleteClusterRoleBindings(client rbacv1client.RbacV1Interface, clusterRoleBindings []*rbacv1.ClusterRoleBinding,
	timeout time.Duration) error {
	var danglingClusterRoleBindings []string
	var deletedClusterRoleBindings []string

	for _, clusterRoleBinding := range clusterRoleBindings {
		if err := deleteClusterRoleBinding(client, clusterRoleBinding.Name); err != nil {
			danglingClusterRoleBindings = append(danglingClusterRoleBindings, fmt.Sprintf("name: %s reasone: %v", clusterRoleBinding.Name, err))
			continue
		}
		deletedClusterRoleBindings = append(deletedClusterRoleBindings, clusterRoleBinding.Name)
	}

	for _, deletedClusterRoleBinding := range deletedClusterRoleBindings {
		if err := waitForClusterRoleBindingDeletion(client, deletedClusterRoleBinding, timeout); err != nil {
			danglingClusterRoleBindings = append(danglingClusterRoleBindings, fmt.Sprintf("name: %s reasone: %v", deletedClusterRoleBinding, err))
			continue
		}
	}

	if len(danglingClusterRoleBindings) > 0 {
		return fmt.Errorf("failed to delete ClusterRoleBindings: %s", strings.Join(danglingClusterRoleBindings, ", "))
	}

	return nil
}

func deleteClusterRoleBinding(client rbacv1client.RbacV1Interface, clusterRoleBindingName string) error {
	if err := client.ClusterRoleBindings().Delete(context.Background(), clusterRoleBindingName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	log.Printf("delete ClusterRoleBinding %q request sent", clusterRoleBindingName)
	return nil
}

// waitForClusterRoleBindingDeletion waits until the given ClusterRoleBinding is disposed.
func waitForClusterRoleBindingDeletion(client rbacv1client.RbacV1Interface, name string, timeout time.Duration) error {
	log.Printf("waiting for ClusterRoleBinding %q to dispose", name)

	const pollInterval = time.Second * 5
	conditionFn := func() (bool, error) {
		_, err := client.ClusterRoleBindings().Get(context.Background(), name, metav1.GetOptions{})
		custerRoleBindingNotFound := k8serrors.IsNotFound(err)
		if err != nil && !custerRoleBindingNotFound {
			log.Printf("failed to get ClusterRoleBinding %q while waiting for it to dispose: %v", name, err)
		}
		return custerRoleBindingNotFound, nil
	}
	if err := wait.PollImmediate(pollInterval, timeout, conditionFn); err != nil {
		return err
	}

	log.Printf("ClusterRoleBinding %q successfully deleted", name)
	return nil
}

func CreateRoles(client rbacv1client.RbacV1Interface, roles []*rbacv1.Role) ([]*rbacv1.Role, error) {
	var createdRoles []*rbacv1.Role
	for _, role := range roles {
		createRole, err := createRole(client, role)
		if err != nil {
			return nil, err
		}
		createdRoles = append(createdRoles, createRole)
	}

	return createdRoles, nil
}

func createRole(client rbacv1client.RbacV1Interface, role *rbacv1.Role) (*rbacv1.Role, error) {
	createdRole, err := client.Roles(role.Namespace).Create(context.Background(), role, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Role '%s/%s' successfully created\n", createdRole.Namespace, createdRole.Name)

	return createdRole, nil
}

func CreateRoleBindings(client rbacv1client.RbacV1Interface, bindings []*rbacv1.RoleBinding) ([]*rbacv1.RoleBinding, error) {
	var createdRoleBindings []*rbacv1.RoleBinding
	for _, roleBinding := range bindings {
		createdBinding, err := createRoleBinding(client, roleBinding)
		if err != nil {
			return nil, err
		}
		createdRoleBindings = append(createdRoleBindings, createdBinding)
	}

	return createdRoleBindings, nil
}

func createRoleBinding(client rbacv1client.RbacV1Interface, crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	createdRb, err := client.RoleBindings(crb.Namespace).Create(context.Background(), crb, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("RoleBinding '%s/%s' successfully created\n", createdRb.Namespace, createdRb.Name)

	return createdRb, nil
}
