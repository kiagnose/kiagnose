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

package kiagnose_test

import (
	"errors"
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/kiagnose"
)

func TestRun(t *testing.T) {
	t.Run("should run successfully", func(t *testing.T) {
		assert.NoError(t, kiagnose.Run(launcherStub{}))
	})

	t.Run("should fail when report is failing", func(t *testing.T) {
		assert.ErrorContains(t, kiagnose.Run(launcherStub{failReport: errorReport}), errorReport.Error())
	})

	t.Run("should fail when setup is failing", func(t *testing.T) {
		assert.ErrorContains(t, kiagnose.Run(launcherStub{failSetup: errorSetup}), errorSetup.Error())
	})

	t.Run("should fail when setup and report are failing", func(t *testing.T) {
		err := kiagnose.Run(launcherStub{
			failSetup:  errorSetup,
			failReport: errorReport,
		})
		assert.ErrorContains(t, err, errorSetup.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("should fail when run is failing", func(t *testing.T) {
		assert.ErrorContains(t, kiagnose.Run(launcherStub{failRun: errorRun}), errorRun.Error())
	})

	t.Run("should fail when teardown is failing", func(t *testing.T) {
		assert.ErrorContains(t, kiagnose.Run(launcherStub{failTeardown: errorTeardown}), errorTeardown.Error())
	})

	t.Run("should fail when run and report are failing", func(t *testing.T) {
		err := kiagnose.Run(launcherStub{
			failRun:    errorRun,
			failReport: errorReport})
		assert.ErrorContains(t, err, errorRun.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("should fail when teardown and report are failing", func(t *testing.T) {
		err := kiagnose.Run(launcherStub{
			failTeardown: errorTeardown,
			failReport:   errorReport})
		assert.ErrorContains(t, err, errorTeardown.Error())
		assert.ErrorContains(t, err, errorReport.Error())
	})

	t.Run("should fail when run, teardown and report are failing", func(t *testing.T) {
		err := kiagnose.Run(launcherStub{
			failRun:      errorRun,
			failTeardown: errorTeardown,
			failReport:   errorReport})
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

type launcherStub struct {
	failSetup    error
	failRun      error
	failTeardown error
	failReport   error
}

func (s launcherStub) Setup() error {
	return s.failSetup
}

func (s launcherStub) Run() error {
	return s.failRun
}

func (s launcherStub) Teardown() error {
	return s.failTeardown
}

func (s launcherStub) Report() error {
	return s.failReport
}
