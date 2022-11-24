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

package vmlatency

import (
	kconfig "github.com/kiagnose/kiagnose/kiagnose/config"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/checkup"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/client"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/latency"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/launcher"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/reporter"
)

func Run(rawEnv map[string]string, namespace string) error {
	c, err := client.New()
	if err != nil {
		return err
	}

	environment := kconfig.NewEnvironment(rawEnv)
	err = environment.Validate()
	if err != nil {
		return err
	}

	baseConfig, err := kconfig.ReadFromConfigMap(c, environment.ConfigMapNamespace, environment.ConfigMapName)
	if err != nil {
		return err
	}

	cfg, err := config.New(baseConfig)
	if err != nil {
		return err
	}

	l := launcher.New(
		checkup.New(c, baseConfig.UID, namespace, cfg, latency.New(c)),
		reporter.New(c, environment.ConfigMapNamespace, environment.ConfigMapName),
	)
	return l.Run()
}
