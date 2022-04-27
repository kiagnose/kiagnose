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

package checkup

import (
	"fmt"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/client"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/status"
)

type checkup struct {
	client  *client.Client
	params  config.CheckupParameters
	results status.Results
}

func New(c *client.Client, params config.CheckupParameters) *checkup {
	return &checkup{client: c, params: params}
}

func (c *checkup) Preflight() error {
	return nil
}

func (c *checkup) Setup() error {
	return nil
}

func (c *checkup) Run() error {
	const errMessagePrefix = "run"
	return fmt.Errorf("%s: not implemented", errMessagePrefix)
}

func (c *checkup) Teardown() error {
	return nil
}

func (c *checkup) Results() status.Results {
	return c.results
}
