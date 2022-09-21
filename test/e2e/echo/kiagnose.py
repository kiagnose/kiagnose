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

import os

from ocp_resources.configmap import ConfigMap
from ocp_resources.job import Job
from openshift.dynamic import DynamicClient

FRAMEWORK_IMAGE = os.getenv("KIAGNOSE_IMAGE", "quay.io/kiagnose/kiagnose:devel")
CHECKUP_IMAGE = os.getenv("CHECKUP_IMAGE", "quay.io/kiagnose/echo-checkup:devel")

ENV_CONFIGMAP_NAMESPACE_KEY = "CONFIGMAP_NAMESPACE"
ENV_CONFIGMAP_NAME_KEY = "CONFIGMAP_NAME"


def job(
    client: DynamicClient, namespace: str, service_account: str, configmap: ConfigMap
) -> Job:
    return Job(
        name="test-checkup",
        namespace=namespace,
        client=client,
        backoff_limit=0,
        service_account=service_account,
        restart_policy="Never",
        containers=[
            {
                "name": "framework",
                "image": FRAMEWORK_IMAGE,
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
    )
