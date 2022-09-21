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

from ocp_resources.resource import get_client
from ocp_resources.namespace import Namespace
from ocp_resources.service_account import ServiceAccount
from ocp_resources.cluster_role import ClusterRole
from ocp_resources.cluster_role_binding import ClusterRoleBinding

import pytest


@pytest.fixture(scope="session")
def kclient():
    return get_client()


@pytest.fixture(scope="session")
def kiagnose_cluster_role(kclient) -> ClusterRole:
    cluster_role = ClusterRole(
        name="kiagnose",
        client=kclient,
        api_groups=[""],
        permissions_to_resources=["configmaps"],
        verbs=["get", "list", "create", "delete", "update", "patch"],
    )
    cluster_role.add_rule(
        api_groups=["rbac.authorization.k8s.io"],
        permissions_to_resources=["roles", "rolebindings"],
        verbs=["get", "list", "create", "delete"],
    )
    cluster_role.add_rule(
        api_groups=["batch"],
        permissions_to_resources=["jobs"],
        verbs=["get", "list", "create", "delete", "watch"],
    )

    with cluster_role as cr:
        yield cr


@pytest.fixture(scope="session")
def kiagnose_namespace(kclient) -> Namespace:
    with Namespace("kiagnose", client=kclient) as ns:
        yield ns


@pytest.fixture(scope="session")
def kiagnose_deployment(kclient, kiagnose_namespace, kiagnose_cluster_role) -> dict:
    with ServiceAccount("kiagnose", kiagnose_namespace.name, client=kclient) as sa:
        with ClusterRoleBinding(
            name="kiagnose",
            cluster_role=kiagnose_cluster_role.name,
            subjects=[
                {
                    "kind": "ServiceAccount",
                    "name": sa.name,
                    "namespace": kiagnose_namespace.name,
                },
            ],
        ) as crb:
            yield {
                "namespace": kiagnose_namespace,
                "serviceaccount": sa,
                "clusterrole": kiagnose_cluster_role,
                "clusterrolebinding": crb,
            }
