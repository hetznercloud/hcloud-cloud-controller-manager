# Load Balancers

Load Balancer support is implemented in the Cloud Controller as of
version v1.6.0. For using the Hetzner Cloud Load Balancers you need to
deploy a `Service` of type `LoadBalancer`.

## Sample Service:

```
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

The sample service will create a Load Balancer in the location `hel1`
with a service with `listen_port = 80` and `destination_port = <random-node-port>`. So
every traffic that arrives at the Load Balancer on Port 80 will be
routed to the public interface of the targets on a node port, which is randomly selected by default. You can
change the behavior of the Load Balancer by specifying more annotations.
A list of all available annotations can be found on
[pkg.go.dev](https://pkg.go.dev/github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation#Name).
If you have the cloud controller deployed with Private Network Support,
we attach the Load Balancer to the specific network automatically. You
can specify with an annotation that the Load Balancer should use the
private network instead of the public network.

## Sample Service with Networks:

```
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

For IPVS based plugins (kube-router, kube-proxy in ipvs mode, etc...) make sure you
supply '**load-balancer.hetzner.cloud/disable-private-ingress: "true"**' annotation
to your service or set "**HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS**" environment variable
to true on hcloud-cloud-controller-manager deployment as mentioned in a paragraph below. Otherwise, network
plugin installs load balancer's IP address on system's dummy interface effectively
looping IPVS system in a cycle. In such scenario cluster nodes won't ever pass load balancer's health probes

## Cluster-wide Defaults

For convenience, you can set the following environment variables as cluster-wide defaults, so you don't have to set them on each load balancer service. If a load balancer service has the corresponding annotation set, it overrides the default.

- `HCLOUD_LOAD_BALANCERS_LOCATION` (mutually exclusive with `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`)
- `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE` (mutually exclusive with `HCLOUD_LOAD_BALANCERS_LOCATION`)
- `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS`
- `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP`
- `HCLOUD_LOAD_BALANCERS_ENABLED`

## Reference existing Load Balancers

If you already have a Load Balancer that you want to use in Kubernetes, for
example if you provisioned it in Terraform, you can specify these two
annotations:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: example-service
  annotations:
    load-balancer.hetzner.cloud/location: hel1
    load-balancer.hetzner.cloud/name: your-existing-lb-name
```

The Load Balancer will then be adopted by the hcloud-cloud-controller-manager,
and the services and targets are set up for your cluster.

If you delete this `Service` in Kubernetes, the hcloud-cloud-controller-manager
will delete the associated Load Balancer. If the Load Balancer is managed
through Terraform, this causes problems. To disable this, you can enable
deletion protection on the Load Balancer, this way hcloud-cloud-controller-manager
will just skip deleting it when the associated `Service` is deleted.

## Per-Port Protocol and Certificate Configuration

The hcloud-cloud-controller-manager supports configuring different protocols and certificates for different ports of a single service using per-port annotations.

### Per-Port Protocol Configuration

Use the `load-balancer.hetzner.cloud/protocol-ports` annotation to specify different protocols for different ports:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: multi-protocol-service
  annotations:
    load-balancer.hetzner.cloud/protocol-ports: "80:http,443:https,9000:tcp"
spec:
  type: LoadBalancer
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  - name: https
    port: 443
    targetPort: 8443
    protocol: TCP
  - name: tcp
    port: 9000
    targetPort: 9000
    protocol: TCP
  selector:
    app: my-app
```

### Per-Port Certificate Configuration

Use the `load-balancer.hetzner.cloud/http-certificates-ports` annotation to specify different certificates for different HTTPS ports:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: multi-https-service
  annotations:
    load-balancer.hetzner.cloud/protocol-ports: "443:https,8443:https"
    load-balancer.hetzner.cloud/http-certificates-ports: "443:cert1,cert2;8443:cert3"
spec:
  type: LoadBalancer
  ports:
  - name: https-main
    port: 443
    targetPort: 8443
    protocol: TCP
  - name: https-alt
    port: 8443
    targetPort: 8443
    protocol: TCP
  selector:
    app: my-app
```

### Format Specification

**Protocol Ports Format:**
- Format: `"port:protocol,port:protocol,..."`
- Example: `"80:http,443:https,9000:tcp"`
- Supported protocols: `tcp`, `http`, `https`

**Certificate Ports Format:**
- Format: `"port:cert1,cert2;port:cert3,..."`
- Example: `"443:cert1,cert2;8443:cert3"`
- Supports both certificate names and IDs
- Use semicolons (`;`) to separate different ports
- Use commas (`,`) to separate multiple certificates for the same port

### Fallback Behavior

- If per-port configuration is not specified for a port, the global annotation values are used
- Global annotations: `load-balancer.hetzner.cloud/protocol` and `load-balancer.hetzner.cloud/http-certificates`
- If no global annotation is set, defaults to `tcp` protocol
