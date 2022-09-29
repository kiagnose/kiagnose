# Kiagnose as an Operator 

**Authors**: [Or Mergi](https://github.com/ormergi)

# Summary
This is a proposal to transform Kiagnose to [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) enabling it to be consumed through [Operator-Lifecycle-Manager](https://sdk.operatorframework.io/).
And streamline checkups lifecycle management by transitioning to use a dedicated [Custom-Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) and [controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers).

# Motivation
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

As a namespace administrator:
- I want to have an API to pass parameters to a checkup.
- I want to have an API to fetch a checkup's state, and realize its progress.
- I want to have an API to fetch a checkup's results.
- I want to be able to deploy multiple checkups simultaneously.

## API Extensions

### User Interface

#### Overview

1. The cluster administrator deploys Kiagnose operator though OLM (or manually).
2. Kiagnose operator deploys the checkup CRD and controller.
3. The user creates an instance of the checkup CRD with a spec that represent the desired checkup.
4. The user monitors the checkup state, when finished fetch the results.

> **_Note_**:
> It is still necessary to specify a service-account that have the required permissions for the checkup to operate (e.g: create VMs) in the checkup configuration.

#### Checkup Custom-Resource-Definition <a id="checkup-crd"></a>
The CRD shall replace the user-supplied ConfigMap which is the user front-end for configuring a checkup.
Using a CRD will provide an API for the user to configure any type of the checkup, monitor its progress and get its results.

The checkup parameters shall be populated under `spec.params` field as a key-value map JSON object.<br/>
Similarly, the checkup results shall be populated under `status.results` filed as key-value map JSON object.<br/>
Having the parameters and result field as JSON objects enables representing any kind of checkup with single CRD (as it's not bounded to certain set of parameters/result fields). 

Defining a CRD for each checkup, but it comes with a cost as Kiagnose must be aware of each CRD, watch them all over the cluster and act accordingly.

> **_Note_**:
> JSON object can be replaced with similar YAML objects.

#### Example
```yaml
---
apiVersion: v1
kind: Checkup
metadata:
  name: <arbitrary name>
  namespace: <target namespace>
spec:
  timeoutSeconds: <timeout to wait for checkup to finish>
  image: <checkup image name>
  serviceAccountName: <service account name>
  params: | 
  {
    "<parameter name>": "<parameter value>",
    "<another parameter name>": "<another parameter value>",
    ...
  }
status:
  startTime: <checkup execution start time>
  completionTime: <checkup execution completion time>
  conditions: <conditions list>
  results: |
  {
    "<results filed name>": "<results field name>",
    "<another results filed name>": "<another results filed name>",
    ...
  }
```

##### Spec
| Key                  | Description                                              | Is Mandatory | Type   | Remarks                                                                                                    |
|----------------------|----------------------------------------------------------|--------------|--------|------------------------------------------------------------------------------------------------------------|
| `timeoutSeconds`     | Time to wait for a checkup to finish in seconds          | True         | string | When timeout is reached, Kiagnose shall start a teardown                                                   |
| `image`              | The checkup container image                              | True         | string |                                                                                                            |
| `serviceAccountName` | Name of the ServiceAccount that will run the checkup     | True         | string | Should already exist before creating the `Checkup` CR, at the target namespace with necessary permissions. |
| `params`             | JSON object with the checkup parameters as key-value map | False        | string |                                                                                                            |

##### Status
| Key              | Description                                        | Remarks                                                                                                                                                                                                            |
|------------------|----------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `startTime`      | The checkup start timestamp                        |                                                                                                                                                                                                                    | 
| `completionTime` | The checkup completion timestamp                   |                                                                                                                                                                                                                    |
| `conditions`     | Conditions list                                    | Similar to [Pod conditions](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions), it shall describe the checkup state whether it failed or succeeded<br/> e.g: `Succeeded` condition. |
| `results`        | JSON object with the checkup results key-value map |                                                                                                                                                                                                                    |

> **_Note_**:
> The status can be extended to provide more information regarding the checkup state (e.g: `phase`).

##### Example
Given the following Namespace and ServiceAccount:
```yaml
---
apiVersion: v1
kind: Namespace 
metadata:
  name: echo-test
---
apiVersion: batch/v1
kind: ServiceAccount
metadata:
  name: echo
  namespace: echo-test
```

Executing the echo checkup: 
```yaml
---
apiVersion: v1
kind: Checkup
metadata:
  name: echo-checkup-config
  namespace: echo-test
spec:
  timeout: 1
  image: quay.io/kiagnose/echo-checkup:main
  serviceAccountName: echo
  params: |
  {
    "message": "Hi!"
  }
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
  timeout: 1
  image: quay.io/kiagnose/echo-checkup:main
  serviceAccountName: echo
  params: |
  {
    "message": "Hi!"
  }
status:
  startTimestamp: "2022-06-06T10:59:58Z"
  completionTimestamp: "2022-06-06T11:00:10Z"
  conditions:
  - type: Succeeded
    status: "True"
    message: "The checkup finished successfully"
  results: " 
      {
        \"echo\": \"Hi!\"
      }"
```

# Design Details
## Main Components
![](images/kiagnose-operator-components-diagram.svg)

### Kiagnose Operator
Responsible for configuring, deploying and updating all Kiagnose components including the checkup CRD and controller.

> **_Note_**: Kiagnose operator should be able to create the checkup CRD and the checkup controller ServiceAccount and Deployment.

> **_Note_**: Kiagnose operator can be extended to have its own CRD if advance configurations are necessary.

### [Checkup CRD](#checkup-crd)
The user executes a checkup by creating `Checkup` CR object at the namespace. 

### Checkup Controller
Responsible for the checkups lifecycle, it watches and listen to `Checkup` CRs events across all namespaces, and run/teardown checkups according their spec.

> **_Note_**: 
> The checkup controller shall maintain the current functionality.

> **_Note_**:
> The results ConfigMap remains as intermediate layer between the executed checkup and Kiagnose operator to prevent coupling.<br/>
> One could suggest that an executed checkup shall update the `Checkup` CR status or add an annotation instead, but it's not a good practice as it prune to bugs, data loss and may introduce a race with the checkup-controller.

> **_Note_**: 
> The checkup controller should be able to create and watch the checkup Job, creating the results ConfigMap and grant its service-account with permissions to update it.

# Main Process
## Kiagnose Operator Deployment
The cluster-administrator deploys Kiagnose operator through OLM, or manually, either way the following objects shall be created:
1. Kiagnose operator namespace.
2. Kiagnose operator ServiceAccount.
3. ClusterRole and ClusterRoleBinding with the required permissions for Kiagnose to operate.
4. Kiagnose operator Deployment.

Manifests example [Appendix 1 - Kiagnose Operator Manifest Example](#kiagnose-operator-manifest)

## Kiagnose Checkup Controller Deployment
Once Kiagnose operator is ready, it shall deploy the checkup CRD and controller as follows and create:
1. A service-account for the checkup-controller.
3. Role and RoleBinding with the required permissions for the checkup-controller to operate.
4. The checkup CRD.
5. The checkup-controller deployment.

Manifest example [Appendix 2 - Kiagnose Checkup Controller Manifest Example](#kiagnose-checkup-controller-manifest)

## Checkup Deployment
The user shall create a `Checkup` CRD object in order to run a checkup.

## Checkup Execution
Once a `Checkup` CR object is created, the checkup-controller will act as follows:
- Create results ConfigMap at the target namespace.
- Create Role and RoleBinding and grant the given service-account with permissions to update the results ConfigMap.
- Deploy the batch Job with the desired checkup image and parameters.
- Watch and listen for the deployed Job events:
    - If Job finished successfully, fetch the results from the results ConfigMap and add them to status along with `Succeeded` condition.
    - Else, add `Failed` condition with the error message from the checkup.
      When checkup delete event occurs.

> **_Note_**: The checkup related objects shall remain until its `Checkup` CRD object is deleted.

## Checkup Removal
In order to remove a checkup the user shall delete the `Checkup` CR object.
Once a `Checkup` CR object is deleted, the checkup-controller is triggered and performs a checkup-teardown, it shall delete the objects it created.

# Implementation Phases
## Phase 1 - Introduce the checkup controller and CRD
### Required Functionality
- Create checkups using a CRD instead of a ConfigMap.
- Monitor a checkup state.
- Run few checkups simultaneously.
### Deliverables
- Kiagnose checkup controller container image.
- Kiagnose checkup controller deployment manifests example.

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

# Appendix 2 - Kiagnose Checkup Controller Manifest Example <a id="kiagnose-checkup-controller-manifest"><a/>
The following manifests represent the objects Kiagnose operator will create.
```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kiagnose-checkup-controller
  namespace: kiagnose
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kiagnose-checkup-controller
rules:
- apiGroups: [ "" ]
  resources: [ "configmaps" ]
  verbs:
  - get
  - list
  - create
  - update
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
  name: kiagnose-checkup-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kiagnose-checkup-controller
subjects:
- kind: ServiceAccount
  name: kiagnose-operator
  namespace: kiagnose
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kiagnose-checkup-controller
  namespace: kiagnose
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kiagnose-checkup-controller
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: kiagnose-checkup-controller
    spec:
      serviceAccountName: kiagnose-checkup-controller
      containers:
      - name: kiagnose-checkup-controller
        command: ["checkup-controller"]
        image: quay.io/kiagnose/kiagnose-checkup-controller:v0.0.0
```
