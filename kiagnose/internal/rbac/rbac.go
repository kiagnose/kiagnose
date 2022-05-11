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

	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// CreateClusterRoleBindings creates the given ClusterRoleBindings in the cluster.
// In case of failure it will delete and waits for the ClusterRoleBindings to dispose.
func CreateClusterRoleBindings(client kubernetes.Interface, clusterRoleBindings []*rbacv1.ClusterRoleBinding,
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

func createClusterRoleBinding(c kubernetes.Interface, bindings *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	createdClusterRoleBinding, err := c.RbacV1().ClusterRoleBindings().Create(context.Background(), bindings, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("ClusterRoleBinding %q successfully created\n", createdClusterRoleBinding.Name)

	return createdClusterRoleBinding, nil
}

func GetClusterRoles(client kubernetes.Interface, clusterRoleNames []string) ([]*rbacv1.ClusterRole, error) {
	var clusterRoles []*rbacv1.ClusterRole

	for _, name := range clusterRoleNames {
		clusterRole, err := getClusterRole(client, name)
		if err != nil {
			return nil, err
		}

		clusterRoles = append(clusterRoles, clusterRole)
	}

	return clusterRoles, nil
}

func GetRoles(client kubernetes.Interface, roleNames []string) ([]*rbacv1.Role, error) {
	const requiredPartsCount = 2
	var roles []*rbacv1.Role

	for _, roleFullName := range roleNames {
		nameParts := strings.Split(roleFullName, "/")
		if len(nameParts) != requiredPartsCount {
			return nil, fmt.Errorf("role name: %q is illeagal", roleFullName)
		}

		roleNamespace := nameParts[0]
		roleName := nameParts[1]

		role, err := getRole(client, roleNamespace, roleName)
		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func getClusterRole(client kubernetes.Interface, name string) (*rbacv1.ClusterRole, error) {
	clusterRole, err := client.RbacV1().ClusterRoles().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return clusterRole, nil
}

func getRole(client kubernetes.Interface, namespace, name string) (*rbacv1.Role, error) {
	role, err := client.RbacV1().Roles(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return role, nil
}

// DeleteClusterRoleBindings delete and waits for the given ClusterRoleBindings to dispose.
func DeleteClusterRoleBindings(client kubernetes.Interface, clusterRoleBindings []*rbacv1.ClusterRoleBinding,
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

func deleteClusterRoleBinding(client kubernetes.Interface, clusterRoleBindingName string) error {
	if err := client.RbacV1().ClusterRoleBindings().Delete(context.Background(), clusterRoleBindingName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	log.Printf("delete ClusterRoleBinding %q request sent", clusterRoleBindingName)
	return nil
}

// waitForClusterRoleBindingDeletion waits until the given ClusterRoleBinding is disposed.
func waitForClusterRoleBindingDeletion(client kubernetes.Interface, name string, timeout time.Duration) error {
	log.Printf("waiting for ClusterRoleBinding %q to dispose", name)

	const pollInterval = time.Second * 5
	conditionFn := func() (bool, error) {
		_, err := client.RbacV1().ClusterRoleBindings().Get(context.Background(), name, metav1.GetOptions{})
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

func CreateRoles(client kubernetes.Interface, roles []*rbacv1.Role) ([]*rbacv1.Role, error) {
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

func createRole(client kubernetes.Interface, role *rbacv1.Role) (*rbacv1.Role, error) {
	createdRole, err := client.RbacV1().Roles(role.Namespace).Create(context.Background(), role, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Role '%s/%s' successfully created\n", createdRole.Namespace, createdRole.Name)

	return createdRole, nil
}

func CreateRoleBindings(client kubernetes.Interface, bindings []*rbacv1.RoleBinding) ([]*rbacv1.RoleBinding, error) {
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

func createRoleBinding(client kubernetes.Interface, crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	createdRb, err := client.RbacV1().RoleBindings(crb.Namespace).Create(context.Background(), crb, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("RoleBinding '%s/%s' successfully created\n", createdRb.Namespace, createdRb.Name)

	return createdRb, nil
}
