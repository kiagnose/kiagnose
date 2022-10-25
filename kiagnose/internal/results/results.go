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

package results

import (
	"errors"
	"strconv"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/kiagnose/kiagnose/kiagnose/configmap"
)

const (
	SucceededKey     = "status.succeeded"
	FailureReasonKey = "status.failureReason"
	ResultsPrefix    = "status.result."
)

var (
	ErrConfigMapDataIsNil        = errors.New("results: ConfigMap Data is nil")
	ErrSucceededFieldMissing     = errors.New("results: succeeded field is missing")
	ErrSucceededFieldIllegal     = errors.New("results: succeeded field is illegal")
	ErrFailureReasonFieldMissing = errors.New("results: failureReason field is missing")
)

type Results struct {
	Succeeded     bool
	FailureReason string
	Results       map[string]string
}

func ReadFromConfigMap(client kubernetes.Interface, configMapNamespace, configMapName string) (Results, error) {
	resultsData := Results{}

	configMap, err := configmap.Get(client, configMapNamespace, configMapName)
	if err != nil {
		return resultsData, err
	}

	if configMap.Data == nil {
		return resultsData, ErrConfigMapDataIsNil
	}

	if resultsData.Succeeded, err = parseSucceededField(configMap.Data); err != nil {
		return resultsData, err
	}

	if resultsData.FailureReason, err = parseFailureReasonField(configMap.Data); err != nil {
		return resultsData, err
	}

	resultsData.Results = parseResultsField(configMap.Data)

	return resultsData, nil
}

func parseSucceededField(data map[string]string) (bool, error) {
	rawSucceededField, exists := data[SucceededKey]
	if !exists {
		return false, ErrSucceededFieldMissing
	}

	succeeded, err := strconv.ParseBool(rawSucceededField)
	if err != nil {
		return false, ErrSucceededFieldIllegal
	}

	return succeeded, nil
}

func parseFailureReasonField(data map[string]string) (string, error) {
	failureReason, exists := data[FailureReasonKey]
	if !exists {
		return "", ErrFailureReasonFieldMissing
	}

	return failureReason, nil
}

func parseResultsField(data map[string]string) map[string]string {
	results := map[string]string{}

	for k, v := range data {
		if strings.HasPrefix(k, ResultsPrefix) {
			trimmedKey := strings.TrimPrefix(k, ResultsPrefix)
			results[trimmedKey] = v
		}
	}

	return results
}
