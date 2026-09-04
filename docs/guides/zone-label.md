<!--
---
date: "2026-09-07"
date_changed: "2026-09-07"
title: "Disabling the zone label"
tags: []
language: "en"
description: ""
docs_type: ["getting_started"]
product_category: ["Integrations"]
translation: ["Integrations", "Kubernetes Cloud Controller Manager", "Getting Started", "Disabling the zone label"]
scrape_type: "whole"
priority: 70
---
-->

# Zone Label

When a node is initialized, the hcloud-cloud-controller-manager sets these topology labels on it:

| Label                                                                       | Value                                | Example     |
| --------------------------------------------------------------------------- | ------------------------------------ | ----------- |
| `topology.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/region` | Location of the Server               | `fsn1`      |
| `topology.kubernetes.io/zone`, `failure-domain.beta.kubernetes.io/zone`     | Legacy datacenter name of the Server | `fsn1-dc14` |

The zone label can be disabled by setting `HCLOUD_INSTANCES_ZONE_LABEL_ENABLED` to `false`. By default, the value is set to `true`.

**We recommend disabling the zone label for new clusters.** Datacenters are deprecated in the Hetzner Cloud API, and the legacy datacenter names the label uses are only known for a fixed set of older locations. For any newer location, the label falls back to the location name and is therefore identical to the region label. The location is the failure domain you actually want to spread workloads over, and it is already available via `topology.kubernetes.io/region`. We plan to remove the zone label entirely in the next major version.

The label remains enabled by default because disabling it on a running cluster changes node labels, which existing workloads may depend on. See [Existing clusters](#existing-clusters) below.

## Configuration via Helm

```yaml
# values.yaml
---
env:
  HCLOUD_INSTANCES_ZONE_LABEL_ENABLED:
    value: "false"
```

## Existing clusters

Topology labels are only applied while a node is being initialized. Changing `HCLOUD_INSTANCES_ZONE_LABEL_ENABLED` does not relabel nodes that are already part of the cluster, so after disabling it you end up with a mixed cluster: existing nodes keep their zone label, and nodes joining afterwards do not get one. Anything that selects on the zone label — `nodeSelector`, node affinities, `topologySpreadConstraints`, or the node affinity of already-provisioned `PersistentVolume`s — will treat these two groups differently.

Before disabling the label, check whether anything in your cluster relies on it:

```sh
kubectl get pods,pv --all-namespaces -o yaml | grep -c 'topology.kubernetes.io/zone\|failure-domain.beta.kubernetes.io/zone'
```

To reach a consistent state afterwards, either recreate your nodes, or remove the labels from the existing ones:

```sh
kubectl label nodes --all topology.kubernetes.io/zone- failure-domain.beta.kubernetes.io/zone-
```

## Robot servers

This setting only affects Hetzner Cloud Servers. [Robot](robot/README.md) servers are always labeled with their datacenter (e.g. `fsn1-dc14`).
