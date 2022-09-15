# Checkup API

A checkup is a containerized application, which checks that a certain cluster functionality is working properly.
A checkup provided by a third party vendor should adhere to the API described in this document.

Kiagnose executes a checkup as a [Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/) in an existing [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).
> **_Note:_** In order to get the namespace in which the checkup runs: 
> from within a Pod - read the content of the following file:
> `/var/run/secrets/kubernetes.io/serviceaccount/namespace`
>
> Please refer to [Directly accessing the REST API](https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api) for more details.

The checkup lifecycle is:
1. Kiagnose creates the following objects for a checkup instance:
   - A [ServiceAccount](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
   - An empty [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) object to store the checkup's results (results ConfigMap).
   - [Role](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole) and [RoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding) which grants write permissions of the `ConfigMap`.
   - [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding) objects to `ClusterRole` objects that were defined by the user. 
2. Kiagnose runs the checkup as a [Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/), with a single-container [Pod](https://kubernetes.io/docs/concepts/workloads/pods/).
3. Kiagnose configures the checkup using [environment variables](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/).
4. Kiagnose expects the checkup's [results](#output) to be written to the results ConfigMap. 

A checkup is free to create arbitrary objects in the target Namespace.
It is expected that the checkup will delete the objects it has created.

> **_Note:_** Please see the [Accessing the API from within a Pod](https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#accessing-the-api-from-within-a-pod)
for more details about accessing the Kubernetes API from within a Pod.

## Input

### Pre-set Inputs

The checkup should expect the following environment variables:

| Environment Variable Name    | Description                 | Remarks                                                      |
|------------------------------|-----------------------------|--------------------------------------------------------------|
| `CHECKUP_NAME`               | Checkup Name                | Could be used by a checkup to name child objects             |
| `RESULT_CONFIGMAP_NAMESPACE` | Results ConfigMap Namespace | Used by the underlying Pod need to write the checkup results |
| `RESULT_CONFIGMAP_NAME`      | Results ConfigMap name      | Used by the underlying Pod need to write the checkup results |

### Custom Inputs

Kiagnose enables passing custom parameters to a checkup.
Excluding the names of the pre-set environment variables, the checkup author is free to define additional parameters names:
- The parameters will be specified by the user on the user-supplied `ConfigMap`, under the `data` field.
- The parameters should match the following format: `spec.param.<parameter key>: <key value>`.

Kiagnose strips the `spec.param.` prefix, and passes the rest of the key as an environment variable in upper case.

For example:

The following parameter in the user-supplied `ConfigMap` object:
```yaml
spec.param.my_key: my value
```

Will be accessible to the checkup as the following environment variable:
```bash
MY_KEY="my value"
```

## Output

Kiagnose expects the checkup results to be reported under the `data` field of the result `ConfigMap` object.
The checkup is required to fill in the following fields:

| Key                    | Description               | Remarks                                         |
|------------------------|---------------------------|-------------------------------------------------|
| `status.succeeded`     | Is the checkup successful | "true"/"false"  as a string                     |
| `status.failureReason` | Checkup failure reason    | `<empty string/input/setup/check>: <free text>` |

Custom results should be concatenated to the `data` field of the result ConfigMap object, with `status.result.` prefix.
Kiagnose copies the checkup's results to the user-supplied `ConfigMap` object.

## Deliverables
The checkup author is expected to provide:
1. A container image, which could run in a Kubernetes cluster.
2. Documentation of:
   - Required and optional parameters, that affects its behavior.
   - Checkup results and their meaning (physical units etc.).
3. Manifest files of required objects:
- A ServiceAccount object
- Role object(s) - optional
- RoleBinding object(s) - optional
- ClusterRole object(s) - optional
- ClusterRoleBinding object(s) - optional
