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
	"errors"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/kiagnose/kiagnose/kiagnose/internal/configmap"
)

const (
	SucceededKey           = "status.succeeded"
	FailureReasonKey       = "status.failureReason"
	StartTimestampKey      = "status.startTimestamp"
	CompletionTimestampKey = "status.completionTimestamp"
)

var ErrConfigMapDataIsNil = errors.New("configMap's Data field is nil")

type Reporter struct {
	client             kubernetes.Interface
	configMapNamespace string
	configMapName      string
	reportee           reportee
}

func New(client kubernetes.Interface, configMapNamespace, configMapName string, reportee reportee) *Reporter {
	return &Reporter{
		client:             client,
		configMapNamespace: configMapNamespace,
		configMapName:      configMapName,
		reportee:           reportee,
	}
}

func (r *Reporter) Report() error {
	configMap, err := configmap.Get(r.client.CoreV1(), r.configMapNamespace, r.configMapName)
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		return ErrConfigMapDataIsNil
	}

	if succeeded := r.reportee.Succeeded(); succeeded != "" {
		configMap.Data[SucceededKey] = succeeded
	}

	configMap.Data[FailureReasonKey] = r.reportee.FailureReason()

	startTimestamp := r.reportee.StartTimestamp()
	if !startTimestamp.IsZero() {
		configMap.Data[StartTimestampKey] = startTimestamp.Format(time.RFC3339)
	}

	completionTimestamp := r.reportee.CompletionTimestamp()
	if !completionTimestamp.IsZero() {
		configMap.Data[CompletionTimestampKey] = completionTimestamp.Format(time.RFC3339)
	}

	if _, err := configmap.Update(r.client.CoreV1(), configMap); err != nil {
		return err
	}

	return nil
}
