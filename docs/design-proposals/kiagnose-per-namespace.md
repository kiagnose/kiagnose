Kiagnose Per Namespace
=

# Summary
Reduce permissions for Kiagnose checkup deployment and execution to a namespace-admin (instead of
a cluster-admin).

# Motivation
Currently, Kiagnose requires a cluster-admin to deploy its core component and execute individual checkups.

The requirement for a cluster-admin (or a user with pod execution on the kiagnose control namespace)
to execute a checkup is too limiting and creates a strong dependency on the cluster-admin for each
checkup.

In addition, the two steps execution that exists today, where a kiagnose application runs a checkup
application becomes redundant. Once all components have the same permissions (e.g. namespace-admin),
there is no strong benefit to split them as independent applications.

## Goal
- Deployment and execution of checkups should require only namespace-admin permissions and below
  (i.e. eliminate the need for a cluster-admin intervention).
- Simplify and unify API definition for checkups and their clients.
- Minimize the required knowledge to compose a checkup.

## Non-Goal
- Handle Kiagnose checkup life-cycle (operation).

# Proposal

## Definition Of Personas
- Kiagnose checkup users.
- Kiagnose checkup authors.

## User Stories
As a checkup user (that is **not** a cluster-admin):
- I want to run an individual checkup with optional parameters on a specific target namespace.
- I want to observe the checkup progress and its final result.
- I want to collect all relevant logs and/or artifacts after the execution.

As a checkup author:
- I want to have an API specification that defines what I must conform to.
- I want to have references to existing checkups.
- I want to have tutorials for writing a checkup.
- I want to use tooling and libraries that assist in writing a checkup.
- I want to certify my checkup once (I think) it is ready.
- I want to make my checkup available for deployment on K8s based clusters.

## Solution Overview
The described proposal suggests to deploy and execute checkups in the same namespace, where the
checkups are aimed to run in (AKA *target* namespace).

Instead of deploying kiagnose framework in a dedicated control plane namespace and starting
a checkup execution by defining a job in that same namespace, the deployment and execution
will move to the target namespace, which in turn will drop the need for a cluster-admin intervention.

The solution includes the simplification of the execution flow and the removal of intermediate API
definitions which existed so far. It is suggested to convert the existing kiagnose application
(i.e. the framework) to a library that checkup authors may use to compose their application.
By doing so, instead of two applications with two API definitions, a single application, which is
the checkup, will be required (checkup authors may decide to expand to many applications per their need).

As a subsequence, the results-configmap which has been used by the checkup to output the results, is removed.
Checkups will directly write the results to the user-facing configmap.

## API Changes

The existing solution involved two API layers:
- User facing API: Exposed to checkup users that execute them through the kiagnose framework.
- Checkup API: The API checkups interact with the kiagnose framework.

> **_NOTE:_** The user facing API format is left as a configmap.

The checkup API is dropped and checkups are expected to directly interact with the user facing API.
As with the existing solution, the API includes basic common fields that all checkups **must** implement
and checkup-specific fields which extend it.

In order to further simplify the API and take advantage of the new execution flow (single application),
some base fields may be dropped now in the configmap:
- spec.image
- spec.serviceAccountName

### Input Fields
The user can configure the following under the `data` field:

| Property                | Description                                                                                                                 | Mandatory | Remarks                               |
|-------------------------|-----------------------------------------------------------------------------------------------------------------------------|-----------|---------------------------------------|
| spec.timeout            | After how much time should Kiagnose stop the running checkup                                                                | Yes       | 5m, 1h etc                            |
| spec.param.*            | Arbitrary strings that will be passed to the checkup as input parameters                                                    | No        | [0..N]                                |

### Output Fields

| Property                   | Description                                         | Mandatory | Remarks |
|----------------------------|-----------------------------------------------------|-----------|---------|
| status.succeeded           | Has the checkup succeeded                           | Yes       |         |
| status.failureReason       | Failure reason in case of a failure                 | Yes       |         |
| status.startTimestamp      | Checkup start timestamp                             | Yes       |         |
| status.completionTimestamp | Checkup completion timestamp                        | Yes       |         |
| status.result.*            | Arbitrary strings that were reported by the checkup | No        | [0..N]  |

## Operational Flow
The following main steps are expected to occur in order to use a checkup:
- Deploy base kiagnose checkup permissions (in the target namespace).
- Deploy checkup specific permissions (in the target namespace).
- Create a checkup instance configmap (with relevant input data).
- Create a Kubernetes Job to start the checkup execution.
- Examine the checkup progress and results by viewing the initial configmap.
- Delete the configmap once all data is collected for the specific checkup run.
- Optionally, delete the base kiagnose and specific checkup permissions.

> **_NOTE:_** Any permissions (e.g. SA, Role, RoleBinding) are under the namespace-admin
> responsibility to examine, approve and deploy.

### Checkup Execution Example

- ConfigMap (input):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-checkup-config
data:
  spec.timeout: 5m
  spec.param.param_key_1: "value 1"
  spec.param.param_key_2: "value 2"
```

- Jod (execution):
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: example-checkup
spec:
  backoffLimit: 0
  activeDeadlineSeconds: 500
  template:
    spec:
      serviceAccount: example-checkup-sa
      restartPolicy: Never
      containers:
        - name: checkup
          image: quay.io/kiagnose/example-checkup:main
          imagePullPolicy: Always
          env:
            - name: CONFIGMAP_NAMESPACE
              value: <target namespace>
            - name: CONFIGMAP_NAME
              value: example-checkup-config
```

- ConfigMap (output):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-checkup-config
data:
  spec.timeout: 5m
  spec.param.param_key_1: "value 1"
  spec.param.param_key_2: "value 2"

  status.succeeded: "true"
  status.failureReason: ''
  status.startTimestamp: "2022-05-25T11:53:49Z"
  status.completionTimestamp: "2022-05-25T11:54:46Z"
  status.result.key1: "result 1"
  status.result.key2: "result 2"
```

> **_NOTE:_** It is recommended to split the role of the base Kiagnose and the checkup extensions.
> E.G. the base Kiagnose role includes rules to access configmaps for read and write.

> **_NOTE:_** It is recommended to specify a timeout in the Job spec (`activeDeadlineSeconds`),
> assuring the Job finishes eventually.
> When specified, it should be higher than the timeout inputted in the ConfiMap (`spec.timeout`),
> allowing the checkup application to process the teardown once the timeout occurred.
