# Kiagnose

[Kiagnose](https://github.com/kiagnose/kiagnose) is a [Kubernetes](https://kubernetes.io) diagnostic framework which enables validation of cluster functionality.

A checkup is a containerized application, which checks that a certain cluster functionality is working.
A checkup can be provided by a third party vendor, and should adhere to the Kiagnose checkup API.

The checkup runs in an existing Namespace.
The checkup reads the user-supplied configuration and reports back the results upon termination.

# Usage
## Prerequisites
In order to use Kiagnose you should have:
1. A running Kubernetes cluster.
2. Namespace-Admin privileges on this cluster.
3. [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) utility.

## Checkup Installation
A vendor creating a checkup should provide:
1. Checkup's documentation (what it does, how to configure it, what are its output, etc). 
2. A checkup image.

> **_WARNING:_**
> 1. One should make sure a trustful checkup is used.
> 2. It is up to the namespace administrator to **ALWAYS** check the checkup's required **permissions**
**BEFORE** attempting to run the checkup.
Kiagnose checkups are expecting the namespace admin to supply the ServiceAccount, Roles and RoleBindings objects.

### Installation Steps
1. Make sure the checkup's image is accessible to your cluster.
2. Assure the required checkup permissions are in place.
3. Grant Kiagnose Job with access permissions to the checkup config-map:
    ```yaml
    ---
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: example-sa
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      name: kiagnose-configmap-access
    rules:
    - apiGroups: [ "" ]
      resources: [ "configmaps" ]
      verbs: ["get", "update"]
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: kiagnose-configmap-access
    subjects:
    - kind: ServiceAccount
      name: example-sa
    roleRef:
      kind: Role
      apiGroup: rbac.authorization.k8s.io
      name: kiagnose-configmap-access
   ```
## Checkup Configuration

The main user-interface is a [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) object with a
certain structure.

In order to execute a checkup in an existing namespace, create the ConfigMap object in it.

### Input Fields
The user can configure the following under the `data` field:

| Property                | Description                                                                                                                 | Mandatory | Remarks                               |
|-------------------------|-----------------------------------------------------------------------------------------------------------------------------|-----------|---------------------------------------|
| spec.timeout            | After how much time should Kiagnose stop the running checkup                                                                | Yes       | 5m, 1h etc                            |
| spec.param.*            | Arbitrary strings that will be passed to the checkup as input parameters                                                    | No        | [0..N]                                |

Example configuration:

> **_NOTE:_** `metadata.namespace` field is optional.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-checkup-config
  namespace: <target-namespace>
data:
  spec.timeout: 5m
  spec.param.param_key_1: "value 1"
  spec.param.param_key_2: "value 2"
```

> **_NOTE:_** Kiagnose checks if the ConfigMap object had been previously used. If so, it will refuse to run the checkup. 

## Checkup Execution
In order to execute a checkup, Kiagnose needs to run a Kiagnose Job.
The Kiagnose Job acts as a "short-lived" controller, and controls the checkup lifecycle:
1. Read the checkup configuration from the user-supplied ConfigMap object.
2. Set up the objects required to run the checkup.
3. Run the checkup as a Job in the target Namespace and wait for its termination or timeout expiration.
4. Clean the objects created in the setup stage.


Apply a Kiagnose Job using the following manifest file:

> **_NOTE:_** `metadata.namespace` field is optional.

> **_NOTE:_** The `CONFIGMAP_NAMESPACE` and `CONFIGMAP_NAME` environment variables should point to the previously applied ConfigMap object.
 
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: example-checkup
  namespace: <target-namespace>
spec:
  backoffLimit: 0
  template:
    spec:
      serviceAccount: example-sa
      restartPolicy: Never
      containers:
        - name: example-checkup
          image: my-registry/example-checkup:main
          imagePullPolicy: Always
          env:
            - name: CONFIGMAP_NAMESPACE
              value: <target-namespace>
            - name: CONFIGMAP_NAME
              value: example-checkup-config
```

Kiagnose Job service-account should have permission to access the checkup ConfigMap

## Checkup Results Retrieval

After the checkup Job had completed, the results are made available at the user-supplied ConfigMap object:

```bash
kubectl get configmap example-checkup-config -n <target-namespace> -o yaml
```
### Output Fields
| Property                   | Description                                         | Mandatory | Remarks |
|----------------------------|-----------------------------------------------------|-----------|---------|
| status.succeeded           | Has the checkup succeeded                           | Yes       |         |
| status.failureReason       | Failure reason in case of a failure                 | Yes       |         |
| status.startTimestamp      | Checkup start timestamp                             | Yes       |         |
| status.completionTimestamp | Checkup completion timestamp                        | Yes       |         |
| status.result.*            | Arbitrary strings that were reported by the checkup | No        | [0..N]  |

Example output:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-checkup-config
  namespace: <target-namespace>
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

In order to read the checkup's logs (during or after its execution):
```bash
kubectl logs job.batch/<checkup-job-name> -n <target-namespace>
```

Remove the checkup Job and the ConfigMap object when the logs and the results are no longer needed:
```bash
kubectl delete job.batch/<checkup-job-name> -n <target-namespace>
kubectl delete configmap <ConfigMap name> -n <target-namespace>
```

## Checkup Removal
In order to remove a checkup from the cluster:
1. Remove any leftover checkup jobs and configmaps in the namespace. 
2. If the checkup's image is stored on your` registry - remove it.
