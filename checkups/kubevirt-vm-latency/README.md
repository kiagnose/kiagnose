# Kubevirt VM latency checkup

[Kubevirt](https://kubevirt.io/) users are often required to check network connectivity and performance of their virtualized workloads.
Such checks might be required to do immediately after a cluster with Kubevirt is deployed (as an acceptance test), when new networks are introduced or as part of a troubleshooting effort.

Kubevirt VM latency checkup performs network connectivity and latency measurement between two virtual machines over a given network.
Currently, the measurement is done using standard [ping](https://en.wikipedia.org/wiki/Ping_(networking_utility)) utility.

It can ease the maintainability effort required from a cluster administrator by removing the burden of manually testing network connectivity and performance, reduce mistakes and generally save time.

![block diagram](./docs/images/kubevirt-vm-latency-diagram.svg)

## VM network binding
The checkup binds the VMs to the given `NetworkAttachmentDefinition` using one of the
following binding methods:
- [bridge](https://kubevirt.io/user-guide/virtual_machines/interfaces_and_networks/#bridge)
- [sriov](hhttps://kubevirt.io/user-guide/virtual_machines/interfaces_and_networks/#sriov)

The default binding method is `bridge`.

> **_Note_**:
> In case [SR-IOV CNI plugin](https://github.com/k8snetworkplumbingwg/sriov-cni) is being used, `sriov` binding method is used.

## Prerequisites
- [Kiagnose](../../README.install.md)
- [Kubevirt](https://kubevirt.io//quickstart_minikube/#deploy-kubevirt)
- [Multus](https://github.com/k8snetworkplumbingwg/multus-cni#quickstart-installation-guide)
- Cluster nodes have one of the desired [CNI](https://www.cni.dev/) plugins installed.
- `NetworkAttachmentDefinition` object to exists.

## Permissions
The checkup requires some additional permissions in order to operate:
```bash
kubectl apply -f ./manifests/clusterroles.yaml
```

## Configuration
The checkup is configured by the following parameters:

| Name                                                                               | Description                                                                                                                  |
|:-----------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------|
| `spec.image`                                                                       | The checkup container image.                                                                                                 |
| `spec.timeout`                                                                     | Overall time the checkup can run.                                                                                            |
| `spec.clusterRoles`                                                                | ClusterRole name with the required permission.                                                                               |
| `network_attachment_definition_namespace`<br/>`network_attachment_definition_name` | `NetworkAttachmentDefinition` object on which <br/> the VMs are connected to and measure network latency.                    |
| `sample_duration_seconds`                                                          | Network latency measurement sample time (optional).<br/> Default is 5 seconds.                                               |
| `max_desired_latency_milliseconds`                                                 | Maximal network latency accepted, if the actual latency <br/> is higher the checkup will be considered as failed (optional). |
| `source_node`<br/>`target_node`                                                    | Two ends in the of the network latency measurement (optional).                                                               |

> **_Note_**:
> `timeout` should be greater than `sample_duration_seconds`.

### Example
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-vm-latency-checkup-config
  namespace: kiagnose
data:
  spec.image: quay.io/kiagnose/kubevirt-vm-latency:main
  spec.timeout: 5m
  spec.clusterRoles: |
    kubevirt-vm-latency-checker
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
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: kubevirt-vm-latency-checkup
  namespace: kiagnose
spec:
  backoffLimit: 0
  template:
    spec:
      serviceAccountName: kiagnose
      restartPolicy: Never
      containers:
        - name: framework
          image: quay.io/kiagnose/kiagnose:main
          env:
            - name: CONFIGMAP_NAMESPACE
              value: kiagnose
            - name: CONFIGMAP_NAME
              value: kubevirt-vm-latency-checkup-config
EOF
```

> **_Note_**:
> `CONFIGMAP_NAMESPACE` and `CONFIGMAP_NAME` environment variables are required in order to pass the checkup configuration to Kiagnose.

Wait for the checkup to finish:
```bash
kubectl wait job kubevirt-vm-latency-checkup -n kiagnose --for condition=complete --timeout 6m
```

## Results
### Example
Retrieve the checkup results:
```bash
kubectl get configmap kubevirt-vm-latency-checkup-config -n kiagnose -o yaml
```
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-vm-latency-checkup-config
  namespace: kiagnose
data:
  spec.image: quay.io/kiagnose/kubevirt-vm-latency:main
  spec.timeout: 5m
  spec.clusterRoles: |
    kubevirt-vm-latency-checker
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
kubectl delete job -n kiagnose kubevirt-vm-latency-checkup
kubectl delete config-map -n kubevirt-vm-latency-checkup-config
```
Once the checkup is finished it's safe to remove the ClusterRole:
```bash
kubectl delete -f ./manifests/clusterroles.yaml
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
