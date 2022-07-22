# Way towards v1

This document aims to define acceptance criteria that should guide redesign of
Kiagnose, driving it to v1.

The goal of the v1 release is to provide a solid core of the framework and
stable APIs. The goal is not to provide a feature rich tool.

## Personas

1. Cluster admin
2. Vendor writting a checkup
3. Project admin running one or more checkups

## User requirements

1. Nothing but `kubectl` should be required on the client-side to interact with
   checkups.
2. Kiagnose must not create a service account or bind any roles. It must rely on
   existing Kubernetes autorization mechanisms, and on present RBAC
   configuration.
3. Checkup must be initiated in the namespace it was requested from.
4. A project-admin can run a checkup without any prior work done by
   cluster-admin.
5. Checkups must share a single well-defined API endpoint.

## Out of v1 scope

1. Kiagnose is not required to cleanup of objects it did not create, e.g. objects
   created by checkups.
2. Kiagnose does not need to provide tooling allowing checkups to export
   artifacts.

## User stories

 * As a cluster admin,
   I don't want to be bothered by project admins wanting to run a test of their
   application, that could be executed with their current privileges.
 * As a cluster admin,
   I'm only willing to deal with native Kubernetes resources,
   I don't want to be bothered with any Kiagnose-specific resources.
 * As a project admin,
   I want to be able to run a checkup in a given namespace,
   to confirm that I configured a namespaced operator (e.g. MariaDB) correctly
   in my namespace. I expect the checkup vendor to declare all the special
   resources needed by it (e.g. NAD, memory quota, device plugin). I would make
   the necessary preparations to make these resources ready in my project.
 * As a project admin,
   I would like to automate running several checkups.  Therefore I need a clear
   API to pass parameters to a checkup and collect its output. Note: To operate
   over heterogenous set of checkups and expose them as a single resource, they
   need to share the same API group and kind.
 * As a checkup vendor,
   I would like to have a clear API that my checkup must adhere to, so it is
   easy to integrate to the framework.

