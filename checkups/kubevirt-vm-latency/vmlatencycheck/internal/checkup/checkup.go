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

type checkup struct {
	client interface{}
	config interface{}
}

func New(client, cfg interface{}) *checkup {
	return &checkup{client: client, config: cfg}
}

func (c *checkup) Preflights() error {
	return nil
}

func (c *checkup) Setup() error {
	return nil
}

func (c *checkup) Run() error {
	return nil
}

func (c *checkup) Teardown() error {
	return nil
}

func (c *checkup) Results() map[string]string {
	return nil
}
