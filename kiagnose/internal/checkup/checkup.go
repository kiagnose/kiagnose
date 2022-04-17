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

package checkup

import (
	"errors"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"

	"github.com/kiagnose/kiagnose/kiagnose/internal/checkup/configmap"
	"github.com/kiagnose/kiagnose/kiagnose/internal/checkup/namespace"
	"github.com/kiagnose/kiagnose/kiagnose/internal/checkup/rbac"
)

type client interface {
	CoreV1() corev1client.CoreV1Interface
	RbacV1() rbacv1client.RbacV1Interface
}

type Checkup struct {
	client              client
	teardownTimeout     time.Duration
	namespace           *corev1.Namespace
	serviceAccount      *corev1.ServiceAccount
	resultConfigMap     *corev1.ConfigMap
	roles               []*rbacv1.Role
	roleBindings        []*rbacv1.RoleBinding
	clusterRoleBindings []*rbacv1.ClusterRoleBinding
	job                 *batchv1.Job
}

const (
	NamespaceName                  = "checkup-workspace"
	ServiceAccountName             = "checkup-sa"
	ResultsConfigMapName           = "checkup-results"
	ResultsConfigMapWriterRoleName = "results-configmap-writer"
	JobName                        = "checkup-job"

	ResultsConfigMapNameEnvVarName      = "RESULT_CONFIGMAP_NAME"
	ResultsConfigMapNameEnvVarNamespace = "RESULT_CONFIGMAP_NAMESPACE"
)

func New(c client, image string, timeout time.Duration, envVars []corev1.EnvVar, clusterRoles []*rbacv1.ClusterRole,
	_ []*rbacv1.Role) *Checkup {
	checkupRoles := []*rbacv1.Role{NewConfigMapWriterRole(ResultsConfigMapWriterRoleName, NamespaceName, ResultsConfigMapName)}

	subject := rbacv1.Subject{Kind: rbacv1.ServiceAccountKind, Name: ServiceAccountName, Namespace: NamespaceName}
	var checkupRoleBindings []*rbacv1.RoleBinding
	for i := range checkupRoles {
		checkupRoleBindings = append(checkupRoleBindings, NewRoleBinding(checkupRoles[i].Name, NamespaceName, subject))
	}
	var checkupClusterRoleBindings []*rbacv1.ClusterRoleBinding
	for i := range clusterRoles {
		checkupClusterRoleBindings = append(checkupClusterRoleBindings, NewClusterRoleBinding(clusterRoles[i].Name, subject))
	}

	checkupEnvVars := []corev1.EnvVar{
		{Name: ResultsConfigMapNameEnvVarName, Value: ResultsConfigMapName},
		{Name: ResultsConfigMapNameEnvVarNamespace, Value: NamespaceName},
	}
	checkupEnvVars = append(checkupEnvVars, envVars...)

	const defaultTeardownTimeout = time.Minute * 5
	return &Checkup{
		client:              c,
		teardownTimeout:     defaultTeardownTimeout,
		namespace:           NewNamespace(NamespaceName),
		serviceAccount:      NewServiceAccount(ServiceAccountName, NamespaceName),
		resultConfigMap:     NewConfigMap(ResultsConfigMapName, NamespaceName),
		roles:               checkupRoles,
		roleBindings:        checkupRoleBindings,
		clusterRoleBindings: checkupClusterRoleBindings,
		job:                 newCheckupJob(JobName, NamespaceName, ServiceAccountName, image, int64(timeout.Seconds()), checkupEnvVars),
	}
}

func NewNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewServiceAccount(name, namespaceName string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
	}
}

func NewConfigMap(name, namespaceName string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
	}
}

func NewConfigMapWriterRole(name, namespaceName, configMapName string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: rbacv1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Rules: []rbacv1.PolicyRule{
			newConfigMapWriterPolicyRule(configMapName),
		},
	}
}

func newConfigMapWriterPolicyRule(cmName string) rbacv1.PolicyRule {
	return rbacv1.PolicyRule{
		Verbs:         []string{"get", "update", "patch"},
		APIGroups:     []string{""},
		Resources:     []string{"configmaps"},
		ResourceNames: []string{cmName},
	}
}

func NewRoleBinding(roleName, namespaceName string, subject rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespaceName},
		Subjects: []rbacv1.Subject{subject},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
			Name:     roleName},
	}
}

func NewClusterRoleBinding(clusterRoleName string, subject rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: rbacv1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName},
		Subjects: []rbacv1.Subject{subject},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: rbacv1.GroupName,
			Name:     clusterRoleName},
	}
}

func newCheckupJob(name, namespaceName, serviceAccountName, image string, activeDeadlineSeconds int64, envs []corev1.EnvVar) *batchv1.Job {
	const containerName = "checkup"

	checkupContainer := corev1.Container{
		Name:            containerName,
		ImagePullPolicy: corev1.PullAlways,
		Image:           image,
		Env:             envs,
	}
	const defaultTerminationGracePeriodSeconds int64 = 5
	t := defaultTerminationGracePeriodSeconds
	checkupPodSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: corev1.PodSpec{
			ServiceAccountName:            serviceAccountName,
			RestartPolicy:                 corev1.RestartPolicyNever,
			TerminationGracePeriodSeconds: &t,
			Containers:                    []corev1.Container{checkupContainer},
		},
	}
	backoffLimit := int32(0)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoffLimit,
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			Template:              checkupPodSpec,
		},
	}
}

// Setup creates each of the checkup objects inside the cluster.
// In case of failure, an attempt to clean up the objects that already been created is made,
// by deleting the Namespace and eventually all the objects inside it
// https://kubernetes.io/docs/concepts/architecture/garbage-collection/#background-deletion
func (c *Checkup) Setup() error {
	const errMessage = "checkup setup failed"
	var err error

	if c.namespace, err = namespace.Create(c.client.CoreV1(), c.namespace); err != nil {
		return fmt.Errorf("%s: %v", errMessage, err)
	}
	defer func() {
		if err != nil {
			_ = namespace.Delete(c.client.CoreV1(), c.namespace.Name)
			_ = namespace.WaitForDeletion(c.client.CoreV1(), c.namespace.Name, c.teardownTimeout)
		}
	}()

	if c.serviceAccount, err = rbac.CreateServiceAccount(c.client.CoreV1(), c.serviceAccount); err != nil {
		return fmt.Errorf("%s: %v", errMessage, err)
	}

	if c.resultConfigMap, err = configmap.Create(c.client.CoreV1(), c.resultConfigMap); err != nil {
		return fmt.Errorf("%s: %v", errMessage, err)
	}

	if c.roles, err = rbac.CreateRoles(c.client.RbacV1(), c.roles); err != nil {
		return fmt.Errorf("%s: %v", errMessage, err)
	}

	if c.roleBindings, err = rbac.CreateRoleBindings(c.client.RbacV1(), c.roleBindings); err != nil {
		return fmt.Errorf("%s: %v", errMessage, err)
	}

	defer func() {
		if err != nil {
			_ = rbac.DeleteClusterRoleBindings(c.client.RbacV1(), c.clusterRoleBindings)
			_ = rbac.WaitForClusterRoleBindingsDeletion(c.client.RbacV1(), c.clusterRoleBindings, c.teardownTimeout)
		}
	}()
	if c.clusterRoleBindings, err = rbac.CreateClusterRoleBindings(c.client.RbacV1(), c.clusterRoleBindings); err != nil {
		return fmt.Errorf("%s: %v", errMessage, err)
	}

	return nil
}

func (c *Checkup) Run() error {
	return nil
}

func (c *Checkup) SetTeardownTimeout(duration time.Duration) {
	c.teardownTimeout = duration
}

func (c *Checkup) Teardown() error {
	var errs []error

	if err := rbac.DeleteClusterRoleBindings(c.client.RbacV1(), c.clusterRoleBindings); err != nil {
		errs = append(errs, err)
	}
	if err := namespace.Delete(c.client.CoreV1(), c.namespace.Name); err != nil {
		errs = append(errs, err)
	}

	if err := namespace.WaitForDeletion(c.client.CoreV1(), c.namespace.Name, c.teardownTimeout); err != nil {
		errs = append(errs, err)
	}
	if err := rbac.WaitForClusterRoleBindingsDeletion(c.client.RbacV1(), c.clusterRoleBindings, c.teardownTimeout); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to teardown checkup: %v", concentrateErrors(errs))
	}

	return nil
}

func concentrateErrors(errs []error) error {
	sb := strings.Builder{}
	for _, err := range errs {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}

	return errors.New(sb.String())
}
