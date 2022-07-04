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

## User Stories

