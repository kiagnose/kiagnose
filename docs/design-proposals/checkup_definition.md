Kiagnose Checkup Specification
=

Author: [Edward Haas](https://github.com/eddev)

# Summary
Define a kiagnose *checkup* specification and a certification process,
to standardize and assist in creating and consuming checkups.

# Motivation
The kiagnose project has started off as a simple job that ran a checkup.

The data inputted by the user and the results can be split in two categories:
- Generic, common to all checkups.
- Extended, unique and specific per each checkup.

In order to allow kiganose checkups to integrate in the existing Kubernetes and OpenShift
ecosystem, multiple approaches have been raised.
The existing approach and the new suggestions focus on providing a running application (and controllers),
either as a reference or as a service.

Classic services for checkup execution & control encounter issues of permission management and elevation concerns.
On the other hand, focusing on specific checkups as a stand-alone product, lack the ability to impose a common/generic
API, which in turn prevents clients from operating on checkups independently of the checkup kind/type and require
frequent intervention from the cluster admin.

There is a need to find a solution which can impose hard requirements, like a common API but
at the same time provide flexibility for vendors to create checkups as they see fit and per
their ability.
Placing a low entry bar for creating a checkup, opens up kiagnose for a larger community of checkup creators
and in turn to a wider consumption potential.

## Goal
- Minimize the required knowledge to compose a checkup.
- Provide as much independence to checkup writers as possible.
- Assure consumers can identify and consume checkups regardless of type.
- Minimize (or even eliminate) the need for a cluster-admin to approve each checkup.

# Proposal

## Definition of Personas
- Kiagnose checkup users.
- Kiagnose checkup authors.

## User Stories
As a checkup user:
- I want to identify available checkups for me to run.
- I want to run an individual checkup with default parameters on a specific target namespace.
- I want to run an individual checkup with non-default parameters on a specific target namespace.
- I want to observe the checkup progress and its final result.
- I want to collect all relevant logs and/or artifacts after the execution.

As a checkup author:
- I want to have a specification that defines what I must conform to.
- I want to have references to existing checkups.
- I want to have tutorials for writing a checkup.
- I want to use tooling and libraries that assist in writing a checkup.
- I want to certify my checkup once (I think) it is ready.
- I want to make my checkup available for deployment on K8S based clusters.

## Solution Overview
The proposal approach is focusing on defining a checkup through specification, allowing
checkup authors to implement checkups per hard requirements and at the same time to give
freedom on how they are implemented and extended.

Checkup consumers on the other hand, can use the specifications to interact with checkups
in a generic manner.

De facto, the specification defines a minimum standard that is common to all checkups.
To assure the checkup is conforming to the specification, a certification process is
proposed.

While specifications are enough to guide the creation of checkups, it would also be
beneficial to provide tooling to assist in the checkup development.
Such tooling may include libraries, scripts and tests.

The implementation and its details are left to the checkup author.
Details about using a controller, operator or any other means is left to the author
discretion, as long as the solution conforms to the specification.

The specification includes mandatory and optional fields.
Mandatory fields must be implemented, while optional fields are left to the checkup author
discretion.

> **_NOTE:_**  The proposal has been influenced by the [CNI](https://github.com/containernetworking/cni)
> project.

## Specification

### Version
The kiagnose checkup specification is versioned in order to allow checkup vendors to express
to which specification their checkups conform to.
Consumer clients that interact with checkups may use this information to access and operate checkups.

### Format
Data to and from a checkup will be based on JSON.

To allow for a standard layer of communication, a Custom Resource (CR) is chosen to pass the data.

### Custom Resource Definition
A CRD will be provided to specify the fields that the specification requires.
Checkup authors are free to extend the data with additional fields.

> **_NOTE:_** The usage of a CRD does not imply that there is a need for a controller
> when implementing the checkup. The CRD is used here to define an API, nothing more.

> **_NOTE:_** Checkups may use their own CRD/s for their operation. The CRD specified
> in this section is to be used to interact with the checkup clients.

Checkup deployments should make sure the CRD exists and deploy it if necessary.

```yaml
apiVersion: kiagnose.io/v1alpha1
kind: Checkup
metadata:
  name: CheckupExample
spec:
  version: <Checkup kind name and version (e.g. `foo/v1`)>
  image: <checkup image name>
  timeout: <timeout to wait for checkup to finish [min]>
  serviceAccountName: <SA that provides the required permissions to run the checkup>
  schema: <Optional JSON-schema definition that describes the extended fields under the `param` field>
  param: <Optional key/value fields that extends the base CRD, specific per checkup kind>
```

> **_NOTE:_** The optional `schema` field value contains a JSON-schema that describe the structure of the extension
> fields a specific checkup introduces. The format is the same as the one used in a CRD manifest, at the 
> [openAPIV3Schema](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#jsonschemaprops-v1-apiextensions-k8s-io)
> field.
> Clients may use this schema to learn what are the supported extension fields.

## Tooling
Tools that assist in creating checkups highly depends on how the checkup is implemented.
In case one wants to develop a controller that monitors a specific checkup kind, scripts and libraries
can be provided to assist authors in processing common tasks (e.g. reconcile loop template, watching the CR, validating
base fields, enforcing timeout).

E2E tests which validate the checkup specification can be included, allowing authors to validate their checkups before
proceeding to certify them.

## Deployment Concerns
As with permissions, the implementation method also has affect on the deployment of a checkup.

- Cluster Controller: If an individual checkup kind has a dedicated controller, it requires a Cluster Admin intervention
  for each checkup kind.
- Namespace Controller: For individual checkup kind controllers that are deployed in a target namespace, the
  Cluster Admin is usually not involved in the process of the deployment.
- Namespace Job: As implemented originally, a checkup can be executed using a simple job, removing the need to deploy
  controllers and simplifying in many cases the deployment steps.

> **_NOTE:_** Deploying controllers usually also imply operators involved (such controllers need to be deployed and
> upgraded).

## Permission Concerns
The checkup specification gives freedom to implement the checkups in any form possible.
It is however valuable to be aware of the implications of the implementation method chosen, in terms of security.

- Cluster Controller: Controllers that run in a dedicated namespace usually have wiser permissions, covering all cluster
  namespaces.
- Namespace Controller/Job: Controllers or simple jobs that run in the target namespace, have permissions only in that namespace
  and therefore have a smaller permission set.

## Certification

TBD

# Appendix A: Checkup Example (VM Latency Checkup)

TBD
