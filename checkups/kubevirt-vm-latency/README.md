# Kubevirt VM latency checkup

[Kubevirt](https://kubevirt.io/) users are often required to check network connectivity and performance of their virtualized workloads.
Such checks might be required to do immediately after a cluster with Kubevirt is deployed (as an acceptance test), when new networks are introduced or as part of a troubleshooting effort.

Kubevirt VM latency checkup performs network connectivity and latency measurement between two virtual machines over a given network.
Currently, the measurement is done using standard [ping](https://en.wikipedia.org/wiki/Ping_(networking_utility)) utility.

It can ease the maintainability effort required from a cluster administrator by removing the burden of manually testing network connectivity and performance, reduce mistakes and generally save time.

## VM network binding
The checkup binds the VMs to the given `NetworkAttachmentDefinition` using one of the
following binding methods:
- [bridge](https://kubevirt.io/user-guide/virtual_machines/interfaces_and_networks/#bridge)
- [sriov](hhttps://kubevirt.io/user-guide/virtual_machines/interfaces_and_networks/#sriov)

The default binding method is `bridge`.

> **_Note_**:
> In case [SR-IOV CNI plugin](https://github.com/k8snetworkplumbingwg/sriov-cni) is being used, `sriov` binding method is used.

## Prerequisites
- [Kubevirt](https://kubevirt.io//quickstart_minikube/#deploy-kubevirt)
- [Multus](https://github.com/k8snetworkplumbingwg/multus-cni#quickstart-installation-guide)
- Cluster nodes have one of the desired [CNI](https://www.cni.dev/) plugins installed.
- `NetworkAttachmentDefinition` object to exists.

## Permissions
The checkup requires some additional permissions in order to operate:
```bash
cat <<EOF | kubectl apply -n <target-namespace> -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vm-latency-checkup-sa
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
  name: vm-latency-checkup-sa
roleRef:
  kind: Role
  name: kubevirt-vm-latency-checker
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kiagnose-configmap-access
rules:
  - apiGroups: [ "" ]
    resources: [ "configmaps" ]
    verbs:
      - get
      - update
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kiagnose-configmap-access
subjects:
- kind: ServiceAccount
  name: vm-latency-checkup-sa
roleRef:
  kind: Role
  name: kiagnose-configmap-access
  apiGroup: rbac.authorization.k8s.io
EOF
```

## Configuration
The checkup is configured by the following parameters:

| Name                                                                               | Description                                                                                                                  |
|:-----------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------|
| `timeout`                                                                          | Overall time the checkup can run.                                                                                            |
| `network_attachment_definition_namespace`<br/>`network_attachment_definition_name` | `NetworkAttachmentDefinition` object on which <br/> the VMs are connected to and measure network latency.                    |
| `sample_duration_seconds`                                                          | Network latency measurement sample time (optional).<br/> Default is 5 seconds.                                               |
| `max_desired_latency_milliseconds`                                                 | Maximal network latency accepted, if the actual latency <br/> is higher the checkup will be considered as failed (optional). |
| `source_node`<br/>`target_node`                                                    | Two ends of the network latency measurement (optional).<br/> When used, specifying both is mandatory.                        |

> **_Note_**:
> `timeout` should be greater than `sample_duration_seconds`.

> **_Note_**:
> By default the checkup source and target VMs will be created in a way they won't end up on the same cluster node.</br>
> Specifying both `source_node` and `target_node` will override this behaviour and each VM will be created on the desired node.

### Example
```bash
cat <<EOF | kubectl apply -n <target-namespace> -f -
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-vm-latency-checkup-config
data:
  spec.timeout: 5m
  spec.param.network_attachment_definition_namespace: "default"
  spec.param.network_attachment_definition_name: "blue-network"
  spec.param.max_desired_latency_milliseconds: "10"
  spec.param.sample_duration_seconds: "5"
  spec.param.source_node: "worker1"
  spec.param.target_node: "worker2"
EOF
```

## How to run
The checkup can be executed with a Batch Job: 
```bash
cat <<EOF | kubectl apply -n <target-namespace> -f -
---
apiVersion: batch/v1
kind: Job
metadata:
  name: kubevirt-vm-latency-checkup
spec:
  backoffLimit: 0
  template:
    spec:
      serviceAccountName: vm-latency-checkup-sa
      restartPolicy: Never
      containers:
        - name: vm-latency-checkup
          image: quay.io/kiagnose/kubevirt-vm-latency:main
          securityContext:
            runAsUser: 1000
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
            runAsNonRoot: true
            seccompProfile:
              type: "RuntimeDefault"
          env:
            - name: CONFIGMAP_NAMESPACE
              value: <target-namespace>
            - name: CONFIGMAP_NAME
              value: kubevirt-vm-latency-checkup-config
EOF
```

> **_Note_**:
> `CONFIGMAP_NAMESPACE` and `CONFIGMAP_NAME` environment variables are required to allow the checkup application
> access to the input & output API (in the form of a ConfigMap).

Wait for the checkup to finish:
```bash
kubectl wait job kubevirt-vm-latency-checkup -n <target-namespace> --for condition=complete --timeout 6m
```

## Results
### Example
Retrieve the checkup results:
```bash
kubectl get configmap kubevirt-vm-latency-checkup-config -n <target-namespace> -o yaml
```
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-vm-latency-checkup-config
  namespace: <target-namespace>
data:
  spec.timeout: 5m
  spec.param.network_attachment_definition_namespace: "default"
  spec.param.network_attachment_definition_name: "blue-network"
  spec.param.max_desired_latency_milliseconds: "10"
  spec.param.sample_duration_seconds: "5"
  spec.param.source_node: "worker1"
  spec.param.target_node: "worker2"
  status.succeeded: "true"
  status.failureReason: ""
  status.completionTimestamp: "2022-01-01T09:00:00Z"
  status.startTimestamp: "2022-01-01T09:00:07Z"
  status.result.avgLatencyNanoSec: "177000"
  status.result.maxLatencyNanoSec: "244000"
  status.result.measurementDurationSec: "5"
  status.result.minLatencyNanoSec: "135000"
  status.result.sourceNode: "worker1"
  status.result.targetNode: "worker2"
```

When the checkup is finished, the checkup ConfigMap is updated with the following results:

| Result Filed                           | Description                                          |
|:---------------------------------------|:-----------------------------------------------------|
| `status.succeeded`                     | Indicated whether the checkup finished successfully. |
| `status.failureReasone`                | Execution failure reason.                            |
| `status.startTimestamp`                | The time when the checkup execution started.         |
| `status.completionTimestamp`           | The time when the checkup execution completed.       |
| `status.result.minLatencyNanoSec`      | Minimal latency value [nanoseconds].                 |
| `status.result.avgLatencyNanoSec`      | Average latency value [nanoseconds].                 |
| `status.result.maxLatencyNanoSec`      | Maximal latency value [nanoseconds].                 |
| `status.result.measurementDurationSec` | Actual latency measurement time [seconds].           |
| `status.result.sourceNode`             | Actual source node                                   |
| `status.result.targetNode`             | Actual target node                                   |

In case of successful execution the following results are expected:
```yaml
status.succeeded: "true"
status.failureReason: ""
```

In case an environment variable is missing (e.g: `MAX_DESIRED_LATENCY_MILLISECONDS`:
```yaml
status.succeeded: "false"
status.failureReason: "MAX_DESIRED_LATENCY_MILLISECONDS environment variable is missing"
```

In case of a connectivity issues between the VMs:
```yaml
status.succeeded: "false"
status.failureReason: "run: failed to run check: failed due to connectivity issue: 5 packets transmitted, 0 packets received"
```

## Clean up
```bash
kubectl delete job -n <target-namespace> kubevirt-vm-latency-checkup
kubectl delete config-map -n <target-namespace> kubevirt-vm-latency-checkup-config
```

## Build Instructions

### Prerequisites
- [podman](https://podman.io/), [docker](https://docker.io/) or other OCI compliant container runtime capable of building images.

### Steps
```bash
# build binary
./automation/make.sh

# build image 
./automation/make.sh --build-checkup-image

# Using Docker to build the image:
CRI=docker ./automation/make.sh --build-checkup-image
```
