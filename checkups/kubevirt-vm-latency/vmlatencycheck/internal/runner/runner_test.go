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

package runner_test

import (
	"errors"
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatencycheck/internal/runner"
)

func TestRunnerShould(t *testing.T) {
	t.Run("run successfully", func(t *testing.T) {
		testRunner := runner.New(checkupStub{}, reporterStub{})
		assert.NoError(t, testRunner.Run())
	})

	t.Run("fail when report is failing", func(t *testing.T) {
		testRunner := runner.New(checkupStub{}, reporterStub{failReport: errorReport})
		assert.ErrorContains(t, testRunner.Run(), errorReport.Error())
	})

	t.Run("fail when preflight is failing", func(t *testing.T) {
		testRunner := runner.New(checkupStub{failPreflights: errorPreflights}, reporterStub{})
		assert.ErrorContains(t, testRunner.Run(), errorPreflights.Error())
	})

	t.Run("fail when preflight and report are failing", func(t *testing.T) {
		testRunner := runner.New(
			checkupStub{failPreflights: errorPreflights},
			reporterStub{failReport: errorReport},
		)
		err := testRunner.Run()
		assert.ErrorContains(t, err, errorPreflights.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when setup is failing", func(t *testing.T) {
		testRunner := runner.New(checkupStub{failSetup: errorSetup}, reporterStub{})
		assert.ErrorContains(t, testRunner.Run(), errorSetup.Error())
	})

	t.Run("fail when setup and report are failing", func(t *testing.T) {
		testRunner := runner.New(
			checkupStub{failSetup: errorSetup},
			reporterStub{failReport: errorReport},
		)
		err := testRunner.Run()
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run is failing", func(t *testing.T) {
		testRunner := runner.New(checkupStub{failRun: errorRun}, reporterStub{})
		assert.ErrorContains(t, testRunner.Run(), errorRun.Error())
	})

	t.Run("fail when teardown is failing", func(t *testing.T) {
		testRunner := runner.New(checkupStub{failTeardown: errorTeardown}, reporterStub{})
		assert.ErrorContains(t, testRunner.Run(), errorTeardown.Error())
	})

	t.Run("fail when run and report are failing", func(t *testing.T) {
		testRunner := runner.New(
			checkupStub{failRun: errorRun},
			reporterStub{failReport: errorReport},
		)
		err := testRunner.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when teardown and report are failing", func(t *testing.T) {
		testRunner := runner.New(
			checkupStub{failTeardown: errorTeardown},
			reporterStub{failReport: errorReport},
		)
		err := testRunner.Run()
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run, teardown and report are failing", func(t *testing.T) {
		testRunner := runner.New(
			checkupStub{failRun: errorRun, failTeardown: errorTeardown},
			reporterStub{failReport: errorReport},
		)
		err := testRunner.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})
}

var (
	errorPreflights = errors.New("preflight check error")
	errorSetup      = errors.New("setup error")
	errorRun        = errors.New("run error")
	errorTeardown   = errors.New("teardown error")
	errorReport     = errors.New("report error")
)

type checkupStub struct {
	failPreflights error
	failSetup      error
	failRun        error
	failTeardown   error
	results        map[string]string
}

func (s checkupStub) Preflights() error {
	return s.failPreflights
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
	failReport error
}

func (s checkupStub) Results() map[string]string {
	return s.results
}

func (r reporterStub) Report(_ map[string]string) error {
	return r.failReport
}
