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

SCRIPT_PATH=$(dirname $(realpath -s $0))

CRI=${CRI:-podman}

KUBECTL=${KUBECTL:-$PWD/kubectl}

KIND=${KIND:-$PWD/kind}
CLUSTER_NAME=${CLUSTER_NAME:-kind}

KUBEVIRT_VERSION=${KUBEVIRT_VERSION:-v1.0.1}
KUBEVIRT_USE_EMULATION=${KUBEVIRT_USE_EMULATION:-"false"}
CNAO_VERSION=${CNAO_VERSION:-v0.89.2}

BRIDGE_NAME=${BRIDGE_NAME:-br10}

CHECKUP_IMAGE="quay.io/kiagnose/kubevirt-vm-latency:devel"

CHECKUP_JOB=kubevirt-vm-latency-checkup
VM_LATENCY_CONFIGMAP=kubevirt-vm-latency-checkup
VM_LATENCY_SERVICE_ACCOUNT_NAME=kubevirt-vm-latency-checkup-sa

TARGET_NAMESPACE="target-ns"

options=$(getopt --options "" \
    --long deploy-kubevirt,deploy-cnao,deploy-checkup,define-nad,run-tests,run-tests-py,clean-run,help\
    -- "${@}")
eval set -- "$options"
while true; do
    case "$1" in
    --deploy-kubevirt)
        OPT_DEPLOY_KUBEVIRT=1
        ;;
    --deploy-cnao)
        OPT_DEPLOY_CNAO=1
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
    --run-tests-py)
        OPT_RUN_TEST_PY=1
        ;;
    --clean-run)
        OPT_CLEAN_RUN=1
        ;;
    --help)
        set +x
        echo -n "$0 [--deploy-kubevirt] [--deploy-cnao] [--deploy-checkup] [--define-nad] [--run-tests] [--clean-run] [--run-tests-py]"
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
    OPT_DEPLOY_KUBEVIRT=1
    OPT_DEPLOY_CNAO=1
    OPT_DEPLOY_CHECKUP=1
    OPT_DEFINE_NAD=1
    OPT_RUN_TEST=1
fi

if [ -n "${OPT_DEPLOY_KUBEVIRT}" ]; then
    echo
    echo "Deploy kubevirt..."
    echo
    ${KUBECTL} apply -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml
    ${KUBECTL} apply -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml

    if [ "${KUBEVIRT_USE_EMULATION}" = "true" ]; then
      echo "Configure Kubevirt to use emulation"
      ${KUBECTL} patch kubevirt kubevirt --namespace kubevirt --type=merge --patch '{"spec":{"configuration":{"developerConfiguration":{"useEmulation":true}}}}'
    fi

    ${KUBECTL} wait --for=condition=Available kubevirt kubevirt --namespace=kubevirt --timeout=5m

    echo
    echo "Successfully deployed kubevirt:"
    ${KUBECTL} get pods -n kubevirt
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

    ${KUBECTL} wait --for condition=Available networkaddonsconfig cluster --timeout=5m

    echo
    echo "Successfully deployed CNAO:"
    ${KUBECTL} get networkaddonsconfig cluster -o yaml
fi

if [ -n "${OPT_DEFINE_NAD}" ]; then
  ${KUBECTL} create namespace ${TARGET_NAMESPACE}
    echo
    echo "Define NetworkAttachmentDefinition (with a bridge CNI)..."
    echo
    cat <<EOF | ${KUBECTL} apply -f -
---
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: bridge-network
  namespace: ${TARGET_NAMESPACE}
spec:
  config: |
    {
      "cniVersion":"0.3.1",
      "name": "${BRIDGE_NAME}",
      "plugins": [
          {
              "type": "cnv-bridge",
              "bridge": "${BRIDGE_NAME}"
          }
      ]
    }
EOF

fi

if [ -n "${OPT_DEPLOY_CHECKUP}" ]; then
    echo
    echo "Deploy kubevirt-vm-latency..."
    echo

    vmlatency_tar="/tmp/vmlatency-image.tar"
    ${CRI} save -o "${vmlatency_tar}" "${CHECKUP_IMAGE}"
    ${KIND} load image-archive --name "${CLUSTER_NAME}" "${vmlatency_tar}"
    rm "${vmlatency_tar}"
fi

if [ -n "${OPT_RUN_TEST}" ]; then
     ${KUBECTL} apply -n ${TARGET_NAMESPACE} -f ./manifests/kiagnose-configmap-access.yaml

     cat <<EOF | ${KUBECTL} apply -n ${TARGET_NAMESPACE} -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${VM_LATENCY_SERVICE_ACCOUNT_NAME}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kubevirt-vm-latency-checker
rules:
- apiGroups: ["kubevirt.io"]
  resources: ["virtualmachineinstances"]
  verbs: ["get", "create", "delete"]
- apiGroups: ["subresources.kubevirt.io"]
  resources: ["virtualmachineinstances/console"]
  verbs: ["get"]
- apiGroups: ["k8s.cni.cncf.io"]
  resources: ["network-attachment-definitions"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kubevirt-vm-latency-checker
subjects:
- kind: ServiceAccount
  name: ${VM_LATENCY_SERVICE_ACCOUNT_NAME}
roleRef:
  kind: Role
  name: kubevirt-vm-latency-checker
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kiagnose-configmap-access
subjects:
- kind: ServiceAccount
  name: ${VM_LATENCY_SERVICE_ACCOUNT_NAME}
roleRef:
  kind: Role
  name: kiagnose-configmap-access
  apiGroup: rbac.authorization.k8s.io
EOF

    echo
    echo "Deploy ConfigMap with input data: "
    echo
    cat <<EOF | ${KUBECTL} apply -n ${TARGET_NAMESPACE} -f -
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${VM_LATENCY_CONFIGMAP}
data:
  spec.timeout: 10m
  spec.param.networkAttachmentDefinitionNamespace: "${TARGET_NAMESPACE}"
  spec.param.networkAttachmentDefinitionName: "bridge-network"
  spec.param.maxDesiredLatencyMilliseconds: "500"
  spec.param.sampleDurationSeconds: "5"
EOF

    echo
    echo "Deploy and run kiagnose job: "
    echo
    cat <<EOF | ${KUBECTL} apply -n ${TARGET_NAMESPACE} -f -
---
apiVersion: batch/v1
kind: Job
metadata:
  name: ${CHECKUP_JOB}
spec:
  backoffLimit: 0
  template:
    spec:
      serviceAccountName: ${VM_LATENCY_SERVICE_ACCOUNT_NAME}
      restartPolicy: Never
      containers:
        - name: vm-latency-checkup
          image: ${CHECKUP_IMAGE}
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
            runAsNonRoot: true
            seccompProfile:
              type: "RuntimeDefault"
          env:
            - name: CONFIGMAP_NAMESPACE
              value: ${TARGET_NAMESPACE}
            - name: CONFIGMAP_NAME
              value: ${VM_LATENCY_CONFIGMAP}
            - name: POD_UID
              valueFrom:
                fieldRef:
                  fieldPath: metadata.uid
EOF

    ${KUBECTL} wait --for=condition=complete --timeout=10m job.batch/${CHECKUP_JOB} -n ${TARGET_NAMESPACE}

    echo
    echo "Result:"
    echo
    results=$(${KUBECTL} get configmap ${VM_LATENCY_CONFIGMAP} -n ${TARGET_NAMESPACE} -o yaml)
    echo "${results}"

    if echo "${results}" | grep 'status.succeeded: "false"'; then
      failureReason=$(echo ${results} | grep -Po "status.failureReason: \K'.+'")
      echo "Kubevirt VM latency checkup failed: ${failureReason}"
      exit 1
    fi
fi

if [ -n "${OPT_CLEAN_RUN}" ];then
  ${KUBECTL} delete job ${CHECKUP_JOB} -n ${TARGET_NAMESPACE} --ignore-not-found
  ${KUBECTL} delete configmap ${VM_LATENCY_CONFIGMAP} -n ${TARGET_NAMESPACE} --ignore-not-found
fi

if [ -n "${OPT_RUN_TEST_PY}" ]; then
    test -t 1 && USE_TTY="t"
    ${CRI} run -i${USE_TTY} --rm --net=host -v "${PWD}":/workspace/kiagnose:Z -v "${HOME}"/.kube:/root/.kube:ro,Z kiagnose-e2e-test pytest -v ./test/e2e/vmlatency
fi
