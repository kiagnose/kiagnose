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
	"github.com/kiagnose/kiagnose/kiagnose/internal/status"
)

func TestLauncherRunsSuccessfully(t *testing.T) {
	testLauncher := launcher.New(
		checkupStub{},
		&reporterStub{},
	)

	assert.NoError(t, testLauncher.Run())
}

func TestLauncherShould(t *testing.T) {
	t.Run("fail when report on checkup start is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{},
			&reporterStub{reportErr: errorFailOnInitialReport},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorFailOnInitialReport.Error())
	})

	t.Run("fail when report on checkup completion is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorFailOnFinalReport.Error())
	})

	t.Run("fail when setup is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failSetup: errorSetup},
			&reporterStub{},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorSetup.Error())
	})

	t.Run("fail when setup and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failSetup: errorSetup},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})

	t.Run("fail when run is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun},
			&reporterStub{},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorRun.Error())
	})

	t.Run("fail when teardown is failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failTeardown: errorTeardown},
			&reporterStub{},
		)

		assert.ErrorContains(t, testLauncher.Run(), errorTeardown.Error())
	})

	t.Run("fail when run and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failRun: errorRun},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})

	t.Run("fail when teardown and report on checkup completion are failing", func(t *testing.T) {
		testLauncher := launcher.New(
			checkupStub{failTeardown: errorTeardown},
			&reporterStub{reportErr: errorFailOnFinalReport},
		)

		err := testLauncher.Run()
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorFailOnFinalReport.Error())
	})

	t.Run("fail when run, teardown and report on checkup completion are failing", func(t *testing.T) {
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
