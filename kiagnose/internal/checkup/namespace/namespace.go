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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	k8swatch "k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	k8swatchtools "k8s.io/client-go/tools/watch"
)

func Create(client kubernetes.Interface, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	namespace, err := client.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Printf("Namespace %q successfully created", namespace.Name)

	return namespace, nil
}

// DeleteAndWait delete and waits for the given namespace to dispose.
func DeleteAndWait(client kubernetes.Interface, namespace *corev1.Namespace, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.CoreV1().Namespaces().Delete(ctx, namespace.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	log.Printf("delete namespace %q request sent", namespace.Name)

	nsLabel := k8slabels.Set{corev1.LabelMetadataName: namespace.Name}
	w := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (k8swatch.Interface, error) {
			if options.LabelSelector != "" {
				options.LabelSelector = "," + nsLabel.String()
			} else {
				options.LabelSelector = nsLabel.String()
			}
			return client.CoreV1().Namespaces().Watch(ctx, options)
		},
	}
	_, err := k8swatchtools.Until(ctx, namespace.ResourceVersion, w, func(event k8swatch.Event) (bool, error) {
		if event.Type != k8swatch.Deleted {
			return false, nil
		}
		_, ok := event.Object.(*corev1.Namespace)
		return ok, nil
	})
	if err == nil {
		log.Printf("Namespace %q deleted", namespace.Name)
	}
	return err
}
