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

from ocp_resources.configmap import ConfigMap
from ocp_resources.resource import NamespacedResource

from .job import job
from . import vmi


def test_successful_run(kclient, checkup_sa, checkup_role_binding, configmap_role_binding, nad):
    namespace = checkup_sa.namespace
    config_map = ConfigMap(
        name="vmlatency-checkup-test",
        namespace=namespace,
        data={
            "spec.timeout": "5m",
            "spec.param.networkAttachmentDefinitionNamespace": nad.namespace,
            "spec.param.networkAttachmentDefinitionName": nad.name,
            "spec.param.maxDesiredLatencyMilliseconds": "500",
            "spec.param.sampleDurationSeconds": "5",
        },
        client=kclient,
    )
    with resource_dump(config_map) as cm:
        with job(kclient, namespace, checkup_sa.name, cm) as j:
            timeout = 480
            attempts = 4
            attempt_timeout = timeout / attempts
            for attempt in range(attempts):
                try:
                    with vmi.teardown_logging(kclient):
                        j.wait_for_condition(
                            condition=j.Condition.COMPLETE,
                            status=j.Condition.Status.TRUE,
                            timeout=attempt_timeout,
                        )
                except Exception as e:
                    print(f"failed on attempt {attempt}: {e}")
                else:
                    print(f"succeeded on attempt {attempt}")
                    break

        data = cm.instance.data
        assert "true" == data.get("status.succeeded")


@contextlib.contextmanager
def resource_dump(resource: NamespacedResource) -> ConfigMap:
    with resource as r:
        try:
            yield r
        finally:
            print(r.instance)
