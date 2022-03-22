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

package kiagnose

import (
	"fmt"
)

type launcher interface {
	Setup() error
	Run() error
	Teardown() error
	Report() error
}

func Run(launcher launcher) (runErr error) {
	defer func() {
		if reportErr := launcher.Report(); reportErr != nil {
			if runErr != nil {
				runErr = fmt.Errorf("%v, %v", runErr, reportErr)
			} else {
				runErr = reportErr
			}
		}
	}()

	if err := launcher.Setup(); err != nil {
		return err
	}

	defer func() {
		if teardownErr := launcher.Teardown(); teardownErr != nil {
			if runErr != nil {
				runErr = fmt.Errorf("%v, %v", runErr, teardownErr)
			} else {
				runErr = teardownErr
			}
		}
	}()

	if err := launcher.Run(); err != nil {
		return err
	}

	return nil
}
