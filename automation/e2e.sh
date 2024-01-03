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

CRI=${CRI:-podman}

KUBECTL_VERSION=${KUBECTL_VERSION:-v1.27.9}
KUBECTL=${KUBECTL:-$PWD/kubectl}

KIND_VERSION=${KIND_VERSION:-v0.20.0}
KIND=${KIND:-$PWD/kind}
CLUSTER_NAME=${CLUSTER_NAME:-kind}

KOKO_VERSION=${KOKO_VERSION:-0.83}
KOKO=${KOKO:-$PWD/koko}
BRIDGE_NAME=${BRIDGE_NAME:-br10}
VETH_NAME=${VETH_NAME:-link_${BRIDGE_NAME}}

options=$(getopt --options "" \
    --long install-kind,install-kubectl,create-cluster,create-multi-node-cluster,delete-cluster,build-test-image,help\
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
    --create-multi-node-cluster)
        OPT_CREATE_MULTI_NODE_CLUSTER=1
        ;;
    --delete-cluster)
        OPT_DELETE_CLUSTER=1
        ;;
    --build-test-image)
        OPT_BUILD_TEST_IMAGE=1
        ;;
    --help)
        set +x
        echo "$0 [--install-kind] [--install-kubectl] [--create-cluster] [--create-multi-node-cluster] [--delete-cluster] [--build-test-image]"
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
    OPT_DELETE_CLUSTER=1
fi

if [ -n "${OPT_INSTALL_KIND}" ]; then
    if [ ! -f "${KIND}" ]; then
        curl -Lo "${KIND}" https://kind.sigs.k8s.io/dl/"${KIND_VERSION}"/kind-linux-amd64
        chmod +x "${KIND}"
        echo "kind installed successfully at ${KIND}"
        ${KIND} version
    fi
fi

if [ -n "${OPT_INSTALL_KUBECTL}" ]; then
    if [ ! -f "${KUBECTL}" ]; then
        curl -Lo "${KUBECTL}" https://dl.k8s.io/release/"${KUBECTL_VERSION}"/bin/linux/amd64/kubectl
        chmod +x "${KUBECTL}"
        echo "kubectl installed successfully at ${KUBECTL}"
        ${KUBECTL} version --client
    fi
fi

if [ -n "${OPT_CREATE_CLUSTER}" ]; then
    if ! ${KIND} get clusters | grep "${CLUSTER_NAME}"; then
        ${KIND} create cluster --wait 2m
        echo "Waiting for the network to be ready..."
        ${KUBECTL} wait --for=condition=ready pods --namespace=kube-system -l k8s-app=kube-dns --timeout=2m
        echo "K8S cluster is up:"
        ${KUBECTL} get nodes -o wide
    else
        echo "Cluster '${CLUSTER_NAME}' already exists!"
    fi
fi

if [ -n "${OPT_CREATE_MULTI_NODE_CLUSTER}" ]; then
    if ${KIND} get clusters | grep "${CLUSTER_NAME}"; then
        echo "Cluster '${CLUSTER_NAME}' already exists!"
    else
        cat <<EOF | ${KIND} create cluster --wait 2m --config -
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
        echo "Waiting for the network to be ready..."
        ${KUBECTL} wait --for=condition=ready pods --namespace=kube-system -l k8s-app=kube-dns --timeout=2m
        echo "K8S cluster is up:"
        ${KUBECTL} get nodes -o wide

        if [ ! -f "${KOKO}" ]; then
            curl -Lo "${KOKO}" https://github.com/redhat-nfvpe/koko/releases/download/v${KOKO_VERSION}/koko_${KOKO_VERSION}_linux_amd64
            chmod +x "${KOKO}"
            echo "koko installed successfully at ${KOKO}"
        fi

        echo "Interconnect worker nodes with veth pair, requires root privileges"
        worker1="kind-worker"
        worker2="kind-worker2"
        worker1_pid=$(${CRI} inspect --format "{{ .State.Pid }}" "${worker1}")
        worker2_pid=$(${CRI} inspect --format "{{ .State.Pid }}" "${worker2}")
        sudo ${KOKO} -p "${worker1_pid},${VETH_NAME}" -p "${worker2_pid},${VETH_NAME}"

        echo "Establish connectivity between worker nodes over bridge network"
        for node in "${worker1}" "${worker2}"; do
            ${CRI} exec ${node} ip link add ${BRIDGE_NAME} type bridge
            ${CRI} exec ${node} ip link set ${VETH_NAME} master ${BRIDGE_NAME}
            ${CRI} exec ${node} ip link set up ${BRIDGE_NAME}
        done
    fi
fi

if [ -n "${OPT_BUILD_TEST_IMAGE}" ]; then
  ${CRI} build -f ./test/infra/Dockerfile -t kiagnose-e2e-test ./test/infra
fi

if [ -n "${OPT_DELETE_CLUSTER}" ]; then
    ${KIND} delete cluster
fi
