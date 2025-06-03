# Load Balancers with Private Networks

Load Balancer traffic to the targets can be routed via Private Networks. To achieve this, ensure you have set up a cluster with Private Network support according to [this guide](../private-network-setup.md).

## Sample Service with Networks:

If your Private Network configuration is correct, you can use the annotation `load-balancer.hetzner.cloud/use-private-ip`.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: example-service
  annotations:
    load-balancer.hetzner.cloud/location: hel1
    load-balancer.hetzner.cloud/use-private-ip: "true"
spec:
  selector:
    app: example
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
```

For IPVS based plugins (kube-router, kube-proxy in ipvs mode, etc...) make sure you supply 'load-balancer.hetzner.cloud/disable-private-ingress: "true"' annotation to your service or set "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS" environment variable to true on hcloud-cloud-controller-manager deployment as mentioned in a paragraph below. Otherwise, network plugin installs load balancer's IP address on system's dummy interface effectively looping IPVS system in a cycle. In such scenario cluster nodes won't ever pass load balancer's health probes
