<!--
---
date: "2026-09-07"
date_changed: "2026-09-07"
title: "Common issues"
tags: []
language: "en"
description: ""
docs_type: ["faq/troubleshooting"]
product_category: ["Integrations"]
translation: ["Integrations", "Kubernetes Cloud Controller Manager", "Troubleshooting", "Common issues"]
scrape_type: "whole"
priority: 100
---
-->

# Troubleshooting

## Load Balancers

### Load Balancer Targets not Added

If your node is not added as a Load Balancer target, use the following snippet to check if your nodes are excluded from external Load Balancers.

```bash
kubectl get nodes --show-labels | grep node.kubernetes.io/exclude-from-external-load-balancers
```
