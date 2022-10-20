# Kiagnose

[Kiagnose](https://github.com/kiagnose/kiagnose) is a [Kubernetes](https://kubernetes.io) diagnostic framework which enables validation of cluster functionality.

A checkup is a containerized application, which checks that a certain cluster functionality is working.
A checkup can be provided by a third party vendor, and should adhere to the Kiagnose checkup API.

Kiagnose executes a checkup in an existing Namespace.
Kiagnose passes user-supplied configuration to the checkup, and reports the checkup's results upon termination.

# Usage
## Prerequisites
In order to use Kiagnose you should have:
1. A running Kubernetes cluster.
2. Namespace-Admin privileges on this cluster.
3. [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) utility.

## Kiagnose Installation

Please see the [installation instructions](./README.install.md/#kiagnose-installation).

## Kiagnose Removal

Please see the [removal instructions](./README.install.md/#kiagnose-removal).

## Checkup Installation
A vendor creating a checkup should provide:
1. Checkup's documentation (what it does, how to configure it, what are its output, etc). 
2. A checkup image.
3. A yaml file containing the following objects required for the checkup's execution:
 - A ServiceAccount object
 - Role object(s) - optional
 - RoleBinding object(s) - optional
 - ClusterRole object(s) - optional
 - ClusterRoleBinding object(s) - optional

> **_WARNING:_**
> 1. One should make sure a trustful checkup is used.
> 2. It is up to the cluster administrator to **ALWAYS** check the checkup's **image** and **permissions**
**BEFORE** applying them and running the checkup.
Kiagnose will **AUTOMATICALLY** use the user-supplied ServiceAccount for granting permissions to the checkup instance.

### Installation Steps
1. Make sure the checkup's image is accessible to your cluster.
2. Deploy the vendor-supplied permissions.

## Checkup Configuration

The main user-interface is a [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) object with a
certain structure.

In order to execute a checkup in an existing namespace, create the ConfigMap object in it.

### Input Fields
The user can configure the following under the `data` field:

| Property                | Description                                                                                                                 | Mandatory | Remarks                               |
|-------------------------|-----------------------------------------------------------------------------------------------------------------------------|-----------|---------------------------------------|
| spec.image              | Where to pull the checkup's image from                                                                                      | Yes       | A registry accessible to your cluster |
| spec.timeout            | After how much time should Kiagnose stop the running checkup                                                                | Yes       | 5m, 1h etc                            |
| spec.serviceAccountName | Name of ServiceAccount object, that exists in the target namespace and bound to the proper permissions the checkup requires | Yes       | "default" value is illegal            |
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
  spec.image: my-registry/example-checkup:main
  spec.timeout: 5m
  spec.serviceAccountName: example-sa
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
      serviceAccount: kiagnose
      restartPolicy: Never
      containers:
        - name: framework
          image: quay.io/kiagnose/kiagnose:main
          imagePullPolicy: Always
          env:
            - name: CONFIGMAP_NAMESPACE
              value: <target-namespace>
            - name: CONFIGMAP_NAME
              value: example-checkup-config
```

## Checkup Results Retrieval

The Kiagnose Job waits until the checkup Job is completed or timed-out.
After the Kiagnose Job had completed, the results are made available at the user-supplied ConfigMap object:

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
  spec.image: my-registry/example-checkup:main
  spec.timeout: 5m
  spec.serviceAccountName: example-sa
  spec.param.param_key_1: "value 1"
  spec.param.param_key_2: "value 2"
  
  status.succeeded: "true"
  status.failureReason: ''
  status.startTimestamp: "2022-05-25T11:53:49Z"
  status.completionTimestamp: "2022-05-25T11:54:46Z"
  status.result.key1: "result 1"
  status.result.key2: "result 2"
```

In order to read the Kiagnose's logs (during or after its execution):
```bash
kubectl logs job.batch/<Kiagnose-job-name> -n <target-namespace>
```

Remove the Kiagnose Job and the ConfigMap object when the logs and the results are no longer needed:
```bash
kubectl delete job.batch/<Kiagnose-job-name> -n <target-namespace>
kubectl delete configmap <ConfigMap name> -n <target-namespace>
```

## Checkup Removal
In order to remove a checkup from the cluster:
1. Remove vendor-supplied objects:
- A ServiceAccount object
- Role object(s) - optional
- RoleBinding object(s) - optional
- ClusterRole object(s) - optional
- ClusterRoleBinding object(s) - optional
2. If the checkup's image is stored on your registry - remove it.
