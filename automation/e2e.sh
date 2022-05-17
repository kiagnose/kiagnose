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

KUBECTL_VERSION=${KUBECTL_VERSION:-v1.23.0}
KUBECTL=${KUBECTL:-$PWD/kubectl}

KIND_VERSION=${KIND_VERSION:-v0.12.0}
KIND=${KIND:-$PWD/kind}

options=$(getopt --options "" \
    --long install-kind,install-kubectl,create-cluster,delete-cluster,deploy-kiagnose,run-tests,help\
    -- "${@}")
eval set -- "$options"
while true; do
    case "$1" in
    --install-kind)
        OPT_INSTALL_KIND=1
        ;;
    --install-kubectl)
        OPT_INSTALL_KUBECTL=1
        ;;
    --create-cluster)
        OPT_CREATE_CLUSTER=1
        ;;
    --delete-cluster)
        OPT_DELETE_CLUSTER=1
        ;;
    --deploy-kiagnose)
        OPT_DEPLOY_KIAGNOSE=1
        ;;
    --run-tests)
        OPT_RUN_TEST=1
        ;;
    --help)
        set +x
        echo "$0 [--install-kind] [--install-kubectl] [--create-cluster] [--delete-cluster] [--deploy-kiagnose] [--run-tests]"
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
    OPT_INSTALL_KIND=1
    OPT_INSTALL_KUBECTL=1
    OPT_CREATE_CLUSTER=1
    OPT_DEPLOY_KIAGNOSE=1
    OPT_RUN_TEST=1
    OPT_DELETE_CLUSTER=1
fi

if [ -n "${OPT_INSTALL_KIND}" ]; then
    if [ ! -f "${KIND}" ]; then
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64
        chmod +x ./kind
        if [ "${KIND}" != "${PWD}/kind" ]; then
            mv ./kind "${KIND}"
        fi
        echo "kind installed successfully at ${KIND}"
    fi
fi

if [ -n "${OPT_INSTALL_KUBECTL}" ]; then
    if [ ! -f "${KUBECTL}" ]; then
        curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
        chmod +x ./kubectl
        if [ "${KUBECTL}" != "${PWD}/kubectl" ]; then
            mv ./kubectl "${KUBECTL}"
        fi
        echo "kubectl installed successfully at ${KUBECTL}"
    fi
fi

if [ -n "${OPT_CREATE_CLUSTER}" ]; then
    CLUSTER_NAME=kind
    if ! ${KIND} get clusters | grep ${CLUSTER_NAME}; then
        ${KIND} create cluster --wait 2m
        echo "Waiting for the network to be ready..."
        ${KUBECTL} wait --for=condition=ready pods --namespace=kube-system -l k8s-app=kube-dns --timeout=2m
        echo "K8S cluster is up:"
        ${KUBECTL} get nodes -o wide
    else
        echo "Cluster '${CLUSTER_NAME}' already exists!"
    fi
fi

if [ -n "${OPT_DEPLOY_KIAGNOSE}" ]; then
    ${KUBECTL} apply -f manifests/kiagnose.yaml
fi

if [ -n "${OPT_RUN_TEST}" ]; then
    # kiagnose sanity e2e test uses the echo-checkup
    cd checkups/echo

    echo "Post echo checkup ConfigMap & Job:"
    echo
    ${KUBECTL} create -f manifests/echo-checkup.yaml

    KIAGNOSE_NAMESPACE=kiagnose
    KIAGNOSE_JOB=echo-checkup1
    ECHO_CONFIGMAP=echo-checkup-config

    ${KUBECTL} wait --for=condition=complete --timeout=1m job.batch/${KIAGNOSE_JOB} -n ${KIAGNOSE_NAMESPACE}

    echo
    echo "Result:"
    echo
    ${KUBECTL} get configmap ${ECHO_CONFIGMAP} -n ${KIAGNOSE_NAMESPACE} -o yaml
    ${KUBECTL} get configmap ${ECHO_CONFIGMAP} -n ${KIAGNOSE_NAMESPACE} -o yaml | grep -q "status.result.echo: Hi!"
    echo
    echo "Cleanup:"
    echo
    ${KUBECTL} delete -f manifests/echo-checkup.yaml

    cd -
fi

if [ -n "${OPT_DELETE_CLUSTER}" ]; then
    ${KIND} delete cluster
fi
