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

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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

func CreateRoles(client kubernetes.Interface, roles []*rbacv1.Role) ([]*rbacv1.Role, error) {
	var createdRoles []*rbacv1.Role
	for _, role := range roles {
		createdRole, err := createRole(client, role)
		if err != nil {
			return nil, err
		}
		createdRoles = append(createdRoles, createdRole)
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

func DeleteRoles(client kubernetes.Interface, roles []*rbacv1.Role) error {
	var errs []error

	for _, role := range roles {
		if err := deleteRole(client, role.Namespace, role.Name); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%v", concentrateErrors(errs))
	}

	return nil
}

func deleteRole(client kubernetes.Interface, namespace, name string) error {
	return client.RbacV1().Roles(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
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

func DeleteRoleBindings(client kubernetes.Interface, roleBindings []*rbacv1.RoleBinding) error {
	var errs []error

	for _, roleBinding := range roleBindings {
		if err := deleteRoleBinding(client, roleBinding.Namespace, roleBinding.Name); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%v", concentrateErrors(errs))
	}

	return nil
}

func deleteRoleBinding(client kubernetes.Interface, namespace, name string) error {
	return client.RbacV1().RoleBindings(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

func concentrateErrors(errs []error) error {
	sb := strings.Builder{}
	for _, err := range errs {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}

	return errors.New(sb.String())
}
