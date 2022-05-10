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
	"github.com/kiagnose/kiagnose/kiagnose/internal/checkup"
	"github.com/kiagnose/kiagnose/kiagnose/internal/client"
	"github.com/kiagnose/kiagnose/kiagnose/internal/config"
	"github.com/kiagnose/kiagnose/kiagnose/internal/launcher"
	"github.com/kiagnose/kiagnose/kiagnose/internal/reporter"
)

func Run(env map[string]string) error {
	k8sClient, crClient, err := client.New()
	if err != nil {
		return err
	}

	checkupKey, err := config.CheckupKeyFromEnv(env)
	if err != nil {
		return err
	}

	checkupConfig, err := config.ReadFromCR(k8sClient, crClient, checkupKey)
	if err != nil {
		return err
	}

	l := launcher.New(checkup.New(k8sClient, checkupConfig), reporter.New(k8sClient, checkupKey))
	return l.Run()
}
