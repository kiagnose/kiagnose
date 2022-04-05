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

package namespace

import (
	"context"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

func Create(client corev1client.CoreV1Interface, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	var err error

	namespace, err = client.Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Namespace '%s' successfully created", namespace.Name)

	return namespace, nil
}

// DeleteAndWait delete and waits for the given namespace to dispose.
func DeleteAndWait(client corev1client.CoreV1Interface, nsName string, timeout time.Duration) error {
	if err := client.Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	log.Printf("deleted namespace '%s' request sent", nsName)

	if err := waitForDeletion(client, nsName, timeout); err != nil {
		return err
	}
	log.Printf("namespace '%s' successfully deleted", nsName)

	return nil
}

// waitForDeletion waits until the given namespace is disposed.
func waitForDeletion(client corev1client.CoreV1Interface, nsName string, timeout time.Duration) error {
	log.Printf("waiting for namespace '%s' to dispose", nsName)

	const pollInterval = time.Second * 5
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		_, err := client.Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		namespaceNotFound := errors.IsNotFound(err)
		if err != nil && !namespaceNotFound {
			log.Printf("failed to get namespace '%s' while waiting for it to dispose: %v", nsName, err)
		}
		return namespaceNotFound, nil
	})
}
