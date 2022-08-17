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

ECHO_IMAGE_NAME="echo-checkup"
ECHO_IMAGE_TAG=${ECHO_IMAGE_TAG:-devel}
ECHO_IMAGE="${IMAGE_REGISTRY}/${IMAGE_ORG}/${ECHO_IMAGE_NAME}:${ECHO_IMAGE_TAG}"

options=$(getopt --options "" \
    --long unit-test,build-checkup-image,push-checkup-image,e2e,help\
    -- "${@}")
eval set -- "$options"
while true; do
    case "$1" in
    --unit-test)
        OPT_UNIT_TEST=1
        ;;
    --build-checkup-image)
        OPT_BUILD_IMAGE=1
        ;;
    --push-checkup-image)
        OPT_PUSH_IMAGE=1
        ;;
    --e2e)
        OPT_E2E=1
        ;;
    --help)
        set +x
        echo "$0 [--unit-test] [--e2e] [--build-checkup-image] [--push-checkup-image]"
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
    OPT_UNIT_TEST=1
fi

if [ -n "${OPT_UNIT_TEST}" ]; then
  ./entrypoint_test
fi

if [ -n "${OPT_BUILD_IMAGE}" ]; then
    echo "Trying to build image \"${ECHO_IMAGE}\"..."
    ${CRI} build . --file Dockerfile --tag "${ECHO_IMAGE}"
fi

if [ -n "${OPT_PUSH_IMAGE}" ]; then
   echo "Pushing \"${ECHO_IMAGE}\"..."
   ${CRI} push ${ECHO_IMAGE}
fi

if [ -n "${OPT_E2E}" ]; then
    ${SCRIPT_PATH}/e2e.sh $@
fi
