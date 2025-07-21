# Deploying a k3s Cluster with Hetzner Cloud Controller Manager and NGINX Ingress

In this tutorial, you will learn how to set up a lightweight Kubernetes cluster on **Hetzner Cloud** using **k3s**, install the **Hetzner Cloud Controller Manager (HCCM)**, and deploy the **Ingress NGINX Controller** to expose services via a **Hetzner Load Balancer**.

**What you’ll achieve:**

- A running Kubernetes cluster with nodes in Hetzner Cloud
- Automatic Load Balancer creation using HCCM
- An NGINX Ingress Controller with a test application exposed

**Prerequisites:**

- Basic knowledge of Kubernetes concepts
- Installed command-line tools:
  - [hcloud CLI](https://github.com/hetznercloud/cli)
  - [k3sup](https://github.com/alexellis/k3sup)
  - [kubectl](https://kubernetes.io/docs/tasks/tools/)
  - [Helm](https://helm.sh/docs/intro/install/)
  - [jq](https://jqlang.org/)

---

## 1. Set up Hetzner Cloud Resources

### 1.1. Create SSH Key

Lets create and upload a SSH key, which is used by k3sup to access our servers during the Kubernetes installation process.

```bash
ssh-keygen -t ed25519 -f ./hcloud-k3s
hcloud ssh-key create --name k3s-key --public-key-from-file ./hcloud-k3s.pub
```

### 1.2. Create Control Plane and Worker Nodes

Our cluster will consist of a single control plane with a single worker. These servers will be located in Falkenstein, use Ubunut as a base image and use the server-type cx22.

```bash
hcloud server create --name tutorial-control-plane \
  --type cx22 \
  --location fsn1 \
  --image ubuntu-24.04 \
  --ssh-key k3s-key

hcloud server create --name tutorial-worker \
  --type cx22 \
  --location fsn1 \
  --image ubuntu-24.04 \
  --ssh-key k3s-key
```

Use `hcloud server list` to check that the servers are running and note their public IP addresses.

---

## 2. Deploy k3s Cluster

### 2.1. Install k3s on Control Plane

```bash
k3sup install \
  --ip=<CONTROL_PLANE_PUBLIC_IP> \
  --ssh-key ./hcloud-k3s \
  --local-path=./kubeconfig \
  --k3s-extra-args="\
    --kubelet-arg=cloud-provider=external \
    --disable-cloud-controller \
    --disable-network-policy \
    --disable=traefik \
    --disable=servicelb \
    --node-ip='<CONTROL_PLANE_PUBLIC_IP>'"
```

- `cloud-provider=external` prepares the cluster for an external cloud controller. In our case the hcloud-cloud-controller-manager.
- `--disable-network-policy`, `--disable=traefik` and `--disable=servicelb` removes k3s builtin components, which would collide with the products we deploy in this tutorial.

### 2.2. Join Worker Node

```bash
k3sup join \
  --ip=<WORKER_PUBLIC_IP> \
  --server-ip=<CONTROL_PLANE_PUBLIC_IP> \
  --user=root \
  --ssh-key ./hcloud-k3s \
  --k3s-extra-args="\
    --kubelet-arg=cloud-provider=external \
    --node-ip='<WORKER_PUBLIC_IP>'"
```

Set the kubeconfig file:

```bash
export KUBECONFIG=./kubeconfig
kubectl get nodes -o wide
```

---

## 3. Install Hetzner Cloud Controller Manager (HCCM)

### 3.1. Create Hetzner Cloud API Token Secret

```bash
kubectl -n kube-system create secret generic hcloud --from-literal=token=<YOUR_HCLOUD_API_TOKEN>
```

### 3.2. Install HCCM via Helm

```bash
helm repo add hcloud https://charts.hetzner.cloud
helm repo update
helm install hccm hcloud/hcloud-cloud-controller-manager -n kube-system
```

---

## 4. Install Ingress NGINX Controller

### 4.1. Prepare Helm Values for Load Balancer

Ingress Nginx will deploy a Kubernetes Service of type `LoadBalancer`. Here we configure the annotations added to the Kubernetes Service via Helm values. These annotations are read by the HCCM to configure the Hetzner Cloud Load Balancer.

```yaml
# values.yml
controller:
  service:
    annotations:
      load-balancer.hetzner.cloud/location: "fsn1"
      load-balancer.hetzner.cloud/name: "tutorial-lb"
```

### 4.2. Install with Helm

```bash
helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace \
  -f values.yml
```

:white_check_mark: After a couple of seconds, check for the Hetzner Cloud Load Balancer:

```bash
hcloud load-balancer list
```

---

## 5. Deploy a Test Application

### 5.1. Deploy Hello App and Ingress

```yaml
# hello-app.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
        - name: hello
          image: hashicorp/http-echo
          args: ["-text=Hello from Kubernetes!"]
          ports:
            - containerPort: 5678
---
apiVersion: v1
kind: Service
metadata:
  name: hello-service
spec:
  selector:
    app: hello
  ports:
    - port: 80
      targetPort: 5678
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hello-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: hello.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: hello-service
                port:
                  number: 80
```

```bash
kubectl apply -f hello-app.yaml
```

---

## 6. Access the Application

We can now access the application. To do this without creating a proper DNS entry or editing `/etc/hosts`, we use a small workaround: temporarily resolve `hello.local` to the Load Balancer’s public IPv4 address.

```bash
curl --resolve hello.local:80:$(hcloud load-balancer describe tutorial-lb --output json | jq -r .public_net.ipv4.ip) http://hello.local
```

:rocket: You should see:

```
Hello from Kubernetes!
```

## 7. Cleanup

As this tutorial was by no means a production setup, lets cleanup the resources we created.

```bash
hcloud server delete tutorial-control-plane tutorial-worker
hcloud load-balancer delete tutorial-lb
hcloud ssh-key delete k3s-key
```

---

**Related resources:**

- [HCloud Cloud Controller Manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager)
- [HCloud Cloud Controller Manager - Annotations](https://github.com/hetznercloud/hcloud-cloud-controller-manager/blob/v1.26.0/internal/annotation/load_balancer.go) # x-releaser-pleaser-version
- [Ingress NGINX Annotations](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/)
