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
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/kiagnose/internal/launcher"
)

func TestLauncherShould(t *testing.T) {
	t.Run("run successfully", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{}, reporterStub{})
		assert.NoError(t, testLauncher.Run())
	})

	t.Run("fail when report is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{}, reporterStub{failReport: errorReport})
		assert.ErrorContains(t, testLauncher.Run(), errorReport.Error())
	})

	t.Run("fail when setup is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failSetup: errorSetup}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorSetup.Error())
	})

	t.Run("fail when setup and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failSetup: errorSetup},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failRun: errorRun}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorRun.Error())
	})

	t.Run("fail when teardown is failing", func(t *testing.T) {
		testLauncher := launcher.New(checkupStub{failTeardown: errorTeardown}, reporterStub{})
		assert.ErrorContains(t, testLauncher.Run(), errorTeardown.Error())
	})

	t.Run("fail when run and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when teardown and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failTeardown: errorTeardown},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("fail when run, teardown and report are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun, failTeardown: errorTeardown},
			reporterStub{failReport: errorReport},
		)
		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})
}

var (
	errorSetup    = errors.New("setup error")
	errorRun      = errors.New("run error")
	errorTeardown = errors.New("teardown error")
	errorReport   = errors.New("report error")
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
	failReport error
}

func (r reporterStub) Report() error {
	return r.failReport
}
