DPDK Checkup
=

**Authors**: [Edward Haas](https://github.com/eddev), [Orel Misan](https://github.com/orelmisan), [Or Mergi](https://github.com/ormergi), [Ram Lavi](https://github.com/RamLavi).

# Summary

Provide a functionality to validate [DPDK](https://www.dpdk.org/) communication going through a virtual machine that runs on a [Kubernetes](https://kubernetes.io/) cluster with [KubeVirt](https://kubevirt.io/).

# Motivation

Configuring DPDK inside a cluster is a multi-step configuration task. A cluster administrator can benefit from a checkup that can help verify that the overall configuration supports execution of a DPDK application.

The DPDK checkup offers an independent tool that can perform as an acceptance test, helping the administrator to highlight potential mistakes in the cluster.
Using this checkup, The administrator gains a powerful debugging tool, that helps better maintain the cluster.

## Goals

* Validate DPDK communication going through a VMI, on a predefined network.
* Measure latency of traffic going through a VMI, on a predefined network.
* Report results (and logs) with the cluster administrator.

## Non-Goals

* Define or configure the network under test.
* Define or configure the DPDK related configurations on the node.

# Proposal

## Definition of Users
* Kubernetes Cluster Administrators

## User Stories

As a Kubernetes Cluster Administrator I would like to:
1. Check that the cluster network is DPDK-ready by running the DPDK-checkup.
2. Optionally provide the maximum accepted latency, such that beyond this value, the check will fail.
3. Get a report/log if the DPDK-checkup fails, so that I could further debug my cluster.
