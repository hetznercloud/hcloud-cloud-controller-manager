<!--
---
date: "2026-09-07"
date_changed: "2026-09-07"
title: "Address Family"
tags: []
language: "en"
description: ""
docs_type: ["how_to"]
product_category: ["Integrations"]
translation: ["Integrations", "Kubernetes Cloud Controller Manager", "How-To: Networking", "Address Family"]
scrape_type: "whole"
priority: 90
---
-->

# Address Family

To control the address family used when initializing a node, the `HCLOUD_INSTANCES_ADDRESS_FAMILY` environment variable can be set to `ipv4`, `ipv6` or `dualstack`. By default, the value is set to `ipv4`.

## Configuration via Helm

```yaml
# values.yaml
---
env:
  HCLOUD_INSTANCES_ADDRESS_FAMILY:
    value: "dualstack"
```
