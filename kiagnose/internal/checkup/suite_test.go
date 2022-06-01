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

package checkup_test

import (
	"testing"

	suite "github.com/kiagnose/kiagnose/kiagnose/internal/gotestsuite"
)

type checkupSuite struct {
	suite.Suite
	tClient *testsClient
}

func NewCheckupSuite(t *testing.T) *checkupSuite {
	s := &checkupSuite{
		Suite: *suite.NewSuite(t),
	}

	s.AddSetupFixture("Reset Test Client", func(t *testing.T) {
		s.tClient = nil
	}, suite.FixtureOptions{AutoRun: true})

	return s
}

func (cs *checkupSuite) Test(name string, f func(*testing.T, *testsClient), fixture ...string) {
	cs.Suite.Test(name, func(t *testing.T) {
		if cs.tClient == nil {
			panic("Test Client is not set")
		}
		f(t, cs.tClient)
	}, fixture...)
}

func (cs *checkupSuite) SetClient(c *testsClient) {
	cs.tClient = c
}
