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
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kiagnosev1alpha1 "github.com/kiagnose/kiagnose/api/v1alpha1"
	"github.com/kiagnose/kiagnose/kiagnose/internal/status"
)

const (
	SucceededKey           = "status.succeeded"
	FailureReasonKey       = "status.failureReason"
	StartTimestampKey      = "status.startTimestamp"
	CompletionTimestampKey = "status.completionTimestamp"
)

var ErrConfigMapDataIsNil = errors.New("configMap Data is nil")

type Reporter struct {
	client     client.Client
	checkupKey types.NamespacedName
}

func New(client client.Client, checkupKey types.NamespacedName) *Reporter {
	return &Reporter{
		checkupKey: checkupKey,
		client:     client,
	}
}

func (r *Reporter) Report(statusData status.Status) error {
	cr := kiagnosev1alpha1.Checkup{}
	err := r.client.Get(context.Background(), r.checkupKey, &cr)
	if err != nil {
		return err
	}

	if !statusData.StartTimestamp.IsZero() {
		cr.Status.StartTime = metav1.Time{statusData.StartTimestamp}
	}

	if !statusData.CompletionTimestamp.IsZero() {
		cr.Status.CompletionTime = metav1.Time{statusData.CompletionTimestamp}
		if statusData.Succeeded {
			cr.Status.Conditions.Set(kiagnosev1alpha1.CheckupConditionSuccess, corev1.ConditionTrue, kiagnosev1alpha1.CheckupConditionSuccessfullyRun, "")
			cr.Status.Conditions.Set(kiagnosev1alpha1.CheckupConditionFailing, corev1.ConditionFalse, kiagnosev1alpha1.CheckupConditionSuccessfullyRun, "")
		} else {
			cr.Status.Conditions.Set(kiagnosev1alpha1.CheckupConditionFailing, corev1.ConditionTrue, kiagnosev1alpha1.CheckupConditionFailtedToRun, statusData.FailureReason)
			cr.Status.Conditions.Set(kiagnosev1alpha1.CheckupConditionSuccess, corev1.ConditionFalse, kiagnosev1alpha1.CheckupConditionFailtedToRun, statusData.FailureReason)
		}
	}

	err = r.client.Status().Update(context.Background(), &cr)
	if err != nil {
		return err
	}
	return nil
}
