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

package launcher

import (
	"errors"
	"strings"
	"time"

	"github.com/kiagnose/kiagnose/kiagnose/internal/results"
	"github.com/kiagnose/kiagnose/kiagnose/internal/status"
)

type workload interface {
	Setup() error
	Run() error
	Results() (results.Results, error)
	Teardown() error
}

type reporter interface {
	Report(status.Status) error
}

type Launcher struct {
	checkup  workload
	reporter reporter
}

func New(checkup workload, reporter reporter) Launcher {
	return Launcher{
		checkup:  checkup,
		reporter: reporter,
	}
}

func (l Launcher) Run() (runErr error) {
	statusData := status.Status{StartTimestamp: time.Now()}

	if err := l.reporter.Report(statusData); err != nil {
		return err
	}

	var errorPool []error
	defer func() {
		if runErr != nil {
			errorPool = append(errorPool, runErr)
		}

		statusData.CompletionTimestamp = time.Now()
		if len(errorPool) > 0 {
			statusData.Succeeded = false
			statusData.FailureReason = append(statusData.FailureReason, joinErrors(errorPool)...)
		}

		if reportErr := l.reporter.Report(statusData); reportErr != nil {
			errorPool = append(errorPool, reportErr)
		}

		if len(errorPool) > 0 {
			runErr = errors.New(strings.Join(joinErrors(errorPool), ", "))
		}
	}()

	if err := l.checkup.Setup(); err != nil {
		return err
	}

	defer func() {
		if runErr != nil {
			errorPool = append(errorPool, runErr)
		}
		runErr = l.checkup.Teardown()
	}()

	if err := l.checkup.Run(); err != nil {
		return err
	}

	resultData, err := l.checkup.Results()
	if err != nil {
		return err
	}

	statusData.Succeeded = resultData.Succeeded
	statusData.Results = resultData.Results
	if resultData.FailureReason != "" {
		statusData.FailureReason = append(statusData.FailureReason, resultData.FailureReason)
	}

	return nil
}

func joinErrors(errs []error) []string {
	var errorTextPool []string
	for _, e := range errs {
		errorTextPool = append(errorTextPool, e.Error())
	}
	return errorTextPool
}
