#!/usr/bin/env bash
set -ueo pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

if [[ -n "${DEBUG:-}" ]]; then set -x; fi

# Redirect all stdout to stderr.
{
  if ! hcloud version >/dev/null; then echo "ERROR: 'hcloud' CLI not found, please install it and make it available on your \$PATH"; exit 1; fi
  if ! k3sup version >/dev/null; then echo "ERROR: 'k3sup' not found, please install it and make it available on your \$PATH"; exit 1; fi
  if ! helm version >/dev/null; then echo "ERROR: 'helm' not found, please install it and make it available on your \$PATH"; exit 1; fi
  if [[ "${HCLOUD_TOKEN:-}" == "" ]]; then echo "ERROR: please set \$HCLOUD_TOKEN"; exit 1; fi

  # We run a lot of subshells below for speed. If any encounter an error, we shut down the whole process group, pronto.
  function error() {
    echo "Onoes, something went wrong! :( The output above might have some clues."
    kill 0
  }

  trap error ERR

  image_name=${IMAGE_NAME:-ubuntu-22.04}
  instance_count=${INSTANCES:-1}
  instance_type=${INSTANCE_TYPE:-cpx11}
  location=${LOCATION:-fsn1}
  network_zone=${NETWORK_ZONE:-eu-central}
  ssh_keys=${SSH_KEYS:-}
  # All k3s after January 2024 break our e2e tests, we hardcode
  # the versions for now until we can fix the source of this.
  # channel=${K3S_CHANNEL:-stable}
  k3s_version=${K3S_VERSION:-v1.28.5+k3s1}
  network_cidr=${NETWORK_CIDR:-10.0.0.0/8}
  subnet_cidr=${SUBNET_CIDR:-10.0.0.0/24}
  cluster_cidr=${CLUSTER_CIDR:-10.244.0.0/16}
  routes_enabled=${ROUTES_ENABLED:-true}
  scope="${SCOPE:-dev}"
  scope=${scope//[^a-zA-Z0-9_]/-}
  scope_name=hccm-${scope}
  label="managedby=hack,scope=$scope_name"
  ssh_private_key="$SCRIPT_DIR/.ssh-$scope"
  k3s_opts=${K3S_OPTS:-"--kubelet-arg cloud-provider=external"}
  k3s_server_opts=${K3S_SERVER_OPTS:-"--disable-cloud-controller --disable=traefik --disable=servicelb --flannel-backend=none --disable=local-storage --cluster-cidr ${cluster_cidr}"}

  echo -n "$HCLOUD_TOKEN" > "$SCRIPT_DIR/.token-$scope"

  export KUBECONFIG="$SCRIPT_DIR/.kubeconfig-$scope"

  ssh_command="ssh -i $ssh_private_key -o StrictHostKeyChecking=off -o BatchMode=yes -o ConnectTimeout=5"

  # Generate SSH keys and upload publkey to Hetzner Cloud.
  ( trap error ERR
    [[ ! -f $ssh_private_key ]] && ssh-keygen -t ed25519 -f $ssh_private_key -C '' -N ''
    [[ ! -f $ssh_private_key.pub ]] && ssh-keygen -y -f $ssh_private_key > $ssh_private_key.pub
    if ! hcloud ssh-key describe $scope_name >/dev/null 2>&1; then
      hcloud ssh-key create --label $label --name $scope_name --public-key-from-file $ssh_private_key.pub
    fi
  ) &

  # Create Network
  ( trap error ERR
     if ! hcloud network describe $scope_name >/dev/null 2>&1; then
       hcloud network create --label $label --ip-range $network_cidr --name $scope_name
       hcloud network add-subnet --network-zone $network_zone --type cloud --ip-range $subnet_cidr $scope_name
     fi
    ) &


  for num in $(seq $instance_count); do
    # Create server and initialize Kubernetes on it with k3sup.
    ( trap error ERR

      server_name="$scope_name-$num"

      # Maybe cluster is already up and node is already there.
      if kubectl get node $server_name >/dev/null 2>&1; then
        exit 0
      fi

      ip=$(hcloud server ip $server_name 2>/dev/null || true)

      if [[ -z "${ip:-}" ]]; then
        # Wait for SSH key
        until hcloud ssh-key describe $scope_name >/dev/null 2>&1; do sleep 1; done
        until hcloud network describe $scope_name 2>&1 | grep $subnet_cidr >/dev/null; do sleep 1; done

        createcmd="hcloud server create --image $image_name --label $label --location $location --name $server_name --ssh-key=$scope_name --type $instance_type --network $scope_name"
        for key in $ssh_keys; do
          createcmd+=" --ssh-key $key"
        done
        $createcmd
        ip=$(hcloud server ip $server_name)
      fi

      # Wait for SSH.
      until [ "$($ssh_command root@$ip echo ok 2>/dev/null)" = "ok" ]; do
        sleep 1
      done

      $ssh_command root@$ip 'mkdir -p /etc/rancher/k3s && cat > /etc/rancher/k3s/registries.yaml' < $SCRIPT_DIR/k3s-registries.yaml

      private_ip=$(hcloud server describe $server_name -o format="{{ (index .PrivateNet 0).IP }}")
      k3s_node_ip_opts="--node-ip ${ip}"
      if [[ "$routes_enabled" == "true" ]]; then
        # Only advertise the private IP if we have routing enabled, to avoid issues where the nodes can
        # not communicate with each other on the advertised addresses (ie. Robot Servers)
        k3s_node_ip_opts="--node-external-ip ${ip} --node-ip ${private_ip}"
      fi

      if [[ "$num" == "1" ]]; then
        # First node is control plane.
        k3sup install --print-config=false --ip $ip --k3s-version "${k3s_version}" --k3s-extra-args "${k3s_server_opts} ${k3s_opts} ${k3s_node_ip_opts}" --local-path $KUBECONFIG --ssh-key $ssh_private_key
      else
        # All subsequent nodes are initialized as workers.

        # Can't go any further until control plane has bootstrapped a bit though.
        until $ssh_command root@$(hcloud server ip $scope_name-1 || true) stat /etc/rancher/node/password >/dev/null 2>&1; do
          sleep 1
        done

        k3sup join --server-ip $(hcloud server ip $scope_name-1) --ip $ip --k3s-channel $channel --k3s-extra-args "${k3s_opts} ${k3s_node_ip_opts}" --ssh-key $ssh_private_key
      fi
    ) &

    # Wait for this node to show up in the cluster.
    ( trap error ERR; set +x
      until kubectl wait --for=condition=Ready node/$scope_name-$num >/dev/null 2>&1; do sleep 1; done
      echo $scope_name-$num is up and in cluster
    ) &
  done

  ( trap error ERR
    # Control plane init tasks.
    # This is running in parallel with the server init, above.

    # Wait for control plane to look alive.
    until kubectl get nodes >/dev/null 2>&1; do sleep 1; done;

    # Deploy private registry.
    ( trap error ERR
      if ! helm status -n kube-system registry >/dev/null 2>&1; then
        helm upgrade -install registry docker-registry \
          --repo=https://helm.twun.io \
          -n kube-system \
          --version 2.2.2 \
          --set service.clusterIP=10.43.0.2 \
          --set 'tolerations[0].key=node.cloudprovider.kubernetes.io/uninitialized' \
          --set 'tolerations[0].operator=Exists'
      fi
      ) &

    # Install Cilium.
    ( trap error ERR
      if ! helm status -n kube-system cilium >/dev/null 2>&1; then
        values=(
          --set ipam.mode=kubernetes
        )
        if [[ "$routes_enabled" == "true" ]]; then
          # When using the Network Routes, we do not need (or want) Cilium to handle these ranges
          values+=(
            --set tunnel=disabled
            --set ipv4NativeRoutingCIDR="$cluster_cidr"
          )
        fi
        helm upgrade -install cilium cilium --repo https://helm.cilium.io/ -n kube-system --version 1.13.1 "${values[@]}"
      fi) &

    # Create HCLOUD_TOKEN Secret for hcloud-cloud-controller-manager.
    ( trap error ERR
      if ! kubectl -n kube-system get secret hcloud >/dev/null 2>&1; then
        data=(
          --from-literal="token=$HCLOUD_TOKEN"
          --from-literal="network=$scope_name"
        )
        if [[ -v ROBOT_USER ]]; then
          data+=(
            --from-literal="robot-user=$ROBOT_USER"
            --from-literal="robot-password=$ROBOT_PASSWORD"
          )
        fi
        kubectl -n kube-system create secret generic hcloud "${data[@]}"
      fi) &
    wait
  ) &
  wait
  echo "Success - cluster fully initialized and ready, why not see for yourself?"
  echo '$ kubectl get nodes'
  kubectl get nodes
  export CONTROL_IP=$(hcloud server ip "$scope_name-1")
} >&2

echo "export KUBECONFIG=$KUBECONFIG"
$SCRIPT_DIR/registry-port-forward.sh
echo "export SKAFFOLD_DEFAULT_REPO=localhost:30666"
echo "export CONTROL_IP=$CONTROL_IP"
