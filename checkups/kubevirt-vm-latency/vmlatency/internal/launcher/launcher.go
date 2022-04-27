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

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

type checkup interface {
	Preflight() error
	Setup() error
	Run() error
	Teardown() error
	Results() status.Results
}

type reporter interface {
	Report(status.Status) error
}

type launcher struct {
	checkup  checkup
	reporter reporter
}

func New(checkup checkup, reporter reporter) launcher {
	return launcher{
		checkup:  checkup,
		reporter: reporter,
	}
}

func (l launcher) Run() (runErr error) {
	runStatus := status.Status{}

	defer func() {
		runStatus.Results = l.checkup.Results()
		if err := l.reporter.Report(runStatus); err != nil {
			runStatus.FailureReason = append(runStatus.FailureReason, err.Error())
		}
		runErr = failureReason(runStatus)
	}()

	if err := l.checkup.Preflight(); err != nil {
		runStatus.FailureReason = append(runStatus.FailureReason, err.Error())
		return err
	}

	if err := l.checkup.Setup(); err != nil {
		runStatus.FailureReason = append(runStatus.FailureReason, err.Error())
		return err
	}

	defer func() {
		if err := l.checkup.Teardown(); err != nil {
			runStatus.FailureReason = append(runStatus.FailureReason, err.Error())
		}
	}()

	if err := l.checkup.Run(); err != nil {
		runStatus.FailureReason = append(runStatus.FailureReason, err.Error())
		return err
	}

	return nil
}

func failureReason(sts status.Status) error {
	if len(sts.FailureReason) > 0 {
		return errors.New(strings.Join(sts.FailureReason, ", "))
	}
	return nil
}
