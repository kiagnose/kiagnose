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

package checkup_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kiagnose/kiagnose/kiagnose/internal/checkup"
	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
)

const (
	namespaceResource          = "namespaces"
	namespaceKind              = "Namespace"
	clusterRoleBindingResource = "clusterrolebindings"
	clusterRoleBindingKind     = "ClusterRoleBinding"
	rolesResource              = "roles"
	rolesBindingResource       = "rolebindings"
	roleBindingKind            = "RoleBinding"
	serviceAccountResource     = "serviceaccounts"
	configMapResource          = "configmaps"
	jobResource                = "jobs"

	testTargetNs    = "target-ns"
	testCheckupName = "checkup1"
	testImage       = "framework:v1"
	testTimeout     = time.Minute
)

type checkupSetupTestCase struct {
	description string
	clusterRole []*rbacv1.ClusterRole
	roles       []*rbacv1.Role
	envVars     []corev1.EnvVar
	resource    string
}

func TestCheckupWith(t *testing.T) {
	checkupCreateTestCases := []checkupSetupTestCase{
		{description: "no arguments"},
		{description: "ClusterRoles", clusterRole: newTestClusterRoles()},
		{description: "Roles", roles: newTestRoles()},
		{description: "env vars", envVars: newTestEnvVars()},
	}
	for _, testCase := range checkupCreateTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			c := fake.NewSimpleClientset()
			nameGen := nameGeneratorStub{}
			testCheckup := checkup.New(
				c,
				checkup.KiagnoseNamespace,
				testCheckupName,
				&config.Config{
					Image:        testImage,
					Timeout:      testTimeout,
					EnvVars:      testCase.envVars,
					ClusterRoles: testCase.clusterRole,
					Roles:        testCase.roles,
				},
				nameGen,
			)

			checkupNamespaceName := nameGen.Name(checkup.EphemeralNamespacePrefix)
			resultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
			resultsConfigMapWriterRoleName := checkup.NameResultsConfigMapWriterRole(testCheckupName)
			serviceAccountName := checkup.NameServiceAccount(testCheckupName)

			assert.NoError(t, testCheckup.Setup())
			assertNamespaceCreated(t, c, checkupNamespaceName)
			assertServiceAccountCreated(t, c, checkupNamespaceName, serviceAccountName)
			assertResultsConfigMapCreated(t, c, checkupNamespaceName, resultsConfigMapName)
			assertConfigMapWriterRoleCreated(t, c, checkupNamespaceName, resultsConfigMapName, resultsConfigMapWriterRoleName)
			assertConfigMapWriterRoleBindingCreated(t, c, checkupNamespaceName, resultsConfigMapWriterRoleName, serviceAccountName)
			assertClusterRoleBindingsCreated(t, testsClient{c}, testCase.clusterRole, checkupNamespaceName, serviceAccountName, nameGen)
		})
	}
}

func TestCheckupSetupShouldFailWhen(t *testing.T) {
	checkupCreateFailTestCases := []checkupSetupTestCase{
		{description: "Namespace creation failed", resource: namespaceResource},
		{description: "ServiceAccount creation failed", resource: serviceAccountResource},
		{description: "ConfigMap creation failed", resource: configMapResource},
		{description: "Role creation failed", resource: rolesResource},
		{description: "RolesBinding creation failed", resource: rolesBindingResource},
		{description: "ClusterRoleBinding creation failed",
			resource: clusterRoleBindingResource, clusterRole: newTestClusterRoles()},
	}
	for _, testCase := range checkupCreateFailTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newNormalizedFakeClientset()
			expectedErr := fmt.Sprintf("failed to create resource %q object", testCase.resource)
			testClient.injectCreateErrorForResource(testCase.resource, expectedErr)
			testCheckup := checkup.New(
				testClient,
				checkup.KiagnoseNamespace,
				testCheckupName,
				&config.Config{
					Image:        testImage,
					Timeout:      testTimeout,
					EnvVars:      testCase.envVars,
					ClusterRoles: testCase.clusterRole,
					Roles:        testCase.roles,
				},
				nameGeneratorStub{},
			)

			assert.ErrorContains(t, testCheckup.Setup(), expectedErr)

			assertNoObjectExists(t, testClient)
		})
	}
}

func TestCheckupSetupShould(t *testing.T) {
	t.Run("clean-up remaining ClusterRoleBinding on failure", func(t *testing.T) {
		testClient := newNormalizedFakeClientset()
		expectedErr := fmt.Sprintf("failed to create resource %q object", clusterRoleBindingResource)
		expectedClusterRoles := newTestClusterRoles()
		nameGen := nameGeneratorStub{}
		secondClusterRoleBindingName := nameGen.Name(expectedClusterRoles[1].Name)
		testClient.injectClusterRoleBindingCreateError(secondClusterRoleBindingName, expectedErr)
		testClient.injectResourceVersionUpdateOnJobCreation()
		testCheckup := checkup.New(
			testClient,
			checkup.KiagnoseNamespace,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: testTimeout, ClusterRoles: expectedClusterRoles},
			nameGen,
		)

		assert.ErrorContains(t, testCheckup.Setup(), expectedErr)

		assertNoObjectExists(t, testClient)
	})
}

func TestSetupInTargetNamespaceShouldSucceedWith(t *testing.T) {
	checkupCreateTestCases := []checkupSetupTestCase{
		{description: "no arguments"},
		{description: "ClusterRoles", clusterRole: newTestClusterRoles()},
		{description: "Roles", roles: newTestRoles()},
		{description: "env vars", envVars: newTestEnvVars()},
	}

	for _, testCase := range checkupCreateTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			c := fake.NewSimpleClientset()

			targetNs := newNamespace(testTargetNs)
			_, err := c.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			nameGen := nameGeneratorStub{}

			testCheckup := checkup.New(
				c,
				testTargetNs,
				testCheckupName,
				&config.Config{
					Image:        testImage,
					Timeout:      testTimeout,
					EnvVars:      testCase.envVars,
					ClusterRoles: testCase.clusterRole,
					Roles:        testCase.roles,
				},
				nameGen,
			)

			assert.NoError(t, testCheckup.Setup())

			resultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
			resultsConfigMapWriterRoleName := checkup.NameResultsConfigMapWriterRole(testCheckupName)
			serviceAccountName := checkup.NameServiceAccount(testCheckupName)

			expectedEphemeralNamespaceName := nameGen.Name(checkup.EphemeralNamespacePrefix)
			assertNamespaceDoesntExists(t, c, expectedEphemeralNamespaceName)

			assertServiceAccountCreated(t, c, testTargetNs, serviceAccountName)
			assertResultsConfigMapCreated(t, c, testTargetNs, resultsConfigMapName)
			assertConfigMapWriterRoleCreated(t, c, testTargetNs, resultsConfigMapName, resultsConfigMapWriterRoleName)
			assertConfigMapWriterRoleBindingCreated(t, c, testTargetNs, resultsConfigMapWriterRoleName, serviceAccountName)
			assertClusterRoleBindingsCreated(t, testsClient{c}, testCase.clusterRole, testTargetNs, serviceAccountName, nameGen)
		})
	}
}

func TestSetupInTargetNamespaceShouldFailWhen(t *testing.T) {
	testCases := []struct {
		description string
		resource    string
	}{
		{"Failed to create ServiceAccount", serviceAccountResource},
		{"Failed to create results ConfigMap", configMapResource},
		{"Failed to create ConfigMap writer Role", rolesResource},
		{"Failed to create ConfigMap writer RoleBinding", rolesBindingResource},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newNormalizedFakeClientset()

			targetNs := newNamespace(testTargetNs)
			_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			expectedErr := fmt.Sprintf("failed to create resource %q object", testCase.resource)
			testClient.injectCreateErrorForResource(testCase.resource, expectedErr)

			testCheckup := checkup.New(
				testClient,
				testTargetNs,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout},
				nameGeneratorStub{},
			)

			assert.ErrorContains(t, testCheckup.Setup(), expectedErr)
		})
	}
}

func TestCheckTeardownShouldSucceed(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testCheckup := checkup.New(
		testClient,
		checkup.KiagnoseNamespace,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout},
		nameGeneratorStub{},
	)

	testClient.injectResourceVersionUpdateOnNamespaceCreation()
	testClient.injectWatchWithNamespaceDeleteEvent()

	assert.NoError(t, testCheckup.Setup())
	assert.NoError(t, testCheckup.Teardown())
}

func TestTeardownInTargetNamespaceShouldSucceed(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testClient.injectResourceVersionUpdateOnJobCreation()

	targetNs := newNamespace(testTargetNs)
	_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
	assert.NoError(t, err)

	nameGen := nameGeneratorStub{}
	testCheckup := checkup.New(
		testClient,
		testTargetNs,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout, ClusterRoles: newTestClusterRoles()},
		nameGen,
	)

	assert.NoError(t, testCheckup.Setup())

	checkupJobName := checkup.NameJob(testCheckupName)
	completeTrueJobCondition := &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}
	testClient.injectJobWatchEvent(newJobWithCondition(testTargetNs, checkupJobName, completeTrueJobCondition))
	assert.NoError(t, testCheckup.Run())

	testClient.injectWatchWithJobDeleteEvent(testTargetNs, checkupJobName)
	assert.NoError(t, testCheckup.Teardown())

	assertNamespaceExists(t, testClient.Clientset, testTargetNs)
	assertCheckupJobDoesntExists(t, testClient, testTargetNs, checkupJobName)
	assertNoClusterRoleBindingExists(t, testClient)
	assertNoRoleBindingExists(t, testClient)

	serviceAccountName := checkup.NameServiceAccount(testCheckupName)
	assertServiceAccountDoesntExists(t, testClient.Clientset, testTargetNs, serviceAccountName)

	configMapWriterRoleName := checkup.NameResultsConfigMapWriterRole(testCheckupName)
	assertConfigMapWriterRoleDoesntExists(t, testClient.Clientset, testTargetNs, configMapWriterRoleName)

	resultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
	assertConfigMapDoesntExists(t, testClient.Clientset, testTargetNs, resultsConfigMapName)
}

func TestCheckupTeardownShouldFail(t *testing.T) {
	testTeardownShouldFailWhenNamespaceDeletionFails(t)
	testTeardownShouldFailWhenNamespaceDeletionTimeoutExpires(t)
	testTeardownShouldFailWhenClusterRoleBindingsDeletionFails(t)
	testTeardownShouldFailWhenClusterRoleBindingsDeletionTimeoutExpires(t)
	testTeardownShouldFailWhenNamespaceAndClusterRoleBindingsDeletionFails(t)
}

func testTeardownShouldFailWhenNamespaceDeletionFails(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testCheckup := checkup.New(
		testClient,
		checkup.KiagnoseNamespace,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout},
		nameGeneratorStub{},
	)

	testClient.injectResourceVersionUpdateOnNamespaceCreation()

	assert.NoError(t, testCheckup.Setup())

	const expectedErr = "failed to delete Namespace"
	testClient.injectDeleteErrorForResource(namespaceResource, expectedErr)

	assert.ErrorContains(t, testCheckup.Teardown(), expectedErr)
	assertNoClusterRoleBindingExists(t, testClient)
}

func testTeardownShouldFailWhenNamespaceDeletionTimeoutExpires(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testCheckup := checkup.New(
		testClient,
		checkup.KiagnoseNamespace,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout},
		nameGeneratorStub{},
	)

	testCheckup.SetTeardownTimeout(time.Nanosecond)

	testClient.injectResourceVersionUpdateOnNamespaceCreation()

	assert.NoError(t, testCheckup.Setup())

	testClient.injectIgnoreOperation("delete", namespaceResource)

	assert.ErrorContains(t, testCheckup.Teardown(), wait.ErrWaitTimeout.Error())
	assertNoClusterRoleBindingExists(t, testClient)
}

func testTeardownShouldFailWhenClusterRoleBindingsDeletionFails(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testCheckup := checkup.New(
		testClient,
		checkup.KiagnoseNamespace,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout, ClusterRoles: newTestClusterRoles()},
		nameGeneratorStub{},
	)

	testClient.injectResourceVersionUpdateOnNamespaceCreation()
	testClient.injectWatchWithNamespaceDeleteEvent()

	assert.NoError(t, testCheckup.Setup())

	const expectedErr = "failed to delete ClusterRoleBinding"
	testClient.injectDeleteErrorForResource(clusterRoleBindingResource, expectedErr)

	assert.ErrorContains(t, testCheckup.Teardown(), expectedErr)
	assertNoNamespaceExists(t, testClient)
}

func testTeardownShouldFailWhenClusterRoleBindingsDeletionTimeoutExpires(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testCheckup := checkup.New(
		testClient,
		checkup.KiagnoseNamespace,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout, ClusterRoles: newTestClusterRoles()},
		nameGeneratorStub{},
	)

	testCheckup.SetTeardownTimeout(time.Nanosecond)

	assert.NoError(t, testCheckup.Setup())

	const (
		getClusterRoleBindingsError = "failed to get ClusterRoleBinding"
		expectedErrMatch            = "timed out"
	)
	testClient.injectGetErrorForResource(clusterRoleBindingResource, getClusterRoleBindingsError)

	assert.ErrorContains(t, testCheckup.Teardown(), expectedErrMatch)
	assertNoNamespaceExists(t, testClient)
}

func testTeardownShouldFailWhenNamespaceAndClusterRoleBindingsDeletionFails(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testCheckup := checkup.New(
		testClient,
		checkup.KiagnoseNamespace,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout, ClusterRoles: newTestClusterRoles()},
		nameGeneratorStub{},
	)

	testClient.injectResourceVersionUpdateOnNamespaceCreation()

	assert.NoError(t, testCheckup.Setup())

	const (
		deleteNamespaceError          = "failed to delete Namespace"
		deleteClusterRoleBindingError = "failed to delete ClusterRoleBindings"
	)
	testClient.injectDeleteErrorForResource(namespaceResource, deleteNamespaceError)
	testClient.injectDeleteErrorForResource(clusterRoleBindingResource, deleteClusterRoleBindingError)

	err := testCheckup.Teardown()
	assert.ErrorContains(t, err, deleteNamespaceError)
	assert.ErrorContains(t, err, deleteClusterRoleBindingError)
}

type checkupRunTestCase struct {
	description  string
	envVars      []corev1.EnvVar
	jobCondition *batchv1.JobCondition
}

func TestCheckupRunShouldCreateAJob(t *testing.T) {
	checkupRunTestCases := []checkupRunTestCase{
		{description: "with no checkup parameters"},
		{description: "with additional checkup parameters", envVars: newTestEnvVars()},
	}
	for _, testCase := range checkupRunTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newNormalizedFakeClientset()
			testClient.injectResourceVersionUpdateOnJobCreation()
			testClient.injectResourceVersionUpdateOnNamespaceCreation()
			testClient.injectWatchWithNamespaceDeleteEvent()

			nameGen := nameGeneratorStub{}
			testCheckup := checkup.New(
				testClient,
				checkup.KiagnoseNamespace,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout, EnvVars: testCase.envVars},
				nameGen,
			)

			checkupNamespaceName := nameGen.Name(checkup.EphemeralNamespacePrefix)
			checkupJobName := checkup.NameJob(testCheckupName)
			completeTrueJobCondition := &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}
			testClient.injectJobWatchEvent(newJobWithCondition(checkupNamespaceName, checkupJobName, completeTrueJobCondition))

			assert.NoError(t, testCheckup.Setup())
			assert.NoError(t, testCheckup.Run())

			expectedResultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
			expectedEnvVars := []corev1.EnvVar{
				{Name: checkup.ResultsConfigMapNameEnvVarName, Value: expectedResultsConfigMapName},
				{Name: checkup.ResultsConfigMapNameEnvVarNamespace, Value: checkupNamespaceName},
			}
			expectedEnvVars = append(expectedEnvVars, testCase.envVars...)

			serviceAccountName := checkup.NameServiceAccount(testCheckupName)
			expectedJob := checkup.NewCheckupJob(
				checkupNamespaceName, checkupJobName, serviceAccountName, testImage, int64(testTimeout.Seconds()), expectedEnvVars)
			actualJob, err := testClient.BatchV1().Jobs(checkupNamespaceName).Get(context.Background(), checkupJobName, metav1.GetOptions{})
			assert.NoError(t, err)

			actualJob.ResourceVersion = ""
			assert.Equal(t, actualJob, expectedJob)

			assert.NoError(t, testCheckup.Teardown())
			assertNoObjectExists(t, testClient)
		})
	}
}

func TestCheckupRunShouldSucceed(t *testing.T) {
	checkupRunTestCases := []checkupRunTestCase{
		{description: "when job is completed", jobCondition: &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}},
		{description: "when job failed", jobCondition: &batchv1.JobCondition{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}},
	}
	for _, testCase := range checkupRunTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newNormalizedFakeClientset()
			testClient.injectResourceVersionUpdateOnNamespaceCreation()
			testClient.injectResourceVersionUpdateOnJobCreation()
			testClient.injectWatchWithNamespaceDeleteEvent()

			nameGen := nameGeneratorStub{}
			testCheckup := checkup.New(
				testClient,
				checkup.KiagnoseNamespace,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout},
				nameGen)

			checkupNamespaceName := nameGen.Name(checkup.EphemeralNamespacePrefix)
			checkupJobName := checkup.NameJob(testCheckupName)
			testClient.injectJobWatchEvent(newJobWithCondition(checkupNamespaceName, checkupJobName, testCase.jobCondition))

			assert.NoError(t, testCheckup.Setup())
			assert.NoError(t, testCheckup.Run())
			assert.NoError(t, testCheckup.Teardown())
			assertNoObjectExists(t, testClient)
		})
	}
}

func TestCheckupRunShouldFailWhen(t *testing.T) {
	var testClient *testsClient
	var nameGen nameGeneratorStub

	setup := func() {
		testClient = newNormalizedFakeClientset()
		testClient.injectResourceVersionUpdateOnNamespaceCreation()
		testClient.injectWatchWithNamespaceDeleteEvent()
	}

	t.Run("failed to create Job", func(t *testing.T) {
		const expectedErr = "failed to create Job"

		setup()
		testClient.injectCreateErrorForResource(jobResource, expectedErr)
		testClient.injectResourceVersionUpdateOnJobCreation()

		testCheckup := checkup.New(
			testClient,
			checkup.KiagnoseNamespace,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: testTimeout},
			nameGen,
		)

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), expectedErr)
		assert.NoError(t, testCheckup.Teardown())
		assertNoObjectExists(t, testClient)
	})

	t.Run("fail to watch Job", func(t *testing.T) {
		setup()

		testCheckup := checkup.New(
			testClient,
			checkup.KiagnoseNamespace,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: testTimeout},
			nameGen,
		)

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), "initial RV \"\" is not supported")
		assert.NoError(t, testCheckup.Teardown())
		assertNoObjectExists(t, testClient)
	})

	t.Run("Job wont finish on time", func(t *testing.T) {
		setup()
		testClient.injectResourceVersionUpdateOnJobCreation()

		testCheckup := checkup.New(
			testClient,
			checkup.KiagnoseNamespace,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: time.Nanosecond},
			nameGen,
		)

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), wait.ErrWaitTimeout.Error())
		assert.NoError(t, testCheckup.Teardown())
		assertNoObjectExists(t, testClient)
	})

	t.Run("Job wont finish on time with complete condition status false", func(t *testing.T) {
		setup()
		testClient.injectResourceVersionUpdateOnJobCreation()

		testCheckup := checkup.New(
			testClient,
			checkup.KiagnoseNamespace,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: time.Second},
			nameGen,
		)

		checkupNamespaceName := nameGen.Name(checkup.EphemeralNamespacePrefix)
		checkupJobName := checkup.NameJob(testCheckupName)
		completeFalseJobCondition := &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionFalse}
		testClient.injectJobWatchEvent(newJobWithCondition(checkupNamespaceName, checkupJobName, completeFalseJobCondition))

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), wait.ErrWaitTimeout.Error())
		assert.NoError(t, testCheckup.Teardown())
		assertNoObjectExists(t, testClient)
	})
}

func newTestClusterRoles() []*rbacv1.ClusterRole {
	return []*rbacv1.ClusterRole{
		{TypeMeta: metav1.TypeMeta{Kind: "ClusterRole"}, ObjectMeta: metav1.ObjectMeta{Name: "cluster-role1"}},
		{TypeMeta: metav1.TypeMeta{Kind: "ClusterRole"}, ObjectMeta: metav1.ObjectMeta{Name: "cluster-role2"}}}
}

func newTestRoles() []*rbacv1.Role {
	return []*rbacv1.Role{
		{TypeMeta: metav1.TypeMeta{Kind: "Role"}, ObjectMeta: metav1.ObjectMeta{Name: "role1", Namespace: checkup.EphemeralNamespacePrefix}},
		{TypeMeta: metav1.TypeMeta{Kind: "Role"}, ObjectMeta: metav1.ObjectMeta{Name: "role2", Namespace: checkup.EphemeralNamespacePrefix}}}
}

func newTestEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "env-var-1", Value: "env-var-1-value"},
		{Name: "env-var-1", Value: "env-var-1-value"}}
}

func newJobWithCondition(namespace, name string, condition *batchv1.JobCondition) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: "123",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{*condition},
		},
	}
}

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

type testsClient struct{ *fake.Clientset }

// newNormalizedFakeClientset returns a new fake Kubernetes client with initialized empty lists
// for each group version resource used in the tests.
// Golang differentiates between an initialized slice/map that is empty and uninitialized one that is `nil`.
// By normalizing all to initialized & empty, the tests are extremely simplified, having no need to differentiate
// between such cases.
func newNormalizedFakeClientset() *testsClient {
	f := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "o"}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "o"}},
	)
	_ = f.CoreV1().Namespaces().Delete(context.Background(), "o", metav1.DeleteOptions{})
	_ = f.RbacV1().ClusterRoleBindings().Delete(context.Background(), "o", metav1.DeleteOptions{})

	return &testsClient{f}
}

func (c *testsClient) injectCreateErrorForResource(resourceName, err string) {
	const createVerb = "create"
	c.injectResourceManipulationError(createVerb, resourceName, err)
}

func (c *testsClient) injectGetErrorForResource(resourceName, err string) {
	const getVerb = "get"
	c.injectResourceManipulationError(getVerb, resourceName, err)
}

func (c *testsClient) injectDeleteErrorForResource(resourceName, err string) {
	const deleteVerb = "delete"
	c.injectResourceManipulationError(deleteVerb, resourceName, err)
}

func (c *testsClient) injectResourceManipulationError(verb, resourceName, err string) {
	reactionFn := func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New(err)
	}
	c.PrependReactor(verb, resourceName, reactionFn)
}

// injectIgnoreOperation causes the operation described by the verb and resourceName to be ignored.
// e.g. no events are triggered, datastore is not changed.
func (c *testsClient) injectIgnoreOperation(verb, resourceName string) {
	c.PrependReactor(verb, resourceName, func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
}

func (c *testsClient) injectResourceVersionUpdateOnNamespaceCreation() {
	c.injectResourceVersionUpdate("create", namespaceResource)
}

func (c *testsClient) injectResourceVersionUpdateOnJobCreation() {
	c.injectResourceVersionUpdate("create", jobResource)
}

func (c *testsClient) injectResourceVersionUpdate(verb, resourceName string) {
	c.PrependReactor(verb, resourceName, func(action clienttesting.Action) (bool, runtime.Object, error) {
		createAction, ok := action.(clienttesting.CreateAction)
		if !ok {
			return false, nil, nil
		}

		obj := createAction.GetObject()

		if err := k8smeta.NewAccessor().SetResourceVersion(obj, "123"); err != nil {
			return false, nil, err
		}

		return false, obj, nil
	})
}

// injectClusterRoleBindingCreateError injects an error when the given ClusterRoleBinding name is created.
func (c *testsClient) injectClusterRoleBindingCreateError(clusterRoleBindingName, injectedErr string) {
	const createVerb = "create"
	reactionFn := func(action clienttesting.Action) (bool, runtime.Object, error) {
		if createAction, succeed := action.(clienttesting.CreateActionImpl); succeed {
			clusterRoleBinding, succeed := createAction.Object.(*rbacv1.ClusterRoleBinding)
			if succeed && clusterRoleBinding != nil &&
				clusterRoleBinding.Name == clusterRoleBindingName {
				return true, nil, errors.New(injectedErr)
			}
		}
		// delegate this action to the next reactor
		return false, nil, nil
	}
	c.PrependReactor(createVerb, clusterRoleBindingResource, reactionFn)
}

func (c *testsClient) injectJobWatchEvent(job *batchv1.Job) {
	watchReactionFn := func(action clienttesting.Action) (bool, watch.Interface, error) {
		watcher := watch.NewRaceFreeFake()
		watcher.Modify(job)
		return true, watcher, nil
	}
	c.PrependWatchReactor(jobResource, watchReactionFn)
}

func (c *testsClient) injectWatchWithJobDeleteEvent(namespace, name string) {
	watchReactionFn := func(action clienttesting.Action) (bool, watch.Interface, error) {
		watcher := watch.NewRaceFreeFake()
		watcher.Delete(&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
				Namespace:       namespace,
				ResourceVersion: "123",
			},
		})

		return true, watcher, nil
	}
	c.PrependWatchReactor(jobResource, watchReactionFn)
}

func (c *testsClient) listNamespaces() ([]corev1.Namespace, error) {
	objects, err := c.listObjectsByKind("", namespaceResource, namespaceKind)
	if err != nil {
		return nil, err
	}
	if objects != nil {
		namespacesList := objects.(*corev1.NamespaceList)
		return namespacesList.Items, nil
	}
	return nil, nil
}

func (c *testsClient) injectWatchWithNamespaceDeleteEvent() {
	c.PrependWatchReactor(namespaceResource, func(action clienttesting.Action) (bool, watch.Interface, error) {
		watcher := watch.NewRaceFreeFake()
		watcher.Delete(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:            checkup.EphemeralNamespacePrefix,
					Labels:          k8slabels.Set{corev1.LabelMetadataName: checkup.EphemeralNamespacePrefix},
					ResourceVersion: "123",
				},
			},
		)
		return true, watcher, nil
	})
}

func (c *testsClient) listClusterRoleBindings() ([]rbacv1.ClusterRoleBinding, error) {
	objects, err := c.listObjectsByKind(rbacv1.GroupName, clusterRoleBindingResource, clusterRoleBindingKind)
	if err != nil {
		return nil, err
	}
	if objects != nil {
		clusterRoleBindingsList := objects.(*rbacv1.ClusterRoleBindingList)
		return clusterRoleBindingsList.Items, nil
	}
	return nil, nil
}

func (c *testsClient) listRoleBindings() ([]rbacv1.RoleBinding, error) {
	objects, err := c.listObjectsByKind(rbacv1.GroupName, rolesBindingResource, roleBindingKind)
	if err != nil {
		return nil, err
	}
	if objects != nil {
		roleBindingsList := objects.(*rbacv1.RoleBindingList)
		return roleBindingsList.Items, nil
	}
	return nil, nil
}

func (c *testsClient) listObjectsByKind(group, resourceName, resourceKind string) (runtime.Object, error) {
	gvr := schema.GroupVersionResource{Group: group, Version: "v1", Resource: resourceName}
	gvk := schema.GroupVersionKind{Group: group, Version: "v1", Kind: resourceKind}
	return c.Tracker().List(gvr, gvk, "")
}

func assertNamespaceCreated(t *testing.T, testClient *fake.Clientset, nsName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: namespaceResource}
	actualNs, err := testClient.Tracker().Get(gvr, "", nsName)

	assert.NoError(t, err)
	assert.Equal(t, checkup.NewNamespace(nsName), actualNs)
}

func assertNamespaceExists(t *testing.T, testClient *fake.Clientset, nsName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: namespaceResource}
	_, err := testClient.Tracker().Get(gvr, "", nsName)

	assert.NoError(t, err)
}

func assertNamespaceDoesntExists(t *testing.T, testClient *fake.Clientset, nsName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: namespaceResource}
	_, err := testClient.Tracker().Get(gvr, "", nsName)

	assert.ErrorContains(t, err, "not found")
}

func assertServiceAccountCreated(t *testing.T, testClient *fake.Clientset, nsName, serviceAccountName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: serviceAccountResource}
	actualServiceAccount, err := testClient.Tracker().Get(gvr, nsName, serviceAccountName)

	assert.NoError(t, err)
	assert.Equal(t, checkup.NewServiceAccount(nsName, serviceAccountName), actualServiceAccount)
}

func assertServiceAccountDoesntExists(t *testing.T, testClient *fake.Clientset, nsName, serviceAccountName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: serviceAccountResource}
	_, err := testClient.Tracker().Get(gvr, nsName, serviceAccountName)

	assert.ErrorContains(t, err, "not found")
}

func assertResultsConfigMapCreated(t *testing.T, testClient *fake.Clientset, nsName, expectedConfigMapName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: configMapResource}
	actualConfigMap, err := testClient.Tracker().Get(gvr, nsName, expectedConfigMapName)

	assert.NoError(t, err)
	assert.Equal(t, checkup.NewConfigMap(nsName, expectedConfigMapName), actualConfigMap)
}

func assertConfigMapDoesntExists(t *testing.T, testClient *fake.Clientset, nsName, expectedConfigMapName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: configMapResource}
	_, err := testClient.Tracker().Get(gvr, nsName, expectedConfigMapName)

	assert.ErrorContains(t, err, "not found")
}

func assertConfigMapWriterRoleCreated(t *testing.T, testClient *fake.Clientset, nsName, configMapName, roleName string) {
	gvr := schema.GroupVersionResource{Group: rbacv1.GroupName, Version: "v1", Resource: rolesResource}
	actualRole, err := testClient.Tracker().Get(gvr, nsName, roleName)

	assert.NoError(t, err)

	expectedRole := checkup.NewConfigMapWriterRole(nsName, roleName, configMapName)

	assert.Equal(t, expectedRole, actualRole)
}

func assertConfigMapWriterRoleDoesntExists(t *testing.T, testClient *fake.Clientset, nsName, roleName string) {
	gvr := schema.GroupVersionResource{Group: rbacv1.GroupName, Version: "v1", Resource: rolesResource}
	_, err := testClient.Tracker().Get(gvr, nsName, roleName)

	assert.ErrorContains(t, err, "not found")
}

func assertConfigMapWriterRoleBindingCreated(t *testing.T, testClient *fake.Clientset, nsName, roleName, serviceAccountName string) {
	gvr := schema.GroupVersionResource{Group: rbacv1.GroupName, Version: "v1", Resource: rolesBindingResource}
	actualRoleBinding, err := testClient.Tracker().Get(gvr, nsName, roleName)

	assert.NoError(t, err)

	serviceAccountSubject := checkup.NewServiceAccountSubject(nsName, serviceAccountName)
	expectedRoleBinding := checkup.NewRoleBinding(nsName, roleName, serviceAccountSubject)

	assert.Equal(t, expectedRoleBinding, actualRoleBinding)
}

func assertClusterRoleBindingsCreated(
	t *testing.T,
	testClient testsClient,
	clusterRoles []*rbacv1.ClusterRole,
	nsName,
	serviceAccountName string,
	nameGen nameGeneratorStub) {
	actualClusterRoleBindings, err := testClient.listClusterRoleBindings()
	assert.NoError(t, err)

	serviceAccountSubject := checkup.NewServiceAccountSubject(nsName, serviceAccountName)

	var expectedClusterRoleBindings []rbacv1.ClusterRoleBinding
	for _, clusterRoleBindingPtr := range checkup.NewClusterRoleBindings(clusterRoles, serviceAccountSubject, nameGen) {
		expectedClusterRoleBindings = append(expectedClusterRoleBindings, *clusterRoleBindingPtr)
	}
	assert.Equal(t, actualClusterRoleBindings, expectedClusterRoleBindings)
}

// assertNoObjectExists checks that the checkup's Namespace and ClusterRoleBinding's are deleted.
// When a namespace is deleted each object inside it will be deleted eventually.
// The fake client won't perform forwarder checking when a Namespace is deleted and won't delete each
// associated object, Thus other objects that were created inside the checkup Namespace
// are not being checked explicitly.
func assertNoObjectExists(t *testing.T, testClient *testsClient) {
	assertNoNamespaceExists(t, testClient)
	assertNoClusterRoleBindingExists(t, testClient)
}

func assertNoNamespaceExists(t *testing.T, testClient *testsClient) {
	namespaces, err := testClient.listNamespaces()
	assert.NoError(t, err)
	assert.Empty(t, namespaces)
}

func assertNoClusterRoleBindingExists(t *testing.T, testClient *testsClient) {
	clusterRoleBindings, err := testClient.listClusterRoleBindings()
	assert.NoError(t, err)
	assert.Empty(t, clusterRoleBindings)
}

func assertCheckupJobDoesntExists(t *testing.T, testClient *testsClient, namespace, name string) {
	_, err := testClient.BatchV1().Jobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.ErrorContains(t, err, "not found")
}

func assertNoRoleBindingExists(t *testing.T, testClient *testsClient) {
	roleBindings, err := testClient.listRoleBindings()
	assert.NoError(t, err)
	assert.Empty(t, roleBindings)
}

type nameGeneratorStub struct{}

func (ngs nameGeneratorStub) Name(prefix string) string {
	return fmt.Sprintf("%s-12345", prefix)
}
