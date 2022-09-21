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

from ocp_resources.role import Role as baseRole


class Role(baseRole):
    def __init__(self, client, name, namespace, rules=(), **kwargs):
        super().__init__(client=client, name=name, namespace=namespace, **kwargs)
        self.rules = list(rules)

    def to_dict(self):
        self.res = super().to_dict()
        if self.yaml_file:
            return self.res

        if self.rules:
            self._set_rules()

        return self.res

    def add_rule(self, api_groups, resources, verbs):
        self.rules.append(
            {
                "apiGroups": api_groups,
                "resources": resources,
                "verbs": verbs,
            }
        )

    def _set_rules(self):
        self.res["rules"] = self.rules
