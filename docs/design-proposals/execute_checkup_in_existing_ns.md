Execute Checkups In an Existing Namespace
=

Author: [Orel Misan](https://github.com/orelmisan)

# Summary

Currently, kiagnose creates a dedicated ephemeral [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) object for each checkup it executes.
The suggested solution is for the user to create the user-supplied [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) objects in a pre-existing namespace.
The checkup will then be executed on that namespace.
The Kiagnose Job will continue to be executed at the `kiagnose` namespace.

# Motivation

Execution of checkups in ephemeral namespaces has several benefits:
1. Easy cleanup - Kiagnose deletes the entire namespace without knowing what objects had been created by the checkup.
2. Kiagnose does not need to have permissions to delete arbitrary objects.

This also has several downsides as well:
1. Checkups cannot use pre-existing objects in their namespace, so they have to be given permissions to read them from other namespaces (for example NetworkAttachmentDefinition objects).
2. Kiagnose is given permissions to create and delete Namespace objects, which is a pretty high permission.
3. Kiagnose is given permissions to bind ClusterRoles to ServiceAccounts it creates which is a very high permission as well.

One motivation is to mitigate the downsides of having checkups execution in ephemeral namespaces.
Another motivation is to give checkups the ability to use pre-existing objects in their namespace, such as NetworkAttachmentDefinition objects.

## Goal

Optionally execute a checkup in an existing namespace.

# Proposal

## Definition of Personas
- Kubernetes cluster administrators.

## User Stories
As a K8s cluster administrator:
- I want to create a checkup configuration, in the target namespace, in order to configure the checkup.
- I want to create a Kiagnose Job in the `kiagnose` namespace, so Kiagnose shall execute the checkup.
- I want to be able to monitor and observe if the checkup is undergoing or finished.
- I need to manually verify that the checkup had cleaned after itself.

## API Extensions
### Checkup Interface
A checkup shall be required to mark all objects created by it, in order to help users recognize leftover objects.

## Additional Required Permissions
Kiagnose should be granted with “delete” permissions of the following objects:
1. ConfigMap - in order to remove the results ConfigMap object.
2. ServiceAccount.

## Logic Changes
If the `CONFIGMAP_NAMESPACE` environment variable is not equal to `kiagnose`:
1. The value of `CONFIGMAP_NAMESPACE` shall be considered as the target namespace's name.
2. Kiagnose shall create all the required objects for the checkup in the target namespace.
3. The checkup’s teardown shall include deletion the following objects:
   - Results ConfigMap.
   - RoleBindings.
   - Role.
   - ServiceAccount.
4. Kiagnose shall not delete a non-ephemeral namespace.

## Implementation Steps
1. Create the results ConfigMap object with the name of the user-supplied ConfigMap + a suffix.
2. Create a Role to update / patch the results ConfigMap with the name of the user-supplied ConfigMap + a suffix.
3. Create the ServiceAccount object with the name of the user-supplied ConfigMap + a suffix.
4. Create a RoleBinding with the name of the user-supplied ConfigMap + a suffix.
5. Create the checkup Job with the name of the user-supplied ConfigMap + a suffix.
6. Change the teardown logic, to delete only created objects (and not the whole namespace).
7. Change the setup logic to create objects in an existing namespace.
8. [Optional] Label / put ownership information on objects Kiagnose creates.
9. [Optional] Add logic to run checkup in labeled namespaces.
10. [Optional] Add support for custom RoleBinding.
11. [Optional] Change VM latency checkup’s permissions to Role instead of ClusterRole.

# Appendix
A future change could be to remove the ephemeral namespace feature of Kiagnose.
