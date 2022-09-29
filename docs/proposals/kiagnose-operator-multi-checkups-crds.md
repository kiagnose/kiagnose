# Kiagnose as an Operator

**Authors**: [Or Mergi](https://github.com/ormergi)

# Summary
This is a proposal to transform Kiagnose to [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) enabling it to be consumed through [Operator-Lifecycle-Manager](https://sdk.operatorframework.io/).
And streamline checkups lifecycle management by transitioning to use a dedicated [Custom-Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) and [controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers) **for each checkup**.

# Motivation
This design proposal combines between the "checkups as an operator" and "Kiagnose as an operator" (with single CRD) approaches in order to close the gap between them regarding which entity should grant permissions to run a checkup and reduce the overhead of creating an operator for each checkup.<br/>
The advantage of this approach is that the cluster administrator won't be bothered by the users asking to create service-accounts for their checkups, nor worry about malicious entities hijacking the checkups service-accounts.

Following this approach may require setting guidelines for checkup authors on how to contribute new checkups so that Kiagnose operator will deploy.<br/>
It may add friction for checkup author but less than creating a dedicated operator for each checkup or worry about each checkup executing service-account.

Having Kiagnose delivered as an operator solve half of the problem where the end-user can consume it easily by automation.

As for today, in order to deploy a checkup the user need to craft Kiagnose Job/Pod manifest and a ConfigMap for the checkup parameters along with other objects (Namespace, ServiceAccount, Role and RoleBinding) which require the cluster administrator intervention.<br/>
Also, due to the current implementation it's not possible to run multiple checkups simultaneously neither.

Introducing Kiagnose operator is the next natural phase toward automation allowing it to be consumed through [OLM](https://sdk.operatorframework.io/) and have Kiagnose deployed and updated automatically.<br/>
Along the operator, transitioning to use a dedicated [CustomResource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) and [controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers) as the checkups lifecycle manager 
will streamline the way checkups are deployed, configured and monitored by users and enable running multiple checkups simultaneously.</br>
Also, it will enable cluster administrators to manage who can run checkups, and the other way around, once a user have the permission it can run checkups with no cluster administrator intervention.

## Goals
- Enable consuming Kiagnose through OLM.
- Streamline checkups lifecycle management (configure, deploy and monitor).
- Enable running multiple checkups simultaneously.

## Non Goals
- GUI is not in the scope of this document.

# Proposal

## Definition of Personas

- Cluster administrator.
- Namespace administrator.

## User Stories

As a cluster administrator:
- I want to be able to deploy Kiagnose through OLM.
- I want to manage who can run a checkup.

As a namespace administrator:
- I want to have an API to pass parameters to a checkup.
- I want to have an API to fetch a checkup's state, and realize its progress.
- I want to have an API to fetch a checkup's results.
- I want to run a checkup with no intervention from the cluster administrator.
- I want to be able to deploy multiple checkups simultaneously.

## API Extensions

### User Interface

#### Overview

1. The cluster administrator deploys Kiagnose operator though OLM (or manually).
2. Kiagnose operator deploys the checkups CRDs and controllers.
3. The user creates an instance of the desired checkup CRD.
4. The correspond checkup controller will act and checkup the desired checkup according to the spec.
5. The user monitors the checkup state, when finished fetch the results from the checkup CRD status.

#### Checkups Custom-Resource-Definitions <a id="checkup-crd"></a>
The CRD shall replace the user-supplied ConfigMap which is the user front-end for configuring a checkup.
Using a CRD will provide an API for the user to configure any type of the checkup, monitor its progress and get its results.

Each checkup CRD shall represent a unique checkup with their set of parameters and results fields as spec and status, respectively.
For example: `VmLatnecyCheckup` CRD represent Kubevirt VM latency, `EchoCheckup` represent echo checkup.

With a dedicated CRD and controller the user is no longer required to specify the checkup image in the checkup spec, the controller shall specify it when executing the checkup Job.<br/>
It eliminated the chance for a user to specify invalid image, and in a malicious entity who manage to put hands on the user credentials from executing any image.   
The checkup image can be controlled by setting an environment variable or passing it as parameter to the controller.

#### Echo checkup CRD <a id="echo-checkup-crd"></a>
```yaml
---
apiVersion: v1
kind: EchoCheckup
metadata:
  name: <arbitrary name>
  namespace: <target namespace>
spec:
  timeoutSeconds: <time to wait for a checkup to finish in seconds, when reached teardown is performed>
  message: "<echo content>"
status:
  startTime: <checkup start time>
  completionTime: <checkup completion time>
  conditions: <conditions list>
  echoed: "<actual echoed message>"
```

##### Spec
| Key                  | Description                                          | Is Mandatory | Type   | Remarks                                                                                                    |
|----------------------|------------------------------------------------------|--------------|--------|------------------------------------------------------------------------------------------------------------|
| `timeoutSeconds`     | Time to wait for a checkup to finish in seconds      | True         | string | When timeout is reached, Kiagnose shall start a teardown                                                   |
| `message`            | The echo string                                      | True         | string | Checkup specific parameter.                                                                                |

##### Status
| Key              | Description                          | Remarks                                                                                                                                                                                                            |
|------------------|--------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `startTime`      | The checkup start timestamp          |                                                                                                                                                                                                                    | 
| `completionTime` | The checkup completion timestamp     |                                                                                                                                                                                                                    |
| `conditions`     | Conditions list                      | Similar to [Pod conditions](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions), it shall describe the checkup state whether it failed or succeeded<br/> e.g: `Succeeded` condition. |
| `echoed`         | The actual message that been echoed. |                                                                                                                                                                                                                    |

##### Example
Given the following target namespace:
```yaml
---
apiVersion: v1
kind: Namespace 
metadata:
  name: echo-test
```

Executing the checkup:
```yaml
---
apiVersion: v1
kind: Checkup
metadata:
  name: echo-checkup-config
  namespace: echo-test
spec:
  timeoutSeconds: 5
  message": "Hi!"
```

When finished, the status presents the checkup results:
```yaml
---
apiVersion: v1
kind: Checkup
metadata:
  name: echo-checkup-config
  namespace: echo-test
spec:
  timeout: 5
  message": "Hi!"
status:
  startTimestamp: "2022-06-06T10:59:58Z"
  completionTimestamp: "2022-06-06T11:00:10Z"
  conditions:
  - type: Succeeded
    status: "True"
    message: "The checkup finished successfully"
  echoed: "Hi!"
```

#### Kubevirt VM latency checkup CRD <a id="vmlatency-checkup-crd"></a>
```yaml
---
apiVersion: v1
kind: KubevirtVMLatencyCheckup
metadata:
  name: <arbitrary name>
  namespace: <target namespace>
spec:
  timeoutSeconds: <timeout to wait for checkup to finish>
  sourceNode: <node where source VM shall be created>
  targetNode: <node where target VM shall be created>
  network: <network-attachment-definition object name>
  sampleDurationSeconds: <time to measure latency>
  maxDesiredLatency: <maximal accepted latency>
status:
  startTime: <checkup execution start time>
  completionTime: <checkup execution completion time>
  conditions: <conditions list>
  sourceNode: <the node name where source VM was actually created on>
  targetNode: <the node name where target VM was actually created on>
  minLatencyNanoseconds: <minimal latency>
  maxLatencyNanoseconds: <minimal latency>
  averageLatencyNanoseconds: <average latency>
```

##### Spec
| Key                             | Description                                                                | Is Mandatory | Type   | Remarks                                                                                         |
|---------------------------------|----------------------------------------------------------------------------|--------------|--------|-------------------------------------------------------------------------------------------------|
| `timeoutSeconds`                | Time to wait for a checkup to finish in seconds                            | True         | string | When timeout is reached, Kiagnose shall start a teardown                                        |
| `sourceNode`, `targetNode`      | The node name where source VM shall be created                             | False        | string | Both must be specified in order to set where VMs will be created                                |
| `network`                       | The network-attachment-definition object the VMs will be connected to      | True         | string | Accepted format: `<namespace name>/<object-name>`, e.g: "test1/blue-net"                        |
| `sampleDurationSeconds`         | Latency measurement sample time                                            | True         | uint   | Must be greater then `timeoutSeconds`                                                           |
| `maxDesiredLatencyMilliseconds` | Maximal accepted latency in milliseconds to declare the checkup as failure | True         | uint   | If the maximal latency result is lower then this parameter, the checkup considered as succeeded |

##### Status
| Key                                                                                 | Description                                         | Type   |                                                                                                                                                                                                                    | Remarks |
|-------------------------------------------------------------------------------------|-----------------------------------------------------|--------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `startTime`                                                                         | The checkup start timestamp                         | string |                                                                                                                                                                                                                    | 
| `completionTime`                                                                    | The checkup completion timestamp                    | string |                                                                                                                                                                                                                    |
| `conditions`                                                                        | Conditions list                                     | string | Similar to [Pod conditions](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions), it shall describe the checkup state whether it failed or succeeded<br/> e.g: `Succeeded` condition. |
| `minLatencyNanoseconds`<br/>`maxLatencyNanoseconds`<br/>`averageLatencyNanoseconds` | Latency measurements results                        | uint   |                                                                                                                                                                                                                    |
| `sourceNode`, `targetNode`                                                          | The nodes names where the VMs were actually created | string |                                                                                                                                                                                                                    |

##### Example
Given the following Namespace and ServiceAccount:
```yaml
---
apiVersion: v1
kind: Namespace 
metadata:
  name: vmlatency-test
```

Executing the checkup:
```yaml
---
apiVersion: v1
kind: KubevirtVMLatencyCheckup
metadata:
  name: vmlatnecy-checkup-config
  namespace: vmlatency-test
spec:
  timeoutSeconds: 300
  sourceNode: "node1"
  targetNode: "node2"
  network: "vmlatnecy-test/blue-net"
  sampleDurationSeconds: 60
  maxDesiredLatencyMilliseconds: 10
```

When finished, the status presents the checkup results:
```yaml
---
apiVersion: v1
kind: Checkup
metadata:
  name: echo-checkup-config
  namespace: echo-test
spec:
  timeoutSeconds: 300
  image: quay.io/kiagnose/kubevirt-vm-latency-checkup:main
  sourceNode: "node1"
  targetNode: "node2"
  network: "vmlatnecy-test/blue-net"
  sampleDurationSeconds: 60
  maxDesiredLatencyMilliseconds: 10
status:
  startTimestamp: "2022-06-06T11:00:00Z"
  completionTimestamp: "2022-06-06T11:05:10Z"
  conditions:
  - type: Succeeded
    status: "True"
    message: "The checkup finished successfully"
  minLatencyNanoseconds: "4ms"
  maxLatencyNanoseconds: "1ms"
  averageLatencyNanoseconds: "2.5ms"
  sourceNode: "node1"
  targetNode: "node2"
```

# Design Details

## Overview

This design take inspiration from "checkups as an operator" and "Kiagnose as an operator (with single CRD)" approaches and tries to address each cons.
The main idea is to have unique CRD and controller for each checkup.

Each checkup controller shall be deployed at the same namespace as Kiagnose operator (Kiagnose namespace).<br/>
When a checkup CRD is created, the correspond controller will execute the checkup by creating a batch Job at Kiagnose namespace, the Job will create the checkup related object at the target namespace.
For example:<br/>
The user create Kubevirt VM latency checkup CRD, the Kubevirt VM latency checkup controller shall reconcile the CRD and create the checkup Job according to the spec at Kiagnose namespace, the Job eventually create the VMs at the target and connect them to the specified NetworkAttachmentDefinition at the target namespace.<br/>  

By executing the checkup Job at Kiagnose namespace it allows reusing the checkup service-account, unlike the approach where the checkup Job is created at the target namespace and requires to be dedicated service-account at each target namespace.

The advantage of this approach is that the cluster/namespace administrator is no longer required to create service-accounts for checkups.
No need to worry about malicious entities that could hijack these service-accounts, for example:<br/>
When a cluster-admin create service-account "foo" at namespace "test", an entity with permission to create pods on namespace "test" can privilege escalate by using the existing "foo" service-account. <br/>

Also, since each checkup has a dedicated CRD and controller the user is no longer required to specify the checkup image. <br/>
Having the checkup controller to specify the image when executing the Job eliminates the chance that an invalid or malicious image will be used.  

Following this proposal approach may require defining some guidelines for checkup authors on how to contribute new checkups so that Kiagnose operator will deploy.
It may add friction for checkup author, but it reduces the overhead of creating a dedicated operator for a checkup or manage checkup service-accounts.

## Main Components
![](images/kiagnose-operator-components-diagram.svg)

### Kiagnose Operator
Responsible for configuring, deploying and updating all Kiagnose components, including CRDs, controllers, checkup service-accounts and their ClusterRoles and binding.

> **_Note_**: Kiagnose operator should be able to create the checkups CRDs and the checkup controllers ServiceAccounts and Deployments.

> **_Note_**: Kiagnose operator can be extended to have its own CRD if advance configurations are necessary, for example: control which checkups can be deployed in the cluster.

### Checkup CRDs
The user executes a checkup by creating its CRD object, for example: [echo checkup CRD](#echo-checkup-crd) or [Kubevirt VM latency checkup CRD](#vmlatency-checkup-crd).

### Checkups ServiceAccounts ClusterRoles and ClusterRoleBindings
Each checkup CRD shall have a service-account with permissions for the checkup to operate.<br/>
The same service-account shall be used for a type of checkup, for example: "vmlatnecy-checkup" shall be used for Kubevirt VM latency checkups.

> **_Note_**: 
> The checkups `ClusterRole` and `ClusterRoleBinding` are not specified in the diagram.
 
### Checkups controllers
Each checkup CRD shall be handled by a dedicated controller, each checkup controller shall listen to its CRD events across all namespaces and run/teardown the checkup.<br/>
In the diagram we see an example of the Kubevirt VM latency CRD and controller.

> **_Note_**:
> The results `ConfigMap` remains as intermediate layer between the executed checkup and Kiagnose operator to prevent coupling.<br/>
> One could suggest that an executed checkup shall update the `Checkup` CR status or add an annotation instead, but it's not a good practice as its prune to bugs, data loss and may introduce a race with the checkup-controller.<br/>

> **_Note_**: 
> Each checkup controller shall have dedicated service-account with permissions to create and watch the checkup Job, creating the results ConfigMap.

> **_Note_**: 
> Each checkup controller shall maintain the current functionality of each checkup.

### Checkups Jobs
The checkup controllers executes checkups by creating a correspond batch Job for a CRD object.<br/>
It specifies the dedicated checkup service-account which located at Kiagnose operator namespace.

# Main Process
## Kiagnose Operator Deployment
The cluster-administrator deploys Kiagnose operator through OLM, or manually, either way the following objects shall be created:
1. Kiagnose operator `Namespace`.
2. Kiagnose **operator** `ServiceAccount`,`ClusterRole` and `ClusterRoleBinding` with the required permissions for Kiagnose to operate.
3. Kiagnose operator `Deployment`.

Manifests example [Appendix 1 - Kiagnose Operator Manifest Example](#kiagnose-operator-manifest)

## Kiagnose Checkups Controllers Deployment
Once Kiagnose operator is ready, it shall deploy for each checkup the following objects:
1. The checkup CRD.
2. `ServiceAccount`, `ClusterRole` and `ClusterRoleBinding` for the **checkup-controller**.
3. The checkup controller `Deployment`.
4. `ServiceAccount`, `ClusterRole` and `ClusterRoleBinding` for the **checkup** Jobs.

Manifest example [Appendix 2 - Kiagnose Checkup Controller Manifest Example](#kiagnose-checkup-controller-manifest)

## Checkup Deployment
The user shall create a CRD object of the desired checkup in order to run it.

## Checkup Execution
Once a checkup CRD object is created, to correspond checkup controller (e.g: vmlatnecy checkup controller) shall act as follows:
- Create results ConfigMap at Kiagnose namespace.
- Create a batch Job at Kiagnose namespace with the desired checkup image and parameters (according to spec).
- Watch and listen for the deployed Job events:
    - If Job finished successfully, fetch the results from the results ConfigMap and add them to status along with `Succeeded` condition.
    - Else, add `Failed` condition with the error message from the checkup.
      When checkup delete event occurs.

> **_Note_**: The checkup related objects shall remain until its `Checkup` CRD object is deleted.

## Checkup Removal
In order to remove a checkup the user shall delete the checkup CR object.
Once a checkup CR object is deleted, its correspond controller is triggered and performs a checkup-teardown, it shall delete the objects it created.

# Implementation Phases
## Phase 1 - Introduce the checkups controllers and CRDs
### Required Functionality
- Create checkups using a CRD instead of a ConfigMap.
- Monitor a checkup state.
- Run few checkups simultaneously.
- Enable the cluster administrator to manage who can run checkups.
- Enable users to create checkups with no cluster administrator intervention.
- Eliminate the risk where a malicious entity can hijack checkups service-accounts.
### Deliverables
- Kiagnose checkups controllers container images.
- Kiagnose checkups controllers deployment manifests example.

## Phase 2 - Introduce Kiagnose Operator
### Required Functionality
- Deploy Kiagnose with an operator.
### Deliverables
- Kiagnose operator container image.
- Kiagnose operator deployment example.

## Phase 3 - Integrate with OLM
### Required Functionality
- Deploy Kiagnose operator with OLM.
### Deliverables
- Kiagnose operator OLM content including:
  - [operator manifest (CSV)](https://olm.operatorframework.io/docs/tasks/creating-operator-manifests/)
  - [bundle](https://olm.operatorframework.io/docs/tasks/creating-operator-bundle/)
  - [catalog](https://olm.operatorframework.io/docs/tasks/creating-a-catalog/)
- Document how to deploy Kiagnose with OLM
 
## Phase 4 - Register Kiagnose operator to Operator Hub
### Required Functionality
- Be able to install Kiagnose operator from OperatorHub.io.
### Deliverables

# Appendix 1 - Kiagnose Operator Manifest Example <a id="kiagnose-operator-manifest"><a/>
```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: kiagnose
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kiagnose-operator
  namespace: kiagnose
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kiagnose-operator
rules:
  - apiGroups: [""]
    resources: ["serviceaccounts"]
    verbs:
    - get
    - list
    - create
    - delete
  - apiGroups: [ "rbac.authorization.k8s.io" ]
    resources:
    - roles
    - rolebindings
    verbs:
    - get
    - list
    - create
    - delete
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs:
    - get
    - list
    - watch
    - create
    - delete
    - patch
  - apiGroups: [ "apps"]
    resources: ["deployments"]
    verbs:
    - get
    - list
    - update
    - create
    - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kiagnose-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kiagnose-operator
subjects:
  - kind: ServiceAccount
    name: kiagnose-operator
    namespace: kiagnose
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiagnose-operator
  namespace: kiagnose
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kiagnose-operator
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: kiagnose-operator
    spec:
      serviceAccountName: kiagnose-operator
      containers:
      - name: kiagnose-operator
        command: ["kiagnose"]
        image: quay.io/kiagnose/kiagnose-operator:v0.0.0
```

# Appendix 2 - Kiagnose Checkup Controllers Manifest Example <a id="kiagnose-checkup-controller-manifest"><a/>
The following manifests represent the objects Kiagnose operator will create.

## Echo Checkup Controller
```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: echo-checkup-controller
  namespace: kiagnose
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: echo-checkup-controller
rules:
- apiGroups: [ "" ]
  resources: [ "configmaps" ]
  verbs:
  - get
  - list
  - create
  - update
  - delete
- apiGroups: [ "batch" ]
  resources: [ "jobs" ]
  verbs:
  - get
  - list
  - create
  - delete
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: echo-checkup-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: echo-checkup-controller
subjects:
- kind: ServiceAccount
  name: echo-checkup-controller
  namespace: kiagnose
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-checkup-controller
  namespace: kiagnose
spec:
  replicas: 1
  selector:
    matchLabels:
      name: echo-checkup-controller
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: echo-checkup-controller
    spec:
      serviceAccountName: echo-checkup-controller
      containers:
      - name: echo-checkup-controller
        command: ["echo-checkup-controller"]
        image: quay.io/kiagnose/echo-checkup-controller:v0.0.0
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: echo-checkup
  namespace: kiagnose
```

## Kubevirt VM Latency Checkup Controller
```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vmlatnecy-checkup-controller
  namespace: kiagnose
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: vmlatnecy-checkup-controller
rules:
- apiGroups: [ "" ]
  resources: [ "configmaps" ]
  verbs:
  - get
  - list
  - create
  - update
  - delete
- apiGroups: [ "batch" ]
  resources: [ "jobs" ]
  verbs:
  - get
  - list
  - create
  - delete
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: vmlatnecy-checkup-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: vmlatnecy-checkup-controller
subjects:
- kind: ServiceAccount
  name: vmlatnecy-checkup-controller
  namespace: kiagnose
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vmlatnecy-checkup-controller
  namespace: kiagnose
spec:
  replicas: 1
  selector:
    matchLabels:
      name: vmlatnecy-checkup-controller
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: vmlatnecy-checkup-controller
    spec:
      serviceAccountName: vmlatnecy-checkup-controller
      containers:
      - name: vmlatnecy-checkup-controller
        command: ["vmlatnecy-checkup-controller"]
        image: quay.io/kiagnose/vmlatnecy-checkup-controller:v0.0.0
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vmlatnecy-checkup
  namespace: kiagnose
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: vmlatnecy-checkup
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
kind: ClusterRoleBinding
metadata:
  name: vmlatnecy-checkup
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: vmlatnecy-checkup
subjects:
- kind: ServiceAccount
  name: vmlatnecy-checkup
  namespace: kiagnose
```

# Appendix 3 - Kiagnose CRD
Kiagnose operator can be extended to have its own CRD to enable advance configurations, such as controlling which checkups may be deployed in the cluster similar to [CNAO](https://github.com/kubevirt/cluster-network-addons-operator) [configurations](https://github.com/kubevirt/cluster-network-addons-operator#configuration).

## Example
```yaml
---
apiVersion: v1
kind: Kiagnose
metadata:
  name: Kiagnose
  namespace: kiagnose
spec:
  echo{}
  kubevirtVMLatency{}
```

## Spec
The user shall specify the allowed checkups that could run in the cluster, for example:
- `echo` - will allow once to run the echo checkup.
- `kubevirtVMLatency` - will allow once to run the Kubevirt VM latency checkup.

## Status
Can be extended to reflect the status of each checkup controller.
