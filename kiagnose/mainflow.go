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
	"log"
	"os"

	"github.com/kiagnose/kiagnose/kiagnose/internal/checkup"
	"github.com/kiagnose/kiagnose/kiagnose/internal/launcher"
	"github.com/kiagnose/kiagnose/kiagnose/internal/reporter"
)

const (
	configMapNamespaceEnvVarName = "CONFIGMAP_NAMESPACE"
	configMapNameEnvVarName      = "CONFIGMAP_NAME"
)

func Run() error {
	configMapNamespace, configMapName, err := readConfigMapFullNameFromEnv()
	if err != nil {
		return err
	}
	log.Printf("ConfigMap fullname: \"%s/%s\"", configMapNamespace, configMapName)

	l := launcher.New(checkup.New(), reporter.New())
	return l.Run()
}

func readConfigMapFullNameFromEnv() (namespace, name string, err error) {
	var exists bool

	namespace, exists = os.LookupEnv(configMapNamespaceEnvVarName)
	if !exists {
		return "", "", fmt.Errorf("missing %q environment variable", configMapNamespaceEnvVarName)
	}

	name, exists = os.LookupEnv(configMapNameEnvVarName)
	if !exists {
		return "", "", fmt.Errorf("missing %q environment variable", configMapNameEnvVarName)
	}

	return
}
