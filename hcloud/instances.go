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
	"os"
	"strconv"

	"github.com/hetznercloud/hcloud-go/hcloud"
	hrobot "github.com/syself/hrobot-go"
	"github.com/syself/hrobot-go/models"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	robotClient   hrobot.RobotClient
	addressFamily addressFamily
}

func newInstances(client *hcloud.Client, robotClient hrobot.RobotClient, addressFamily addressFamily) *instances {
	return &instances{client, robotClient, addressFamily}
}

func (i *instances) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	const op = "hcloud/instances.NodeAddressesByProviderID"

	id, isHCloudServer, err := providerIDToServerID(providerID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, err := getHCloudServerByID(ctx, i.client, id)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		return i.hcloudNodeAddresses(ctx, server), nil
	}

	server, err := getRobotServerByID(ctx, i.robotClient, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return i.robotNodeAddresses(ctx, server), nil
}

func (i *instances) NodeAddresses(ctx context.Context, nodeName types.NodeName) ([]v1.NodeAddress, error) {
	const op = "hcloud/instances.NodeAddresses"

	isHCloudServer, err := isHCloudServerByName(string(nodeName))
	if err != nil {
		return nil, err
	}

	if isHCloudServer {
		server, err := getHCloudServerByName(ctx, i.client, string(nodeName))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		return i.hcloudNodeAddresses(ctx, server), nil
	}

	server, err := getRobotServerByName(ctx, i.robotClient, string(nodeName))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return i.robotNodeAddresses(ctx, server), nil
}

func (i *instances) ExternalID(ctx context.Context, nodeName types.NodeName) (string, error) {
	const op = "hcloud/instances.ExternalID"

	id, err := i.InstanceID(ctx, nodeName)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (i *instances) InstanceID(ctx context.Context, nodeName types.NodeName) (string, error) {
	const op = "hcloud/instances.InstanceID"

	isHCloudServer, err := isHCloudServerByName(string(nodeName))
	if err != nil {
		return "", err
	}

	if isHCloudServer {
		server, err := getHCloudServerByName(ctx, i.client, string(nodeName))
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
		return strconv.Itoa(server.ID), nil
	}

	server, err := getRobotServerByName(ctx, i.robotClient, string(nodeName))
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return strconv.Itoa(server.ServerNumber), nil
}

func (i *instances) InstanceType(ctx context.Context, nodeName types.NodeName) (string, error) {
	const op = "hcloud/instances.InstanceType"

	isHCloudServer, err := isHCloudServerByName(string(nodeName))
	if err != nil {
		return "", err
	}

	if isHCloudServer {
		server, err := getHCloudServerByName(ctx, i.client, string(nodeName))
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
		return server.ServerType.Name, nil
	}

	server, err := getRobotServerByName(ctx, i.robotClient, string(nodeName))
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return server.Product, nil
}

func (i *instances) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	const op = "hcloud/instances.InstanceTypeByProviderID"

	id, isHCloudServer, err := providerIDToServerID(providerID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, err := getHCloudServerByID(ctx, i.client, id)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
		return server.ServerType.Name, nil
	}

	server, err := getRobotServerByID(ctx, i.robotClient, id)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return server.Product, nil
}

func (i *instances) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

func (i *instances) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

func (i instances) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {
	const op = "hcloud/instances.InstanceExistsByProviderID"

	id, isHCloudServer, err := providerIDToServerID(providerID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, _, err := i.client.Server.GetByID(ctx, id)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		return server != nil, nil
	}

	if i.robotClient == nil {
		return false, errMissingRobotCredentials
	}

	server, err := i.robotClient.ServerGet(id)
	if err != nil {
		if models.IsError(err, models.ErrorCodeNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return isRobotServerInCluster(server.Name), nil
}

func (i instances) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	const op = "hcloud/instances.InstanceShutdownByProviderID"

	id, isHCloudServer, err := providerIDToServerID(providerID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, _, err := i.client.Server.GetByID(ctx, id)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		return server != nil && server.Status == hcloud.ServerStatusOff, nil
	}

	// Robot does not support shutdowns
	return false, nil
}

func (i *instances) hcloudNodeAddresses(ctx context.Context, server *hcloud.Server) []v1.NodeAddress {
	var addresses []v1.NodeAddress
	addresses = append(
		addresses,
		v1.NodeAddress{Type: v1.NodeHostName, Address: server.Name},
	)

	if i.addressFamily == AddressFamilyIPv6 || i.addressFamily == AddressFamilyDualStack {
		// For a given IPv6 network of 2001:db8:1234::/64, the instance address is 2001:db8:1234::1
		host_address := server.PublicNet.IPv6.IP
		host_address[len(host_address)-1] |= 0x01

		addresses = append(
			addresses,
			v1.NodeAddress{Type: v1.NodeExternalIP, Address: host_address.String()},
		)
	}

	if i.addressFamily == AddressFamilyIPv4 || i.addressFamily == AddressFamilyDualStack {
		addresses = append(
			addresses,
			v1.NodeAddress{Type: v1.NodeExternalIP, Address: server.PublicNet.IPv4.IP.String()},
		)
	}

	n := os.Getenv(hcloudNetworkENVVar)
	if len(n) > 0 {
		network, _, _ := i.client.Network.Get(ctx, n)
		if network != nil {
			for _, privateNet := range server.PrivateNet {
				if privateNet.Network.ID == network.ID {
					addresses = append(
						addresses,
						v1.NodeAddress{Type: v1.NodeInternalIP, Address: privateNet.IP.String()},
					)
				}
			}

		}
	}
	return addresses
}

func (i *instances) robotNodeAddresses(ctx context.Context, server *models.Server) []v1.NodeAddress {
	var addresses []v1.NodeAddress
	addresses = append(
		addresses,
		v1.NodeAddress{Type: v1.NodeHostName, Address: server.Name},
	)

	if i.addressFamily == AddressFamilyIPv6 || i.addressFamily == AddressFamilyDualStack {
		// For a given IPv6 network of 2a01:f48:111:4221::, the instance address is 2a01:f48:111:4221::1
		host_address := server.ServerIPv6Net
		host_address = host_address + "1"

		addresses = append(
			addresses,
			v1.NodeAddress{Type: v1.NodeExternalIP, Address: host_address},
		)
	}

	if i.addressFamily == AddressFamilyIPv4 || i.addressFamily == AddressFamilyDualStack {
		addresses = append(
			addresses,
			v1.NodeAddress{Type: v1.NodeExternalIP, Address: server.ServerIP},
		)
	}

	return addresses
}
