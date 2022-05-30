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
	"fmt"
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	k8swatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	k8swatchtools "k8s.io/client-go/tools/watch"
)

func Create(client kubernetes.Interface, job *batchv1.Job) (*batchv1.Job, error) {
	job, err := client.BatchV1().Jobs(job.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Job '%s/%s' successfully created", job.Namespace, job.Name)
	return job, nil
}

func WaitForJobToFinish(client kubernetes.Interface, job *batchv1.Job, timeout time.Duration) (*batchv1.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	const JobNameLabel = "job-name"
	jobLabel := k8slabels.Set{JobNameLabel: job.Name}
	w := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (k8swatch.Interface, error) {
			if options.LabelSelector != "" {
				options.LabelSelector = "," + jobLabel.String()
			} else {
				options.LabelSelector = jobLabel.String()
			}
			return client.BatchV1().Jobs(job.Namespace).Watch(ctx, options)
		},
	}

	event, err := k8swatchtools.Until(ctx, job.ResourceVersion, w, batchJobFinished)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for Job '%s/%s' to finish: %v", job.Namespace, job.Name, err)
	}

	updatedJob, ok := event.Object.(*batchv1.Job)
	if !ok {
		return nil, fmt.Errorf("failed to wait for Job '%s/%s' to finish: wrong event", job.Namespace, job.Name)
	}

	log.Printf("Job '%s/%s' is finished", job.Namespace, job.Name)

	return updatedJob, nil
}

func batchJobFinished(event k8swatch.Event) (bool, error) {
	j, ok := event.Object.(*batchv1.Job)
	if !ok {
		return false, nil
	}

	switch event.Type {
	case k8swatch.Deleted:
		return false, fmt.Errorf("unexpected event: %+v", event)
	case k8swatch.Added, k8swatch.Modified:
		return finished(j), nil
	case k8swatch.Bookmark, k8swatch.Error:
	}
	return false, nil
}

func finished(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed) &&
			condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
