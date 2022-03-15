#!/usr/bin/env bash
#
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
#

set -e

options=$(getopt --options "" \
    --long unit-test,help\
    -- "${@}")
eval set -- "$options"
while true; do
    case "$1" in
    --lint)
        OPT_LINT=1
        ;;
    --unit-test)
        OPT_UNIT_TEST=1
        ;;
    --help)
        set +x
        echo "$0 [--lint] [--unit-test]"
        exit
        ;;
    --)
        shift
        break
        ;;
    esac
    shift
done

if  [ -z "${OPT_LINT}" ] && [ -z "${OPT_UNIT_TEST}" ]; then
    OPT_LINT=1
    OPT_UNIT_TEST=1
fi

if [ -n "${OPT_LINT}" ]; then
    golangci_lint_version=v1.44.2
    if [ ! -f $(go env GOPATH)/bin/golangci-lint ]; then
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $golangci_lint_version
    fi
    golangci-lint run
fi

if [ -n "${OPT_UNIT_TEST}" ]; then
    go test -v ./kiagnose/...
fi
