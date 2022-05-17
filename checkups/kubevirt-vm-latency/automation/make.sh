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

ARGCOUNT=$#

SCRIPT_PATH=$(dirname $(realpath -s $0))

CRI=${CRI:-podman}

IMAGE_REGISTRY=${IMAGE_REGISTRY:-quay.io}
IMAGE_ORG=${IMAGE_ORG:-kiagnose}

CHECKUP_BINARY_NAME="kubevirt-vm-latency"
CHECKUP_IMAGE_NAME="kubevirt-vm-latency"
CHECKUP_IMAGE_TAG=${CORE_IMAGE_TAG:-devel}
CHECKUP_IMAGE=${IMAGE_REGISTRY}/${IMAGE_ORG}/${CHECKUP_IMAGE_NAME}:${CHECKUP_IMAGE_TAG}

options=$(getopt --options "" \
    --long lint,unit-test,build-checkup,build-checkup-image,push-checkup-image,e2e,help\
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
    --build-checkup)
        OPT_BUILD_CHECKUP=1
        ;;
    --build-checkup-image)
        OPT_BUILD_CHECKUP_IMAGE=1
        ;;
    --push-checkup-image)
        OPT_PUSH_CHECKUP_IMAGE=1
        ;;
    --e2e)
        OPT_E2E=1
        ;;
    --help)
        set +x
        echo "$0 [--lint] [--unit-test] [--e2e] [--build-checkup] [--build-checkup-image] [--push-checkup-image]"
        exit
        ;;
    --)
        shift
        break
        ;;
    esac
    shift
done

if [ "${ARGCOUNT}" -eq "0" ] ; then
    OPT_LINT=1
    OPT_UNIT_TEST=1
    OPT_BUILD_CHECKUP=1
fi

if [ -n "${OPT_LINT}" ]; then
    golangci_lint_version=v1.45.2
    if [ ! -f $(go env GOPATH)/bin/golangci-lint ]; then
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $golangci_lint_version
    fi
    golangci-lint run vmlatency/... cmd/...
fi

if [ -n "${OPT_UNIT_TEST}" ]; then
    go test -v ./...
fi

if [ -n "${OPT_BUILD_CHECKUP}" ]; then
  echo "Trying to build \"${CHECKUP_BINARY_NAME}\"..."
  go build -v -o ./bin/${CHECKUP_BINARY_NAME} ./cmd/
  echo "Successfully built \"${CHECKUP_BINARY_NAME}\""
fi

if [ -n "${OPT_BUILD_CHECKUP_IMAGE}" ]; then
    echo "Trying to build image \"${CHECKUP_IMAGE}\"..."
    ${CRI} build . --file Dockerfile --tag "${CHECKUP_IMAGE}"
fi

if [ -n "${OPT_PUSH_CHECKUP_IMAGE}" ]; then
    echo "Pushing \"${CHECKUP_IMAGE}\"..."
    ${CRI} push ${CHECKUP_IMAGE}
fi

if [ -n "${OPT_E2E}" ]; then
    ${SCRIPT_PATH}/e2e.sh $@
fi
