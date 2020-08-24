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

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

type zones struct {
	client   *hcloud.Client
	nodeName string // name of the node the programm is running on
}

func newZones(client *hcloud.Client, nodeName string) *zones {
	return &zones{client, nodeName}
}

func (z zones) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	const op = "hcloud/zones.GetZone"

	server, err := getServerByName(ctx, z.client, z.nodeName)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}
	return zoneFromServer(server), nil
}

func (z zones) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	const op = "hcloud/zones.GetZoneByProviderID"

	id, err := providerIDToServerID(providerID)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	server, err := getServerByID(ctx, z.client, id)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	return zoneFromServer(server), nil
}

func (z zones) GetZoneByNodeName(ctx context.Context, nodeName types.NodeName) (cloudprovider.Zone, error) {
	const op = "hcloud/zones.GetZoneByNodeName"

	server, err := getServerByName(ctx, z.client, string(nodeName))
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	return zoneFromServer(server), nil
}

func zoneFromServer(server *hcloud.Server) cloudprovider.Zone {
	return cloudprovider.Zone{
		Region:        server.Datacenter.Location.Name,
		FailureDomain: server.Datacenter.Name,
	}
}
