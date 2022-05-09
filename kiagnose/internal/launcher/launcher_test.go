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

package launcher_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
	"github.com/kiagnose/kiagnose/kiagnose/internal/configmap"
	"github.com/kiagnose/kiagnose/kiagnose/internal/launcher"
	"github.com/kiagnose/kiagnose/kiagnose/internal/reporter"
	"github.com/kiagnose/kiagnose/kiagnose/internal/status"
)

const (
	configMapNamespace = "kiagnose"
	configMapName      = "checkup1"
)

func TestLauncherRunsSuccessfully(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(newConfigMap(checkupSpecData()))

	testLauncher := launcher.New(
		checkupStub{},
		reporter.New(fakeClient, configMapNamespace, configMapName),
	)

	assert.NoError(t, testLauncher.Run())

	actualCheckupData := getCheckupData(t, fakeClient, configMapNamespace, configMapName)
	zeroTimestamps(actualCheckupData)

	expectedData := checkupSpecData()
	expectedData[reporter.SucceededKey] = strconv.FormatBool(true)
	expectedData[reporter.FailureReasonKey] = ""
	zeroTimestamps(expectedData)

	assert.Equal(t, expectedData, actualCheckupData)
}

func TestLauncherRunShouldFailWithReportWhen(t *testing.T) {
	type failureTestCase struct {
		description   string
		inputCheckup  checkupStub
		expectedError error
	}

	testCases := []failureTestCase{
		{
			description:   "setup is failing",
			inputCheckup:  checkupStub{failSetup: errorSetup},
			expectedError: errorSetup,
		},
		{
			description:   "run is failing",
			inputCheckup:  checkupStub{failRun: errorRun},
			expectedError: errorRun,
		},
		{
			description:   "teardown is failing",
			inputCheckup:  checkupStub{failTeardown: errorTeardown},
			expectedError: errorTeardown,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(newConfigMap(checkupSpecData()))

			testLauncher := launcher.New(
				testCase.inputCheckup,
				reporter.New(fakeClient, configMapNamespace, configMapName),
			)

			assert.ErrorContains(t, testLauncher.Run(), testCase.expectedError.Error())

			actualCheckupData := getCheckupData(t, fakeClient, configMapNamespace, configMapName)
			zeroTimestamps(actualCheckupData)

			expectedData := checkupSpecData()
			expectedData[reporter.SucceededKey] = strconv.FormatBool(false)
			expectedData[reporter.FailureReasonKey] = testCase.expectedError.Error()
			zeroTimestamps(expectedData)

			assert.Equal(t, expectedData, actualCheckupData)
		})
	}
}

func TestLauncherRunShouldFailWithoutReportWhen(t *testing.T) {
	t.Run("report on checkup start is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{},
			&reporterStub{reportErr: errorFailOnInitialReport},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorFailOnInitialReport.Error())
	})

	t.Run("report on checkup completion is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorFailOnFinalReport.Error())
	})

	t.Run("setup and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failSetup: errorSetup},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})

	t.Run("run and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})

	t.Run("teardown and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failTeardown: errorTeardown},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})

	t.Run("run, teardown and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun, failTeardown: errorTeardown},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})
}

var (
	errorSetup               = errors.New("setup error")
	errorRun                 = errors.New("run error")
	errorTeardown            = errors.New("teardown error")
	errorFailOnInitialReport = errors.New("initial report error")
	errorFailOnFinalReport   = errors.New("final report error")
)

type checkupStub struct {
	failSetup    error
	failRun      error
	failTeardown error
}

func (s checkupStub) Setup() error {
	return s.failSetup
}

func (s checkupStub) Run() error {
	return s.failRun
}

func (s checkupStub) Teardown() error {
	return s.failTeardown
}

type reporterStub struct {
	reportErr   error
	reportCount int
}

func (r *reporterStub) Report(_ status.Status) error {
	r.reportCount++
	if r.reportCount > 2 {
		panic("Report was called more than twice")
	}

	if r.reportCount == 1 && r.reportErr == errorFailOnInitialReport ||
		r.reportCount == 2 && r.reportErr == errorFailOnFinalReport {
		return r.reportErr
	}

	return nil
}

func checkupSpecData() map[string]string {
	const (
		testImageValue   = "mycheckup:v0.1.0"
		testTimeoutValue = "1m"
	)

	return map[string]string{config.ImageKey: testImageValue, config.TimeoutKey: testTimeoutValue}
}

func newConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		Data: data,
	}
}

func getCheckupData(t *testing.T, client kubernetes.Interface, configMapNamespace, configMapName string) map[string]string {
	configMap, err := configmap.Get(client.CoreV1(), configMapNamespace, configMapName)
	assert.NoError(t, err)

	return configMap.Data
}

func zeroTimestamps(data map[string]string) {
	data[reporter.StartTimestampKey] = time.Time{}.Format(time.RFC3339)
	data[reporter.CompletionTimestampKey] = time.Time{}.Format(time.RFC3339)
}
