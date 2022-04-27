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

package reporter

import (
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/client"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

type reporter struct {
	client             *client.Client
	configMapName      string
	configMapNamespace string
}

func New(c *client.Client, configMapNamespace, configMapName string) *reporter {
	return &reporter{
		client:             c,
		configMapNamespace: configMapNamespace,
		configMapName:      configMapName,
	}
}

func (r *reporter) Report(_ status.Status) error {
	return nil
}
