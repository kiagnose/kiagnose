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
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Checkup struct {
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

func New(image string, timeout time.Duration, envVars []corev1.EnvVar, clusterRoles []*rbacv1.ClusterRole, _ []*rbacv1.Role) *Checkup {
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

	return &Checkup{
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

func (c *Checkup) Setup() error {
	return nil
}

func (c *Checkup) Run() error {
	return nil
}

func (c *Checkup) Teardown() error {
	return nil
}
