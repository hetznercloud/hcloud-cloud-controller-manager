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
	"k8s.io/kubernetes/pkg/cloudprovider"
	"strconv"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type instances struct {
	client *hcloud.Client
}

func newInstances(client *hcloud.Client) *instances {
	return &instances{client}
}

func (i *instances) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	id, err := providerIDToServerID(providerID)
	if err != nil {
		return nil, err
	}

	server, err := getServerByID(ctx, i.client, id)
	if err != nil {
		return nil, err
	}
	return nodeAddresses(server), nil
}

func (i *instances) NodeAddresses(ctx context.Context, nodeName types.NodeName) ([]v1.NodeAddress, error) {
	server, err := getServerByName(ctx, i.client, string(nodeName))
	if err != nil {
		return nil, err
	}
	return nodeAddresses(server), nil
}

func (i *instances) ExternalID(ctx context.Context, nodeName types.NodeName) (string, error) {
	return i.InstanceID(ctx, nodeName)
}

func (i *instances) InstanceID(ctx context.Context, nodeName types.NodeName) (string, error) {
	server, err := getServerByName(ctx, i.client, string(nodeName))
	if err != nil {
		return "", err
	}
	return strconv.Itoa(server.ID), nil
}

func (i *instances) InstanceType(ctx context.Context, nodeName types.NodeName) (string, error) {
	server, err := getServerByName(ctx, i.client, string(nodeName))
	if err != nil {
		return "", err
	}
	return server.ServerType.Name, nil
}

func (i *instances) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	id, err := providerIDToServerID(providerID)
	if err != nil {
		return "", err
	}

	server, err := getServerByID(ctx, i.client, id)
	if err != nil {
		return "", err
	}
	return server.ServerType.Name, nil
}

func (i *instances) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

func (i *instances) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

func (i instances) InstanceExistsByProviderID(ctx context.Context, providerID string) (exists bool, err error) {
	var id int
	id, err = providerIDToServerID(providerID)
	if err != nil {
		return
	}

	var server *hcloud.Server
	server, _, err = i.client.Server.GetByID(ctx, id)
	if err != nil {
		return
	}
	exists = server != nil
	return
}

func (i instances) InstanceShutdownByProviderID(ctx context.Context, providerID string) (isOff bool, err error) {
	var id int
	id, err = providerIDToServerID(providerID)
	if err != nil {
		return
	}

	var server *hcloud.Server
	server, _, err = i.client.Server.GetByID(ctx, id)
	if err != nil {
		return
	}
	isOff = server != nil && server.Status == hcloud.ServerStatusOff
	return
}

func nodeAddresses(server *hcloud.Server) []v1.NodeAddress {
	var addresses []v1.NodeAddress
	addresses = append(
		addresses,
		v1.NodeAddress{Type: v1.NodeHostName, Address: server.Name},
		v1.NodeAddress{Type: v1.NodeExternalIP, Address: server.PublicNet.IPv4.IP.String()},
	)
	return addresses
}
