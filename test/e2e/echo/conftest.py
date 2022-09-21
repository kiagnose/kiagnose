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

from ocp_resources.namespace import Namespace
from ocp_resources.service_account import ServiceAccount


TARGET_NAMESPACE = "checkup-e2e-test"
TARGET_SERVICE_ACCOUNT = "checkup-e2e-test"


@pytest.fixture
def target_ns(kclient) -> Namespace:
    with Namespace(TARGET_NAMESPACE, client=kclient) as ns:
        yield ns


@pytest.fixture
def target_sa(kclient, target_ns) -> ServiceAccount:
    with ServiceAccount(TARGET_SERVICE_ACCOUNT, target_ns.name, client=kclient) as sa:
        yield sa
