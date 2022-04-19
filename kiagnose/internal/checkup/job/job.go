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
	"encoding/json"
	"fmt"
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swatch "k8s.io/apimachinery/pkg/watch"
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

func WaitForJobToFinish(client batchv1client.BatchV1Interface, job *batchv1.Job, timeout time.Duration) (*batchv1.Job, error) {
	const JobNameLabel = "job-name"

	jobLabel := fmt.Sprintf("%s=%s", JobNameLabel, job.Name)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	jobWatcher, err := client.Jobs(job.Namespace).Watch(ctx, metav1.ListOptions{LabelSelector: jobLabel})
	if err != nil {
		return nil, err
	}
	log.Printf("'%s/%s' Job watcher obtained", job.Namespace, job.Name)

	finishedJob, err := waitForJob(ctx, jobWatcher)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for Job '%s/%s' to finish: %v", job.Namespace, job.Name, err)
	}
	log.Printf("Job '%s/%s' is finished", job.Namespace, job.Name)

	return finishedJob, nil
}

func waitForJob(ctx context.Context, watcher k8swatch.Interface) (*batchv1.Job, error) {
	eventsCh := watcher.ResultChan()
	defer watcher.Stop()
	for {
		select {
		case event := <-eventsCh:
			job, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}
			if jobStatus, err := json.MarshalIndent(job.Status, "", " "); err == nil {
				log.Printf("received job event '%s/%s': \n%v\n", job.Namespace, job.Name, string(jobStatus))
			}
			if finished(job) {
				return job, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func finished(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed {
			return true
		}
	}
	return false
}
