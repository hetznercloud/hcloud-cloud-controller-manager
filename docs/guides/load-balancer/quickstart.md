# Quickstart

This guide will explain how to setup a simple Load Balancer.

For every Kubernetes `Service` of type `LoadBalancer` the HCCM will create a Hetzner Cloud Load Balancer with the necessary configuration.

## Sample Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: example-service
  annotations:
    load-balancer.hetzner.cloud/location: hel1
spec:
  selector:
    app: example
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
```

This sample service will create a Load Balancer in the location `hel1`. The `listen_port` will be 80. The `destination_port` will be a random node port selected by Kubernetes. Traffic arriving at the Load Balancer on Port 80 will be routed to the public interface of the targets on a node port.
