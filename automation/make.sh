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

CORE_IMAGE_NAME="kiagnose"
CORE_IMAGE_TAG=${CORE_IMAGE_TAG:-devel}
CORE_IMAGE="${IMAGE_REGISTRY}/${IMAGE_ORG}/${CORE_IMAGE_NAME}:${CORE_IMAGE_TAG}"

CORE_BINARY_NAME="kiagnose"

options=$(getopt --options "" \
    --long lint,unit-test,build-core,build-core-image,push-core-image,e2e,help\
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
    --build-core)
        OPT_BUILD_CORE=1
        ;;
    --build-core-image)
        OPT_BUILD_CORE_IMAGE=1
        ;;
    --push-core-image)
        OPT_PUSH_CORE_IMAGE=1
        ;;
    --e2e)
        OPT_E2E=1
        ;;
    --help)
        set +x
        echo "$0 [--lint] [--unit-test] [--e2e] [--build-core] [--build-core-image] [--push-core-image]"
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
    OPT_BUILD_CORE=1
fi

if [ -n "${OPT_LINT}" ]; then
    golangci_lint_version=v1.45.2
    if [ ! -f $(go env GOPATH)/bin/golangci-lint ]; then
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $golangci_lint_version
    fi
    golangci-lint run kiagnose/... cmd/...
fi

if [ -n "${OPT_UNIT_TEST}" ]; then
    go test -v ${PWD}/kiagnose/...
fi

if [ -n "${OPT_BUILD_CORE}" ]; then
    echo "Trying to build \"${CORE_BINARY_NAME}\"..."
    go build -v -o ${PWD}/bin/${CORE_BINARY_NAME} ${PWD}/cmd/
    echo "Successfully built \"${CORE_BINARY_NAME}\""
fi

if [ -n "${OPT_BUILD_CORE_IMAGE}" ]; then
    echo "Trying to build image \"${CORE_IMAGE}\"..."
    ${CRI} build . --file dockerfiles/Dockerfile.kiagnose --tag "${CORE_IMAGE}"
fi

if [ -n "${OPT_PUSH_CORE_IMAGE}" ]; then
    echo "Pushing \"${CORE_IMAGE}\"..."
    ${CRI} push ${CORE_IMAGE}
fi

if [ -n "${OPT_E2E}" ]; then
    ${SCRIPT_PATH}/e2e.sh $@
fi
