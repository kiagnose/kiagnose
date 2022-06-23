# Echo checkup
This is an example checkup, used as a reference for creating more realistic checkups for the [Kiagnose](https://github.com/kiagnose/kiagnose) project.

## Usage
### Prerequisites

1. Kiagnose is [installed](../../README.install.md).
2. The cluster can access `quay.io/kiagnose/` to pull images.

### Installation

The checkup does not require additional permissions.

### Configuration

In the user-supplied `ConfigMap` object, specify an arbitrary string as the value of `spec.param.message` parameter.

In order to configure the checkup:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: echo-checkup-config
  namespace: kiagnose
data:
  spec.image: quay.io/kiagnose/echo-checkup:main
  spec.timeout: 1m
  spec.param.message: "Hi!"
EOF
```

### Execution

In order to execute the checkup:

1. Apply the Kiagnose job:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: echo-checkup1
  namespace: kiagnose
spec:
  backoffLimit: 0
  template:
    spec:
      ServiceAccountName: kiagnose
      restartPolicy: Never
      containers:
        - name: framework
          image: quay.io/kiagnose/kiagnose:main
          imagePullPolicy: Always
          env:
            - name: CONFIGMAP_NAMESPACE
              value: kiagnose
            - name: CONFIGMAP_NAME
              value: echo-checkup-config
EOF
```

2. Wait for the Kiagnose Job to finish:
```bash
kubectl wait --for=condition=complete --timeout=70s job/echo-checkup1 -n kiagnose
```

### Results Retrieval
After the Kiagnose Job had completed, retrieve the user-supplied `ConfigMap` to get the results:
```bash
kubectl get configmap echo-checkup-config -n kiagnose -o yaml
```

In a success scenario, the following results are expected:

```yaml
status.succeeded: "true"
status.failureReason: ""
status.result.echo: <same as spec.param.message>
```

In case the `message` parameter is not supplied, the following results are expected:
```yaml
status.succeeded: "false"
status.failureReason: "MESSAGE environment variable is missing"
```

Example results:

```yaml
apiVersion: v1
data:
  spec.image: quay.io/kiagnose/echo-checkup:main
  spec.param.message: Hi!
  spec.timeout: 1m
  status.completionTimestamp: "2022-06-06T11:00:10Z"
  status.failureReason: ""
  status.result.echo: Hi!
  status.startTimestamp: "2022-06-06T10:59:58Z"
  status.succeeded: "true"
kind: ConfigMap
metadata:
  creationTimestamp: "2022-06-06T10:59:56Z"
  name: echo-checkup-config
  namespace: kiagnose
  resourceVersion: "557"
  uid: e00b801c-3055-4c3c-9dc5-8a944f01d9de
```

Remove the Kiagnose Job and the ConfigMap object when the logs and the results are no longer needed:
```bash
kubectl delete job.batch/echo-checkup1 -n kiagnose
kubectl delete configmap echo-checkup-config -n kiagnose
```

## API
### Inputs
The checkup expects the following environment variables to be supplied:
1. "RESULT_CONFIGMAP_NAMESPACE" - namespace of the results ConfigMap object.
2. "RESULT_CONFIGMAP_NAME" - name of the results ConfigMap object.
3. "MESSAGE" - a message to write to the results ConfigMap object.

### Outputs
The checkup writes its results to the `data` field of the results ConfigMap object:
```yaml
status.succeeded: "true"
status.failureReason: ""
status.result.echo: "$MESSAGE"
```

In case the "MESSAGE" environment variable is missing:
```yaml
status.succeeded: "false"
status.failureReason: "MESSAGE environment variable is missing"
```

## Build Instructions
### Prerequisites
- You have [podman](https://podman.io/) or other container engine capable of building images.
### Steps
```bash
./automation/make.sh --build-checkup-image
# Using Docker to build the image:
CRI=docker ./automation/make.sh --build-checkup-image
```

## Manual Execution Instructions
### Prerequisites
- The checkup container is built, tagged and stored in a registry accessible to your cluster.
- You have "Admin" permissions on the K8s cluster.
- kubectl is configured to connect to your cluster.
### Steps
1. Deploy the checkup manifest using:
```bash
kubectl create -f manifests/dev/echo-checkup.yaml
```

2. To get the checkup results:
```bash
kubectl get configmap echo-checkup-results -n echo-checkup-ns -o yaml > results.yaml
```

3. To remove the created objects:
```bash
kubectl delete -f manifests/dev/echo-checkup.yaml
```
