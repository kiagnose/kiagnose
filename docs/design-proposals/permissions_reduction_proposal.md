Permissions Reduction Proposal
=

Author: [Orel Misan](https://github.com/orelmisan)

# Summary

Currently, kiagnose has a feature to execute a checkup in a dedicated
ephemeral [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).
This causes Kiagnose to require very high, cluster-wide permissions for its operation. For example:

1. Get, List, Create, delete and Watch `Namespace` objects.
2. Get, List, Create and Delete `ServiceAccount` objects.
3. Get, List, Create, Delete and Bind `ClusterRole` and `Role` objects.

The proposed solution is:

1. To use a user-supplied pre-configured ServiceAccount, with which the checkup will be executed.
2. Remove the ability to execute checkups inside ephemeral namespaces.

The proposed change will:

1. Limit checkup execution to existing namespaces.
2. Move the responsibility of creating the proper `ServiceAccount` object for the checkup to the user.
3. Remove the majority of the RBAC-related permissions Kiagnose has.

# Motivation

## Goal

1. Reduce the permissions Kiagnose requires for its operation.
2. Reduce the permissions checkups require for their operation.
3. Remove the ephemeral namespace feature.

# Overview

## Kiagnose Deployment

The deployment process shall stay without a change.

Kiagnose's ClusterRole object will have fewer privileges.

## Checkups Deployment

A checkup shall be deployed to an existing Namespace.

A checkup deployment manifest shall include the following objects:

- ServiceAccount.
- Role (optional).
- RoleBinding (optional).
- ClusterRole (optional).
- ClusterRoleBinding (optional).

The user who deploys these objects is responsible for reviewing the permissions given to the checkup.

## Checkup Configuration

Checkup configuration shall include the name of the pre-deployed checkup’s ServiceAccount object.

## Checkup Setup

The Kiagnose Job shall be created in the `kiagnose` namespace.
The Kiagnose Job shall create the following objects in the target namespace:

1. Results ConfigMap.
2. Role and RoleBinding to the user-supplied `ServiceAccount` so the checkup could write its results to the
   results `ConfigMap`.
3. Checkup Job connected to the pre-deployed checkup’s ServiceAccount object.

## Checkup Execution

Without a change.

## Checkup Teardown

The Kiagnose Job shall delete all objects it created during its execution:

- Job
- RoleBinding
- Role
- Results ConfigMap

Kiagnose shall not delete the user-supplied `ServiceAccount` or any of its user-supplied permissions.

# Proposal

## Definition of Personas

- Kubernetes cluster administrators.

## User Stories

As a K8s cluster administrator:

- I want to deploy Kiagnose with less powerful cluster-wide privileges.
- I want to deploy checkups-related RBAC objects (that have fewer privileges) to specific namespaces.
- I want to create the user-supplied ConfigMap, in the target namespace, in order to configure the checkup.
- I want to create a Kiagnose Job in the `kiagnose` namespace, so Kiagnose shall execute the checkup.
- I want to be able to monitor and observe if the checkup is undergoing or finished.
- I need to manually verify that the checkup had cleaned after itself.

## API Extensions

### User Interface

The following fields shall be included in the user-supplied `ConfigMap` object:

| Key                     | Description                                                              | Is Mandatory | Remarks                                      |
|-------------------------|--------------------------------------------------------------------------|--------------|----------------------------------------------|
| spec.image              | Where to pull the checkup's image from                                   | True         | A registry accessible to your cluster        |
| spec.timeout            | After how much time should Kiagnose stop the running checkup             | True         | 5m, 1h etc                                   |
| spec.serviceAccountName | Existing serviceAccount for the checkup’s Job                            | True         | `default` ServiceAccount name is not allowed |
| spec.spec.param.*       | Arbitrary strings that will be passed to the checkup as input parameters | False        | [0..N]                                       |

## Implementation Steps

1. Fix the `automation/e2e.sh` script's `--run-tests` option to use a local image of Kiagnose (and not the latest `main`
   from quay).
2. Add unit tests for failure cases of the target namespace feature.
3. Remove ephemeral namespace feature and permissions, update the documentation.
4. Add `spec.serviceAccountName` to config, and change the unit and e2e tests, update documentation.
5. Use user-supplied `serviceAccountName` when creating the checkup's Job.
6. Remove `ServiceAccount` teardown and setup logic, and permissions.
7. Remove ClusterRoleBinding logic and permissions.
8. Remove Roles and ClusterRoles from config.
9. Change VM latency checkup’s permissions to Role instead of ClusterRole, update documentation accordingly.
10. Update Echo checkup's documentation.
