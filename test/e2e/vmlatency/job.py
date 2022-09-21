# This file is part of the kiagnose project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2022 Red Hat, Inc.

import contextlib
import os

from ocp_resources.configmap import ConfigMap
from ocp_resources.job import Job
from ocp_resources.pod import Pod

from openshift.dynamic import DynamicClient

CHECKUP_IMAGE = os.getenv("CHECKUP_IMAGE", "quay.io/kiagnose/kubevirt-vm-latency:main")

ENV_CONFIGMAP_NAMESPACE_KEY = "CONFIGMAP_NAMESPACE"
ENV_CONFIGMAP_NAME_KEY = "CONFIGMAP_NAME"


@contextlib.contextmanager
def job(
    client: DynamicClient, namespace: str, service_account: str, configmap: ConfigMap
) -> Job:
    with Job(
        name="test-checkup",
        namespace=namespace,
        client=client,
        backoff_limit=0,
        service_account=service_account,
        restart_policy="Never",
        containers=[
            {
                "name": "vmlatency-checkup",
                "image": CHECKUP_IMAGE,
                "env": [
                    {
                        "name": ENV_CONFIGMAP_NAMESPACE_KEY,
                        "value": configmap.namespace,
                    },
                    {
                        "name": ENV_CONFIGMAP_NAME_KEY,
                        "value": configmap.name,
                    },
                ],
            }
        ],
    ) as j:
        try:
            yield j
        finally:
            print(j.instance)
            print_logs(client, j)


def print_logs(client: DynamicClient, j: Job):
    pods = list(Pod.get(dyn_client=client, label_selector=f"job-name={j.name}"))
    print(f"Checkup job pods: {[p.name for p in pods]}")
    for pod in pods:
        print(f"pod {pod.name} log:\n {pod.log()}")
