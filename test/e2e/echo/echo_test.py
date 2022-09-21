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

from ocp_resources.configmap import ConfigMap

from . import kiagnose


def test_successful_echo(kclient, kiagnose_deployment, target_ns, target_sa):
    with ConfigMap(
        name="echo-checkup-test",
        namespace=target_ns.name,
        data={
            "spec.image": kiagnose.CHECKUP_IMAGE,
            "spec.timeout": "1m",
            "spec.serviceAccountName": target_sa.name,
            "spec.param.message": "Hi!",
        },
        client=kclient,
    ) as cm:
        with kiagnose.job(
            kclient,
            kiagnose_deployment["namespace"].name,
            kiagnose_deployment["serviceaccount"].name,
            cm,
        ) as j:
            j.wait_for_condition(
                condition=j.Condition.COMPLETE,
                status=j.Condition.Status.TRUE,
                timeout=30,
            )

            data = cm.instance.data
            assert "true" == data.get("status.succeeded")
            assert "Hi!" == data.get("status.result.echo")
