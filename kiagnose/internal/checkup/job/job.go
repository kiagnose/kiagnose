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
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/tools/cache"
)

func Create(client batchv1client.BatchV1Interface, job *batchv1.Job) (*batchv1.Job, error) {
	job, err := client.Jobs(job.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Job '%s/%s' successfully created", job.Namespace, job.Name)
	return job, nil
}

func WaitForJobToFinish(lw cache.ListerWatcher, timeout time.Duration) (*batchv1.Job, error) {
	const rsyncPeriod = time.Minute * 5

	jobCh := make(chan interface{}, 1)
	jobEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { jobCh <- obj },
		UpdateFunc: func(oldObj, newObj interface{}) { jobCh <- newObj },
	}
	_, controller := cache.NewInformer(lw, &batchv1.Job{}, rsyncPeriod, jobEventHandler)

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	go controller.Run(ctx.Done())

	for {
		select {
		case obj := <-jobCh:
			job := obj.(*batchv1.Job)
			if raw, err := json.MarshalIndent(job.Status, "", " "); err == nil {
				log.Printf("received job event '%s/%s', job status: \n%v\n", job.Namespace, job.Name, string(raw))
			}
			if finished(job) {
				log.Printf("job '%s/%s' is finished", job.Namespace, job.Name)
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
