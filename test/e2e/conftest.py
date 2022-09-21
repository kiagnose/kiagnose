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

import pytest

from ocp_resources.resource import get_client

from .role import Role


@pytest.fixture(scope="session")
def kclient():
    return get_client()


@pytest.fixture(scope="session")
def checkup_configmap_role(kclient, project_ns) -> Role:
    with Role(
        client=kclient,
        name="checkup-configmap-access",
        namespace=project_ns.name,
        rules=[
            {"apiGroups": [""], "resources": ["configmaps"], "verbs": ["get", "update"]}
        ],
    ) as r:
        yield r
