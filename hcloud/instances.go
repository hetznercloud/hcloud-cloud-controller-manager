/*
Copyright 2018 Hetzner Cloud GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hcloud

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/hcloud"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type addressFamily int

const (
	AddressFamilyDualStack addressFamily = iota
	AddressFamilyIPv6
	AddressFamilyIPv4
)

type instances struct {
	client        *hcloud.Client
	addressFamily addressFamily
	networkID     int
}

func newInstances(client *hcloud.Client, addressFamily addressFamily, networkID int) *instances {
	return &instances{client, addressFamily, networkID}
}

// lookupServer attempts to locate the corresponding hcloud.Server for a given v1.Node
// It returns an error if the Node has an invalid provider ID or if API requests failed.
// It can return a nil hcloud.Server if no
func (i *instances) lookupServer(ctx context.Context, node *v1.Node) (*hcloud.Server, error) {
	var server *hcloud.Server
	if node.Spec.ProviderID != "" {
		serverID, err := providerIDToServerID(node.Spec.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert provider id to server id: %w", err)
		}

		server, _, err = i.client.Server.GetByID(ctx, serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server \"%d\": %w", serverID, err)
		}
	} else {
		var err error
		server, _, err = i.client.Server.GetByName(ctx, node.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup server \"%s\": %w", node.Name, err)
		}
	}
	return server, nil
}

func (i *instances) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	const op = "hcloud/instancesv2.InstanceExists"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupServer(ctx, node)
	if err != nil {
		return false, err
	}

	return server != nil, nil
}

func (i *instances) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	const op = "hcloud/instancesv2.InstanceShutdown"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupServer(ctx, node)
	if err != nil {
		return false, err
	}

	return server.Status == hcloud.ServerStatusOff, nil
}

func (i *instances) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	const op = "hcloud/instancesv2.InstanceMetadata"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupServer(ctx, node)
	if err != nil {
		return nil, err
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:    serverIDToProviderID(server.ID),
		InstanceType:  server.ServerType.Name,
		NodeAddresses: i.nodeAddresses(ctx, server),
		Zone:          server.Datacenter.Name,
		Region:        server.Datacenter.Location.Name,
	}, nil
}

func (i *instances) nodeAddresses(ctx context.Context, server *hcloud.Server) []v1.NodeAddress {
	var addresses []v1.NodeAddress
	addresses = append(
		addresses,
		v1.NodeAddress{Type: v1.NodeHostName, Address: server.Name},
	)

	if i.addressFamily == AddressFamilyIPv4 || i.addressFamily == AddressFamilyDualStack {
		if !server.PublicNet.IPv4.IsUnspecified() {
			addresses = append(
				addresses,
				v1.NodeAddress{Type: v1.NodeExternalIP, Address: server.PublicNet.IPv4.IP.String()},
			)
		}
	}

	if i.addressFamily == AddressFamilyIPv6 || i.addressFamily == AddressFamilyDualStack {
		if !server.PublicNet.IPv6.IsUnspecified() {
			// For a given IPv6 network of 2001:db8:1234::/64, the instance address is 2001:db8:1234::1
			hostAddress := server.PublicNet.IPv6.IP
			hostAddress[len(hostAddress)-1] |= 0x01

			addresses = append(
				addresses,
				v1.NodeAddress{Type: v1.NodeExternalIP, Address: hostAddress.String()},
			)
		}
	}

	// Add private IP from network if network is specified
	if i.networkID > 0 {
		for _, privateNet := range server.PrivateNet {
			if privateNet.Network.ID == i.networkID {
				addresses = append(
					addresses,
					v1.NodeAddress{Type: v1.NodeInternalIP, Address: privateNet.IP.String()},
				)
			}
		}
	}
	return addresses
}
