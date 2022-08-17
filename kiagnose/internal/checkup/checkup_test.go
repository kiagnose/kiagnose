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
	namespaceResource    = "namespaces"
	rolesResource        = "roles"
	rolesBindingResource = "rolebindings"
	roleBindingKind      = "RoleBinding"
	configMapResource    = "configmaps"
	jobResource          = "jobs"

	testTargetNs           = "target-ns"
	testCheckupName        = "checkup1"
	testImage              = "framework:v1"
	testTimeout            = time.Minute
	testServiceAccountName = "test-sa"
)

type checkupSetupTestCase struct {
	description string
	envVars     []corev1.EnvVar
}

func TestSetupInTargetNamespaceShouldSucceedWith(t *testing.T) {
	checkupCreateTestCases := []checkupSetupTestCase{
		{description: "no arguments"},
		{description: "env vars", envVars: newTestEnvVars()},
	}

	for _, testCase := range checkupCreateTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			c := fake.NewSimpleClientset()

			targetNs := newTestNamespace()
			_, err := c.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			serviceAccount := newTestServiceAccount()
			_, err = c.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			assert.NoError(t, err)

			testCheckup := checkup.New(
				c,
				testTargetNs,
				testCheckupName,
				&config.Config{
					Image:              testImage,
					Timeout:            testTimeout,
					ServiceAccountName: testServiceAccountName,
					EnvVars:            testCase.envVars,
				},
			)

			assert.NoError(t, testCheckup.Setup())

			resultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
			resultsConfigMapWriterRoleName := checkup.NameResultsConfigMapWriterRole(testCheckupName)

			assertResultsConfigMapCreated(t, c, testTargetNs, resultsConfigMapName)
			assertConfigMapWriterRoleCreated(t, c, testTargetNs, resultsConfigMapName, resultsConfigMapWriterRoleName)
			assertConfigMapWriterRoleBindingCreated(t, c, testTargetNs, resultsConfigMapWriterRoleName, testServiceAccountName)
		})
	}
}

func TestSetupInTargetNamespaceShouldFailWhen(t *testing.T) {
	testCases := []struct {
		description string
		resource    string
	}{
		{"Failed to create results ConfigMap", configMapResource},
		{"Failed to create ConfigMap writer Role", rolesResource},
		{"Failed to create ConfigMap writer RoleBinding", rolesBindingResource},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newNormalizedFakeClientset()

			targetNs := newTestNamespace()
			_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			serviceAccount := newTestServiceAccount()
			_, err = testClient.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			assert.NoError(t, err)

			expectedErr := fmt.Sprintf("failed to create resource %q object", testCase.resource)
			testClient.injectCreateErrorForResource(testCase.resource, expectedErr)

			testCheckup := checkup.New(
				testClient,
				testTargetNs,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName},
			)

			assert.ErrorContains(t, testCheckup.Setup(), expectedErr)
		})
	}
}

func TestTeardownInTargetNamespaceShouldSucceed(t *testing.T) {
	testClient := newNormalizedFakeClientset()
	testClient.injectResourceVersionUpdateOnJobCreation()

	targetNs := newTestNamespace()
	_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
	assert.NoError(t, err)

	serviceAccount := newTestServiceAccount()
	_, err = testClient.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
	assert.NoError(t, err)

	testCheckup := checkup.New(
		testClient,
		testTargetNs,
		testCheckupName,
		&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName},
	)

	assert.NoError(t, testCheckup.Setup())

	checkupJobName := checkup.NameJob(testCheckupName)
	completeTrueJobCondition := &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}
	testClient.injectJobWatchEvent(newJobWithCondition(checkupJobName, completeTrueJobCondition))
	assert.NoError(t, testCheckup.Run())

	testClient.injectWatchWithJobDeleteEvent(testTargetNs, checkupJobName)
	assert.NoError(t, testCheckup.Teardown())

	assertNamespaceExists(t, testClient.Clientset, testTargetNs)
	assertCheckupJobDoesntExists(t, testClient, testTargetNs, checkupJobName)
	assertNoRoleBindingExists(t, testClient)

	configMapWriterRoleName := checkup.NameResultsConfigMapWriterRole(testCheckupName)
	assertConfigMapWriterRoleDoesntExists(t, testClient.Clientset, testTargetNs, configMapWriterRoleName)

	resultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
	assertConfigMapDoesntExists(t, testClient.Clientset, testTargetNs, resultsConfigMapName)
}

func TestTeardownInTargetNamespaceShouldFailWhen(t *testing.T) {
	testCases := []struct {
		description string
		resource    string
	}{
		{"Failed to delete Job", jobResource},
		{"Failed to create ConfigMap writer RoleBinding", rolesBindingResource},
		{"Failed to create results ConfigMap", configMapResource},
		{"Failed to create ConfigMap writer Role", rolesResource},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testClient := newNormalizedFakeClientset()

			targetNs := newTestNamespace()
			_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			serviceAccount := newTestServiceAccount()
			_, err = testClient.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			assert.NoError(t, err)

			testCheckup := checkup.New(
				testClient,
				testTargetNs,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName},
			)

			assert.NoError(t, testCheckup.Setup())

			expectedErr := fmt.Sprintf("failed to create resource %q object", testCase.resource)
			testClient.injectDeleteErrorForResource(testCase.resource, expectedErr)

			assert.ErrorContains(t, testCheckup.Teardown(), expectedErr)
		})
	}
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

			targetNs := newTestNamespace()
			_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			serviceAccount := newTestServiceAccount()
			_, err = testClient.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			assert.NoError(t, err)

			testClient.injectResourceVersionUpdateOnJobCreation()

			testCheckup := checkup.New(
				testClient,
				testTargetNs,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName, EnvVars: testCase.envVars},
			)

			checkupJobName := checkup.NameJob(testCheckupName)
			completeTrueJobCondition := &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}
			testClient.injectJobWatchEvent(newJobWithCondition(checkupJobName, completeTrueJobCondition))

			assert.NoError(t, testCheckup.Setup())
			assert.NoError(t, testCheckup.Run())

			expectedResultsConfigMapName := checkup.NameResultsConfigMap(testCheckupName)
			expectedEnvVars := []corev1.EnvVar{
				{Name: checkup.ResultsConfigMapNameEnvVarName, Value: expectedResultsConfigMapName},
				{Name: checkup.ResultsConfigMapNameEnvVarNamespace, Value: testTargetNs},
			}
			expectedEnvVars = append(expectedEnvVars, testCase.envVars...)

			expectedJob := checkup.NewCheckupJob(
				testTargetNs, checkupJobName, testServiceAccountName, testImage, int64(testTimeout.Seconds()), expectedEnvVars)
			actualJob, err := testClient.BatchV1().Jobs(testTargetNs).Get(context.Background(), checkupJobName, metav1.GetOptions{})
			assert.NoError(t, err)

			actualJob.ResourceVersion = ""
			assert.Equal(t, actualJob, expectedJob)

			testClient.injectWatchWithJobDeleteEvent(testTargetNs, checkupJobName)
			assert.NoError(t, testCheckup.Teardown())
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

			targetNs := newTestNamespace()
			_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
			assert.NoError(t, err)

			serviceAccount := newTestServiceAccount()
			_, err = testClient.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			assert.NoError(t, err)

			testClient.injectResourceVersionUpdateOnJobCreation()

			testCheckup := checkup.New(
				testClient,
				testTargetNs,
				testCheckupName,
				&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName},
			)

			checkupJobName := checkup.NameJob(testCheckupName)
			testClient.injectJobWatchEvent(newJobWithCondition(checkupJobName, testCase.jobCondition))

			assert.NoError(t, testCheckup.Setup())
			assert.NoError(t, testCheckup.Run())

			testClient.injectWatchWithJobDeleteEvent(testTargetNs, checkupJobName)
			assert.NoError(t, testCheckup.Teardown())
		})
	}
}

func TestCheckupRunShouldFailWhen(t *testing.T) {
	var testClient *testsClient

	setup := func() {
		testClient = newNormalizedFakeClientset()

		targetNs := newTestNamespace()
		_, err := testClient.CoreV1().Namespaces().Create(context.Background(), targetNs, metav1.CreateOptions{})
		assert.NoError(t, err)

		serviceAccount := newTestServiceAccount()
		_, err = testClient.CoreV1().ServiceAccounts(testTargetNs).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	t.Run("failed to create Job", func(t *testing.T) {
		const expectedErr = "failed to create Job"

		setup()
		testClient.injectCreateErrorForResource(jobResource, expectedErr)
		testClient.injectResourceVersionUpdateOnJobCreation()

		testCheckup := checkup.New(
			testClient,
			testTargetNs,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName},
		)

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), expectedErr)
		assert.NoError(t, testCheckup.Teardown())
	})

	t.Run("fail to watch Job", func(t *testing.T) {
		setup()

		testCheckup := checkup.New(
			testClient,
			testTargetNs,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: testTimeout, ServiceAccountName: testServiceAccountName},
		)

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), "initial RV \"\" is not supported")
		assert.NoError(t, testCheckup.Teardown())
	})

	t.Run("Job wont finish on time", func(t *testing.T) {
		setup()
		testClient.injectResourceVersionUpdateOnJobCreation()

		testCheckup := checkup.New(
			testClient,
			testTargetNs,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: time.Nanosecond, ServiceAccountName: testServiceAccountName},
		)

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), wait.ErrWaitTimeout.Error())
		assert.NoError(t, testCheckup.Teardown())
	})

	t.Run("Job wont finish on time with complete condition status false", func(t *testing.T) {
		setup()
		testClient.injectResourceVersionUpdateOnJobCreation()

		testCheckup := checkup.New(
			testClient,
			testTargetNs,
			testCheckupName,
			&config.Config{Image: testImage, Timeout: time.Second, ServiceAccountName: testServiceAccountName},
		)

		checkupJobName := checkup.NameJob(testCheckupName)
		completeFalseJobCondition := &batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionFalse}
		testClient.injectJobWatchEvent(newJobWithCondition(checkupJobName, completeFalseJobCondition))

		assert.NoError(t, testCheckup.Setup())
		assert.ErrorContains(t, testCheckup.Run(), wait.ErrWaitTimeout.Error())
		assert.NoError(t, testCheckup.Teardown())
	})
}

func newTestEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "env-var-1", Value: "env-var-1-value"},
		{Name: "env-var-1", Value: "env-var-1-value"}}
}

func newJobWithCondition(name string, condition *batchv1.JobCondition) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       testTargetNs,
			ResourceVersion: "123",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{*condition},
		},
	}
}

func newTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testTargetNs,
		},
	}
}

func newTestServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServiceAccountName,
			Namespace: testTargetNs,
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

func assertNamespaceExists(t *testing.T, testClient *fake.Clientset, nsName string) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: namespaceResource}
	_, err := testClient.Tracker().Get(gvr, "", nsName)

	assert.NoError(t, err)
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

func assertCheckupJobDoesntExists(t *testing.T, testClient *testsClient, namespace, name string) {
	_, err := testClient.BatchV1().Jobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.ErrorContains(t, err, "not found")
}

func assertNoRoleBindingExists(t *testing.T, testClient *testsClient) {
	roleBindings, err := testClient.listRoleBindings()
	assert.NoError(t, err)
	assert.Empty(t, roleBindings)
}
