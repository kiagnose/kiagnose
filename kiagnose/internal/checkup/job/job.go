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

package job

import (
	"context"
	"log"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
)

func Create(client batchv1client.BatchV1Interface, job *batchv1.Job) (*batchv1.Job, error) {
	job, err := client.Jobs(job.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Job '%s/%s' successfully created", job.Namespace, job.Name)
	return job, nil
}
