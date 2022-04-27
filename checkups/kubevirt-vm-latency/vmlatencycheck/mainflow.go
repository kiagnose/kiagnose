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

package vmlatencycheck

import (
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatencycheck/internal/checkup"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatencycheck/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatencycheck/internal/reporter"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatencycheck/internal/runner"
)

func Run(env map[string]string) error {
	cfg, err := config.NewFromEnv(env)
	if err != nil {
		return err
	}

	r := runner.New(checkup.New(nil, cfg), reporter.New(nil, cfg.ResultsConfigMapNamespace, cfg.ResultsConfigMapName))
	return r.Run()
}
