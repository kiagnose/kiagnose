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
)

type workload interface {
	Setup() error
	Run() error
	Teardown() error
}

type reporter interface {
	Report() error
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
	defer func() {
		if reportErr := l.reporter.Report(); reportErr != nil {
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
