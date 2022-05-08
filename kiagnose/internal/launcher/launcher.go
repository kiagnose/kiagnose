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
	"fmt"
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
	Report(*status.Status) error
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

	if err := l.reporter.Report(&statusData); err != nil {
		return err
	}

	defer func() {
		if runErr != nil {
			statusData.Succeeded = false
			if statusData.FailureReason != "" {
				statusData.FailureReason = fmt.Sprintf("%s, %v", statusData.FailureReason, runErr)
			} else {
				statusData.FailureReason = runErr.Error()
			}
		}

		statusData.CompletionTimestamp = time.Now()

		if reportErr := l.reporter.Report(&statusData); reportErr != nil {
			if runErr != nil {
				runErr = fmt.Errorf("%v, %v", runErr, reportErr)
			} else {
				runErr = reportErr
			}
		}
	}()

	if err := l.checkup.Setup(); err != nil {
		return err
	}

	defer func() {
		if checkupResults, resultsErr := l.checkup.Results(); resultsErr != nil {
			if runErr != nil {
				runErr = fmt.Errorf("%v, %v", runErr, resultsErr)
			} else {
				runErr = resultsErr
			}
		} else {
			statusData.Succeeded = checkupResults.Succeeded
			statusData.FailureReason = checkupResults.FailureReason
			statusData.Results = checkupResults.Results
		}

		if teardownErr := l.checkup.Teardown(); teardownErr != nil {
			if runErr != nil {
				runErr = fmt.Errorf("%v, %v", runErr, teardownErr)
			} else {
				runErr = teardownErr
			}
		}
	}()

	if err := l.checkup.Run(); err != nil {
		return err
	}

	return nil
}
