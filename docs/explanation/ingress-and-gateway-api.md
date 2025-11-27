# Ingress & Gateway API

Ingress and Gateway API resources rely on controllers that watch for changes and take the actions needed to bring the system to the desired state.

A common example is the NGINX Ingress Controller, which watches for Ingress objects and configures an NGINX instance accordingly. To expose traffic to the outside world, the Ingress controller creates a Kubernetes Service of type `LoadBalancer`. At this point, the hcloud-cloud-controller-manager (HCCM) comes into play and provisions the corresponding Hetzner Cloud Load Balancer. This setup is referred to as an "Ingress-managed load balancer" (See: [What is an Ingress?](https://kubernetes.io/docs/concepts/services-networking/ingress/#what-is-ingress)).

The Ingress resource is currently being phased out in favor of the Gateway API resources. This introduces the need for new controllers that watch and reconcile Gateway API objects. One such controller is the **NGINX Gateway Fabric**, which configures an NGINX instance based on Gateway API specifications. Similar to the Ingress controller, it uses a Kubernetes Service of type `LoadBalancer` to expose traffic externally.

Hetzner Cloud Load Balancers currently do not support L7 routing. Because of this limitation, we have not built an Ingress controller, and for the same reason we do not plan to build a Gateway API controller at this time. If Hetzner Cloud Load Balancers eventually gain the necessary L7 capabilities, we would reevaluate the need for building such a product. Such a controller would live in its own project and would not be part of HCCM.
