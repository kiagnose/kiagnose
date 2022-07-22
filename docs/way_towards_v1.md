# Way towards v1

This document aims to define acceptance criteria that should guide redesign of
Kiagnose, driving it to v1.

The goal of the v1 release is to provide a solid core of the framework and
stable APIs. The goal is not to provide a feature rich tool.

## Personas

1. cluster-admin installing Kiagnose
2. Vendor writting a checkup
3. Cluster user running one or more checkups

## User requirements

1. Kiagnose is delivered through OLM and installed by the cluster administrator.
2. Checkups need to run with the same or lesser rights as the user which
   triggered them.
3. As a project owner I would like to automate running several checkups.
   Therefore I need a clear API to pass parameters to a checkup and collect its
   output.
4. As a Vendor I would like to have a clear API that my checkup must adhere to,
   so it is easy to integrate to the framework.

## Out of v1 scope

1. Kiagnose is not required to cleanup of objects it did not create, e.g. objects
   created by checkups.
2. Kiagnose does not need to provide tooling allowing checkups to export
   artifacts.
