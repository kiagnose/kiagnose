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

KUBECTL=${KUBECTL:-$PWD/kubectl}

KIND=${KIND:-$PWD/kind}
CLUSTER_NAME=${CLUSTER_NAME:-kind}

FRAMEWORK_IMAGE="quay.io/kiagnose/kiagnose:devel"
CHECKUP_IMAGE="quay.io/kiagnose/echo-checkup:devel"

KIAGNOSE_NAMESPACE=kiagnose
KIAGNOSE_JOB=echo-checkup
ECHO_CONFIGMAP=echo-checkup
ECHO_SERVICE_ACCOUNT_NAME=echo-sa

TARGET_NAMESPACE="echo-checkup-e2e-test"

options=$(getopt --options "" \
    --long deploy-checkup,run-tests,clean-run,help\
    -- "${@}")
eval set -- "$options"
while true; do
    case "$1" in
    --deploy-checkup)
        OPT_DEPLOY_CHECKUP=1
        ;;
    --run-tests)
        OPT_RUN_TEST=1
        ;;
    --clean-run)
        OPT_CLEAN_RUN=1
        ;;
    --help)
        set +x
        echo -n "$0 [--deploy-checkup] [--run-tests] [--clean-run]"
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
    OPT_DEPLOY_CHECKUP=1
    OPT_RUN_TEST=1
fi

if [ -n "${OPT_DEPLOY_CHECKUP}" ]; then
    echo
    echo "Deploy echo checkup..."
    echo

    ${KIND} load docker-image "${CHECKUP_IMAGE}" --name "${CLUSTER_NAME}"
fi

if [ -n "${OPT_RUN_TEST}" ]; then
    ${KUBECTL} create namespace ${TARGET_NAMESPACE}

    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${ECHO_SERVICE_ACCOUNT_NAME}
  namespace: ${TARGET_NAMESPACE}
EOF

    echo
    echo "Deploy ConfigMap with input data: "
    echo
    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${ECHO_CONFIGMAP}
  namespace: ${TARGET_NAMESPACE}
data:
  spec.image: ${CHECKUP_IMAGE}
  spec.timeout: 1m
  spec.serviceAccountName: ${ECHO_SERVICE_ACCOUNT_NAME}
  spec.param.message: "Hi!"
EOF

    echo
    echo "Deploy and run kiagnose job: "
    echo
    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: batch/v1
kind: Job
metadata:
  name: ${KIAGNOSE_JOB}
  namespace: ${KIAGNOSE_NAMESPACE}
spec:
  backoffLimit: 0
  template:
    spec:
      serviceAccount: kiagnose
      restartPolicy: Never
      containers:
        - name: framework
          image: ${FRAMEWORK_IMAGE}
          env:
            - name: CONFIGMAP_NAMESPACE
              value: ${TARGET_NAMESPACE}
            - name: CONFIGMAP_NAME
              value: ${ECHO_CONFIGMAP}
EOF

    ${KUBECTL} wait --for=condition=complete --timeout=10m job.batch/${KIAGNOSE_JOB} -n ${KIAGNOSE_NAMESPACE}

     echo
     echo "Result:"
     echo
     ${KUBECTL} get configmap ${ECHO_CONFIGMAP} -n ${TARGET_NAMESPACE} -o yaml
     ${KUBECTL} get configmap ${ECHO_CONFIGMAP} -n ${TARGET_NAMESPACE} -o yaml | grep -q "status.result.echo: Hi!"
fi

if [ -n "${OPT_CLEAN_RUN}" ];then
  ${KUBECTL} delete job ${KIAGNOSE_JOB} -n ${KIAGNOSE_NAMESPACE} --ignore-not-found
  ${KUBECTL} delete configmap ${ECHO_CONFIGMAP} -n ${TARGET_NAMESPACE} --ignore-not-found
fi
