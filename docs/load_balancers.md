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
with a service with `listen_port = 80` and `destination_port = 8080`. So
every traffic that arrives at the Load Balancer on Port 80 will be
routed to the public interface of the targets on port 8080.  You can
change the behavior of the Load Balancer by specifying more annotations.
A list of all available annotations can be found on
[pkg.go.dev](https://pkg.go.dev/github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation).
If you have the cloud controller deployed with Private Network Support,
we attach the Load Balancer to the specific network automatically. You
can specifiy with an annotation that the Load Balancer should use the
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


