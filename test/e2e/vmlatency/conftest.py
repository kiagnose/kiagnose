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

import pytest

from ocp_resources.namespace import Namespace
from ocp_resources.service_account import ServiceAccount
from ocp_resources.role_binding import RoleBinding
from ocp_resources.network_attachment_definition import NetworkAttachmentDefinition

from ..role import Role

VM_LATENCY_CHECKUP_NAME = "kubevirt-vm-latency-checkup"

PROJECT_NAMESPACE = "target-ns"
CHECKUP_SERVICE_ACCOUNT = "vm-latency-checkup"

NET_ATTACH_DEF_NAME = os.getenv("NET_ATTACH_DEF_NAME", "bridge-network")


@pytest.fixture(scope="session")
def project_ns(kclient) -> Namespace:
    return Namespace(PROJECT_NAMESPACE, client=kclient)


@pytest.fixture(scope="session")
def checkup_sa(kclient, project_ns) -> ServiceAccount:
    with ServiceAccount(CHECKUP_SERVICE_ACCOUNT, project_ns.name, client=kclient) as sa:
        yield sa


@pytest.fixture(scope="session")
def checkup_role(kclient, project_ns) -> Role:
    role = Role(
        client=kclient,
        name=VM_LATENCY_CHECKUP_NAME,
        namespace=project_ns.name,
    )
    role.add_rule(
        api_groups=["kubevirt.io"],
        resources=["virtualmachineinstances"],
        verbs=["get", "create", "delete"],
    )
    role.add_rule(
        api_groups=["subresources.kubevirt.io"],
        resources=["virtualmachineinstances/console"],
        verbs=["get"],
    )
    role.add_rule(
        api_groups=["k8s.cni.cncf.io"],
        resources=["network-attachment-definitions"],
        verbs=["get"],
    )

    with role as r:
        yield r


@pytest.fixture(scope="session")
def configmap_role_binding(kclient, checkup_configmap_role, checkup_sa) -> RoleBinding:
    with RoleBinding(
        name=checkup_configmap_role.name,
        namespace=checkup_configmap_role.namespace,
        client=kclient,
        role_ref_kind="Role",
        role_ref_name=checkup_configmap_role.name,
        subjects_kind="ServiceAccount",
        subjects_name=checkup_sa.name,
    ) as rb:
        yield rb


@pytest.fixture(scope="session")
def checkup_role_binding(kclient, checkup_role, checkup_sa) -> RoleBinding:
    with RoleBinding(
        name=VM_LATENCY_CHECKUP_NAME,
        namespace=checkup_role.namespace,
        client=kclient,
        role_ref_kind="Role",
        role_ref_name=checkup_role.name,
        subjects_kind="ServiceAccount",
        subjects_name=checkup_sa.name,
    ) as rb:
        yield rb


@pytest.fixture(scope="session")
def nad(project_ns) -> NetworkAttachmentDefinition:
    return NetworkAttachmentDefinition(
        name=NET_ATTACH_DEF_NAME,
        namespace=project_ns.name,
    )
