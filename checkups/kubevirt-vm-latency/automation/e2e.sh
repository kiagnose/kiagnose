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

KUBECTL_VERSION=${KUBECTL_VERSION:-v1.23.0}
KUBECTL=${KUBECTL:-$PWD/kubectl}

KIND_VERSION=${KIND_VERSION:-v0.12.0}
KIND=${KIND:-$PWD/kind}

KUBEVIRT_VERSION=${KUBEVIRT_VERSION:-v0.53.0}
CNAO_VERSION=${CNAO_VERSION:-v0.74.0}

FRAMEWORK_IMAGE="quay.io/kiagnose/kiagnose:devel"
CHECKUP_IMAGE="quay.io/kiagnose/kubevirt-vm-latency:devel"

options=$(getopt --options "" \
    --long install-kind,install-kubectl,create-cluster,delete-cluster,build-kiagnose,deploy-kiagnose,deploy-kubevirt,deploy-cnao,build-checkup,deploy-checkup,define-nad,run-tests,help\
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
    --build-kiagnose)
        OPT_BUILD_KIAGNOSE=1
        ;;
    --deploy-kiagnose)
        OPT_DEPLOY_KIAGNOSE=1
        ;;
    --deploy-kubevirt)
        OPT_DEPLOY_KUBEVIRT=1
        ;;
    --deploy-cnao)
        OPT_DEPLOY_CNAO=1
        ;;
    --build-checkup)
        OPT_BUILD_CHECKUP=1
        ;;
    --deploy-checkup)
        OPT_DEPLOY_CHECKUP=1
        ;;
    --define-nad)
        OPT_DEFINE_NAD=1
        ;;
    --run-tests)
        OPT_RUN_TEST=1
        ;;
    --help)
        set +x
        echo -n "$0 [--install-kind] [--install-kubectl] "
        echo -n "[--create-cluster] [--delete-cluster] "
        echo -n "[--deploy-kubevirt] [--build-kiagnose] [--deploy-kiagnose] [--deploy-cnao] [--build-checkup] [--deploy-checkup] "
        echo "[--define-nad] [--run-tests]"
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
    OPT_BUILD_KIAGNOSE=1
    OPT_DEPLOY_KUBEVIRT=1
    OPT_DEPLOY_CNAO=1
    OPT_BUILD_CHECKUP=1
    OPT_DEPLOY_CHECKUP=1
    OPT_DEFINE_NAD=1
    OPT_RUN_TEST=1
#    OPT_DELETE_CLUSTER=1
fi

if [ -n "${OPT_INSTALL_KIND}" ]; then
    if [ ! -f "${KIND}" ]; then
        curl -Lo ${KIND} https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64
        chmod +x ${KIND}
        echo "kind installed successfully at ${KIND}"
    fi
fi

if [ -n "${OPT_INSTALL_KUBECTL}" ]; then
    if [ ! -f "${KUBECTL}" ]; then
        curl -Lo ${KUBECTL} https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
        chmod +x ${KUBECTL}
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

if [ -n "${OPT_BUILD_KIAGNOSE}" ]; then
  cd ${SCRIPT_PATH}/../../..
  ./automation/make.sh --build-core --build-core-image
  cd -
fi

if [ -n "${OPT_DEPLOY_KIAGNOSE}" ]; then
  ${KIND} load docker-image "${FRAMEWORK_IMAGE}" --name "${CLUSTER_NAME}"
  ${KUBECTL} apply -f manifests/kiagnose.yaml
fi

if [ -n "${OPT_DEPLOY_KUBEVIRT}" ]; then
    echo
    echo "Deploy kubevirt..."
    echo
    kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml
    kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml
    kubectl patch kubevirt kubevirt --namespace kubevirt --type=merge --patch '{"spec":{"configuration":{"developerConfiguration":{"useEmulation":true}}}}'
    kubectl wait --for=condition=Available kubevirt kubevirt --namespace=kubevirt --timeout=2m

    echo
    echo "Successfully deployed kubevirt:"
    kubectl get pods -n kubevirt
fi

if [ -n "${OPT_DEPLOY_CNAO}" ]; then
    echo
    echo "Deploy CNAO (with multus and bridge CNI/s)..."
    echo
    ${KUBECTL} apply -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/${CNAO_VERSION}/namespace.yaml
    ${KUBECTL} apply -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/${CNAO_VERSION}/network-addons-config.crd.yaml
    ${KUBECTL} apply -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/${CNAO_VERSION}/operator.yaml

    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: networkaddonsoperator.network.kubevirt.io/v1
kind: NetworkAddonsConfig
metadata:
  name: cluster
spec:
  imagePullPolicy: IfNotPresent
  linuxBridge: {}
  multus: {}
EOF

    ${KUBECTL} wait --for condition=Available networkaddonsconfig cluster --timeout=2m

    echo
    echo "Successfully deployed CNAO:"
    ${KUBECTL} get networkaddonsconfig cluster -o yaml
fi

if [ -n "${OPT_DEFINE_NAD}" ]; then
    echo
    echo "Define NetworkAttachmentDefinition (with a bridge CNI)..."
    echo
    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: bridge-network
  namespace: default
spec:
  config: |
    {
      "cniVersion":"0.3.1",
      "name": "br10",
      "plugins": [
          {
              "type": "cnv-bridge",
              "bridge": "br10"
          }
      ]
    }
EOF

fi

if [ -n "${OPT_BUILD_CHECKUP}" ]; then
  cd ${SCRIPT_PATH}/..
  ./automation/make.sh --build-checkup --build-checkup-image
  cd -
fi

if [ -n "${OPT_DEPLOY_CHECKUP}" ]; then
    echo
    echo "Deploy kubevirt-vm-latency..."
    echo
    kubectl create -f ${SCRIPT_PATH}/../manifests/clusterroles.yaml

    ${KIND} load docker-image "${CHECKUP_IMAGE}" --name "${CLUSTER_NAME}"
fi

if [ -n "${OPT_RUN_TEST}" ]; then
    KIAGNOSE_NAMESPACE=kiagnose
    KIAGNOSE_JOB=kubevirt-vm-latency-checkup
    VM_LATENCY_CONFIGMAP=kubevirt-vm-latency-checkup

    echo
    echo "Deploy ConfigMap with input data: "
    echo
    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${VM_LATENCY_CONFIGMAP}
  namespace: ${KIAGNOSE_NAMESPACE}
data:
  spec.image: ${CHECKUP_IMAGE}
  spec.timeout: 5m
  spec.clusterRoles: |
    kubevirt-vmis-manager
  spec.param.network_attachment_definition_namespace: "default"
  spec.param.network_attachment_definition_name: "bridge-network"
  spec.param.max_desired_latency_milliseconds: "10"
  spec.param.sample_duration_seconds: "5"
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
              value: ${KIAGNOSE_NAMESPACE}
            - name: CONFIGMAP_NAME
              value: ${VM_LATENCY_CONFIGMAP}
EOF

    # ${KUBECTL} wait --for=condition=complete --timeout=3m job.batch/${KIAGNOSE_JOB} -n ${KIAGNOSE_NAMESPACE}

    echo
    echo "Result:"
    echo
    ${KUBECTL} get configmap ${VM_LATENCY_CONFIGMAP} -n ${KIAGNOSE_NAMESPACE} -o yaml
fi

if [ -n "${OPT_DELETE_CLUSTER}" ]; then
    ${KIND} delete cluster
fi
