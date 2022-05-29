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

package config

import (
	"errors"
	"strings"
	"time"

	"github.com/kiagnose/kiagnose/kiagnose/types"
)

var (
	ErrImageFieldIsMissing   = errors.New("image field is missing")
	ErrImageFieldIsIllegal   = errors.New("image field is illegal")
	ErrTimeoutFieldIsMissing = errors.New("timeout field is missing")
	ErrTimeoutFieldIsIllegal = errors.New("timeout field is illegal")
)

type configMapParser struct {
	configMapRawData map[string]string
	image            string
	timeout          time.Duration
	params           map[string]string
	clusterRoleNames []string
	roleNames        []string
}

func newConfigMapParser(configMapRawData map[string]string) *configMapParser {
	return &configMapParser{
		configMapRawData: configMapRawData,
		params:           map[string]string{},
	}
}

func (cmp *configMapParser) Parse() error {
	if err := cmp.parseImageField(); err != nil {
		return err
	}

	if err := cmp.parseTimeoutField(); err != nil {
		return err
	}

	cmp.parseParamsField()
	cmp.parseClusterRoleNamesField()
	cmp.parseRoleNamesField()

	return nil
}

func (cmp *configMapParser) Image() string {
	return cmp.image
}

func (cmp *configMapParser) Timeout() time.Duration {
	return cmp.timeout
}

func (cmp *configMapParser) Params() map[string]string {
	return cmp.params
}

func (cmp *configMapParser) ClusterRoleNames() []string {
	return cmp.clusterRoleNames
}

func (cmp *configMapParser) RoleNames() []string {
	return cmp.roleNames
}

func (cmp *configMapParser) parseImageField() error {
	var exists bool

	cmp.image, exists = cmp.configMapRawData[types.ImageKey]
	if !exists {
		return ErrImageFieldIsMissing
	}

	if cmp.image == "" {
		return ErrImageFieldIsIllegal
	}

	return nil
}

func (cmp *configMapParser) parseTimeoutField() error {
	rawTimeout, exists := cmp.configMapRawData[types.TimeoutKey]
	if !exists {
		return ErrTimeoutFieldIsMissing
	}

	var err error
	cmp.timeout, err = time.ParseDuration(rawTimeout)
	if err != nil {
		return ErrTimeoutFieldIsIllegal
	}

	return nil
}

func (cmp *configMapParser) parseParamsField() {
	for k, v := range cmp.configMapRawData {
		if strings.HasPrefix(k, types.ParamNameKeyPrefix) {
			paramName := strings.TrimPrefix(k, types.ParamNameKeyPrefix)
			cmp.params[paramName] = v
		}
	}
}

func (cmp *configMapParser) parseClusterRoleNamesField() {
	if rawClusterRoleNames := cmp.configMapRawData[types.ClusterRolesKey]; rawClusterRoleNames != "" {
		cmp.clusterRoleNames = parseListSeparatedByNewlines(rawClusterRoleNames)
	}
}

func (cmp *configMapParser) parseRoleNamesField() {
	if rawRoleNames := cmp.configMapRawData[types.RolesKey]; rawRoleNames != "" {
		cmp.roleNames = parseListSeparatedByNewlines(rawRoleNames)
	}
}

func parseListSeparatedByNewlines(rawString string) []string {
	trimmedString := strings.TrimSpace(rawString)
	return strings.Split(trimmedString, "\n")
}
