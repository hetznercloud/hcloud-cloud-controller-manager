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

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/syself/hetzner-cloud-controller-manager/internal/metrics"
	robotclient "github.com/syself/hetzner-cloud-controller-manager/internal/robot/client"
	"github.com/syself/hrobot-go/models"
	corev1 "k8s.io/api/core/v1"
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
	robotClient   robotclient.Client
	addressFamily addressFamily
	networkID     int64
}

var errServerNotFound = fmt.Errorf("server not found")

func newInstances(client *hcloud.Client, robotClient robotclient.Client, addressFamily addressFamily, networkID int64) *instances {
	return &instances{client, robotClient, addressFamily, networkID}
}

// lookupServer attempts to locate the corresponding hcloud.Server or models.Server (robot server) for a given v1.Node.
// It returns an error if the Node has an invalid provider ID or if API requests failed.
// It can return a nil [*hcloud.Server] if neither the ProviderID nor the Name matches an existing server.
func (i *instances) lookupServer(
	ctx context.Context,
	node *corev1.Node,
) (hcloudServer *hcloud.Server, bmServer *models.Server, isHCloudServer bool, err error) {
	if node.Spec.ProviderID != "" {
		var serverID int64
		serverID, isHCloudServer, err = providerIDToServerID(node.Spec.ProviderID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to convert provider id to server id: %w", err)
		}

		if isHCloudServer {
			hcloudServer, err = getHCloudServerByID(ctx, i.client, serverID)
			if err != nil {
				return nil, nil, false, fmt.Errorf("failed to get hcloud server \"%d\": %w", serverID, err)
			}
		} else {
			if i.robotClient == nil {
				return nil, nil, false, errMissingRobotCredentials
			}
			bmServer, err = getRobotServerByID(i.robotClient, int(serverID), node)
			if err != nil {
				return nil, nil, false, fmt.Errorf("failed to get robot server \"%d\": %w", serverID, err)
			}
		}
	} else {
		if isHCloudServerByName(string(node.Name)) {
			isHCloudServer = true
			hcloudServer, err = getHCloudServerByName(ctx, i.client, string(node.Name))
			if err != nil {
				return nil, nil, false, fmt.Errorf("failed to get hcloud server %q: %w", string(node.Name), err)
			}
		} else {
			if i.robotClient == nil {
				return nil, nil, false, errMissingRobotCredentials
			}
			bmServer, err = getRobotServerByName(i.robotClient, node)
			if err != nil {
				return nil, nil, false, fmt.Errorf("failed to get robot server %q: %w", string(node.Name), err)
			}
		}
	}
	return hcloudServer, bmServer, isHCloudServer, nil
}

func (i *instances) InstanceExists(ctx context.Context, node *corev1.Node) (bool, error) {
	const op = "hcloud/instancesv2.InstanceExists"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	hcloudServer, bmServer, _, err := i.lookupServer(ctx, node)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return hcloudServer != nil || bmServer != nil, nil
}

func (i *instances) InstanceShutdown(ctx context.Context, node *corev1.Node) (bool, error) {
	const op = "hcloud/instancesv2.InstanceShutdown"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	hcloudServer, _, isHCloudServer, err := i.lookupServer(ctx, node)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		if hcloudServer == nil {
			return false, fmt.Errorf("failed to find server status: no matching hcloud server found for node '%s': %w", node.Name, errServerNotFound)
		}
		return hcloudServer.Status == hcloud.ServerStatusOff, nil
	}

	// Robot does not support shutdowns
	return false, nil
}

func (i *instances) InstanceMetadata(ctx context.Context, node *corev1.Node) (*cloudprovider.InstanceMetadata, error) {
	const op = "hcloud/instancesv2.InstanceMetadata"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	hcloudServer, bmServer, isHCloudServer, err := i.lookupServer(ctx, node)
	if err != nil {
		return nil, err
	}

	if isHCloudServer {
		if hcloudServer == nil {
			return nil, fmt.Errorf("failed to get instance metadata: no matching hcloud server found for node '%s': %w",
				node.Name, errServerNotFound)
		}
		return &cloudprovider.InstanceMetadata{
			ProviderID:    serverIDToProviderIDHCloud(hcloudServer.ID),
			InstanceType:  hcloudServer.ServerType.Name,
			NodeAddresses: hcloudNodeAddresses(i.addressFamily, i.networkID, hcloudServer),
			Zone:          hcloudServer.Datacenter.Name,
			Region:        hcloudServer.Datacenter.Location.Name,
		}, nil
	}
	if bmServer == nil {
		return nil, fmt.Errorf("failed to get instance metadata: no matching bare metal server found for node '%s': %w",
			node.Name, errServerNotFound)
	}
	return &cloudprovider.InstanceMetadata{
		ProviderID:    serverIDToProviderIDRobot(bmServer.ServerNumber),
		InstanceType:  getInstanceTypeOfRobotServer(bmServer),
		NodeAddresses: robotNodeAddresses(i.addressFamily, bmServer),
		Zone:          getZoneOfRobotServer(bmServer),
		Region:        getRegionOfRobotServer(bmServer),
	}, nil
}

func hcloudNodeAddresses(addressFamily addressFamily, networkID int64, server *hcloud.Server) []corev1.NodeAddress {
	var addresses []corev1.NodeAddress
	addresses = append(
		addresses,
		corev1.NodeAddress{Type: corev1.NodeHostName, Address: server.Name},
	)

	if addressFamily == AddressFamilyIPv4 || addressFamily == AddressFamilyDualStack {
		if !server.PublicNet.IPv4.IsUnspecified() {
			addresses = append(
				addresses,
				corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: server.PublicNet.IPv4.IP.String()},
			)
		}
	}

	if addressFamily == AddressFamilyIPv6 || addressFamily == AddressFamilyDualStack {
		if !server.PublicNet.IPv6.IsUnspecified() {
			// For a given IPv6 network of 2001:db8:1234::/64, the instance address is 2001:db8:1234::1
			hostAddress := server.PublicNet.IPv6.IP
			hostAddress[len(hostAddress)-1] |= 0x01

			addresses = append(
				addresses,
				corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: hostAddress.String()},
			)
		}
	}

	// Add private IP from network if network is specified
	if networkID > 0 {
		for _, privateNet := range server.PrivateNet {
			if privateNet.Network.ID == networkID {
				addresses = append(
					addresses,
					corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: privateNet.IP.String()},
				)
			}
		}
	}
	return addresses
}

func robotNodeAddresses(addressFamily addressFamily, server *models.Server) []corev1.NodeAddress {
	var addresses []corev1.NodeAddress
	addresses = append(
		addresses,
		corev1.NodeAddress{Type: corev1.NodeHostName, Address: server.Name},
	)

	if addressFamily == AddressFamilyIPv6 || addressFamily == AddressFamilyDualStack {
		// For a given IPv6 network of 2a01:f48:111:4221::, the instance address is 2a01:f48:111:4221::1
		hostAddress := server.ServerIPv6Net
		hostAddress += "1"

		addresses = append(
			addresses,
			corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: hostAddress},
		)
	}

	if addressFamily == AddressFamilyIPv4 || addressFamily == AddressFamilyDualStack {
		addresses = append(
			addresses,
			corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: server.ServerIP},
		)
	}

	return addresses
}
