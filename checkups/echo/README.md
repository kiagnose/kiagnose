# Echo checkup
This is an example checkup, used as a reference for creating more realistic checkups for the [Kiagnose](https://github.com/kiagnose/kiagnose) project.

## Inputs
The checkup expects the following environment variables to be supplied:
1. "RESULT_CONFIGMAP_NAMESPACE" - namespace of the results ConfigMap object.
2. "RESULT_CONFIGMAP_NAME" - name of the results ConfigMap object.
3. "CHECKUP_DATA" - a message to write to the results ConfigMap object.

## Outputs
The checkup writes its results to the `data` field of the results ConfigMap object:
```yaml
status.succeeded: "true"
status.failureReason: ""
status.result.echo: "$CHECKUP_DATA"
```

In case the "CHECKUP_DATA" environment variable is missing:
```yaml
status.succeeded: "false"
status.failureReason: "CHECKUP_DATA environment variable is missing"
```

## Build Instructions
### Prerequisites
- You have [podman](https://podman.io/) or other container engine capable of building images.
### Steps
```bash
$ ./automation/make.sh --build-checkup-image
# Using Docker to build the image:
$ CRI=docker ./automation/make.sh --build-checkup-image
```

## Manual Execution Instructions
### Prerequisites
- The checkup container is built, tagged and stored in a registry accessible to your cluster.
- You have "Admin" permissions on the K8s cluster.
- kubectl is configured to connect to your cluster.
### Steps
1. Deploy the checkup manifest using:
```bash
$ kubectl create -f manifests/dev/echo-checkup.yaml
```

2. To get the checkup results:
```bash
$ kubectl get configmap echo-checkup-results -n echo-checkup-ns -o yaml > results.yaml
```

3. To remove the created objects:
```bash
$ kubectl delete -f manifests/dev/echo-checkup.yaml
```
