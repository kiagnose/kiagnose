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

package gotestsuite

import "testing"

type Suite struct {
	t                   *testing.T
	fixtures            map[string]func(*testing.T)
	autoRunTestFixtures []func(*testing.T)
}

type FixtureOptions struct {
	AutoRun bool
}

func NewSuite(t *testing.T) *Suite {
	return &Suite{
		t:        t,
		fixtures: map[string]func(*testing.T){},
	}
}

func (s *Suite) AddSetupFixture(id string, f func(*testing.T), options FixtureOptions) {
	if _, ok := s.fixtures[id]; ok {
		panic("Fixture already exists: " + id)
	}

	if options.AutoRun {
		s.autoRunTestFixtures = append(s.autoRunTestFixtures, f)
	} else {
		s.fixtures[id] = f
	}
}

func (s *Suite) Test(name string, f func(*testing.T), fixture ...string) {
	var fixtureFuncs []func(*testing.T)

	for _, fixtureName := range fixture {
		fixtureFunc, ok := s.fixtures[fixtureName]
		if !ok {
			panic("Fixture does not exist: " + fixtureName)
		}
		fixtureFuncs = append(fixtureFuncs, fixtureFunc)
	}

	s.t.Run(name, func(t *testing.T) {
		for _, setup := range s.autoRunTestFixtures {
			setup(t)
		}
		for _, setup := range fixtureFuncs {
			setup(t)
		}
		f(t)
	})
}
