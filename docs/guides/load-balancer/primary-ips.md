<!--
---
date: "2026-10-06"
date_changed: "2026-10-06"
title: "Create a Load Balancer with Primary IPs"
tags: []
language: "en"
description: ""
docs_type: ["how_to"]
product_category: ["Integrations"]
translation: ["Integrations", "Hetzner Cloud Controller Manager", "How-To: Load Balancer", "Create a Load Balancer with Primary IPs"]
scrape_type: "whole"
priority: 90
---
-->

# Load Balancers with Primary IPs

By default, a Load Balancer gets system managed public IP addresses, which are created and deleted together with the Load Balancer. If you want to keep the public IP addresses of a Load Balancer, for example to recreate a Service without changing DNS records, you can create the Load Balancer with existing [Primary IPs](https://docs.hetzner.cloud/reference/cloud#tag/primary-ips) instead.

## Create the Primary IPs

Create a Primary IP for each address family you want to provide. The Primary IPs must be bound to the same location as the Load Balancer.

```bash
hcloud primary-ip create --type ipv4 --name my-lb-ipv4 --location hel1
hcloud primary-ip create --type ipv6 --name my-lb-ipv6 --location hel1
```

Look up their IDs, which are used in the annotations below:

```bash
hcloud primary-ip list
```

## Use the Primary IPs in a Service

Reference the Primary IPs by ID with the `load-balancer.hetzner.cloud/public-net-ipv4` and `load-balancer.hetzner.cloud/public-net-ipv6` annotations. Set `load-balancer.hetzner.cloud/location` to the location of the Primary IPs.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: example-service
  annotations:
    load-balancer.hetzner.cloud/location: hel1
    load-balancer.hetzner.cloud/public-net-ipv4: "4711"
    load-balancer.hetzner.cloud/public-net-ipv6: "4712"
spec:
  selector:
    app: example
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
```

Both annotations are optional and independent of each other. If you only set one of them, the Load Balancer gets a system managed IP address for the other address family.

The Service status reports the addresses of the Primary IPs, just like system managed addresses.

## Keep the Primary IPs when the Service is deleted

Deleting the Service deletes the Load Balancer. Whether the Primary IPs are deleted as well depends on the `auto_delete` setting configured on them beforehand.

Check the setting with `hcloud primary-ip describe my-lb-ipv4` and disable it if necessary:

```bash
hcloud primary-ip update --auto-delete=false my-lb-ipv4
```

## Limitations

- **Only applied on creation.** The Primary IPs are assigned when the Load Balancer is created. Adding, changing or removing the annotations on an existing Load Balancer leads to a warning, but has no effect otherwise. To use different Primary IPs, delete and recreate the Service.
- **Requires the public network.** The annotations cannot be combined with `load-balancer.hetzner.cloud/disable-public-network: "true"` or the `HCLOUD_LOAD_BALANCERS_DISABLE_PUBLIC_NETWORK` environment variable. The Load Balancer creation fails in that case.
- **Unassigned and in the same location.** The Primary IPs must not be assigned to another resource, must not be blocked and must be bound to the location of the Load Balancer. Otherwise the Load Balancer creation fails. Check the Events of the Service for the error message.
