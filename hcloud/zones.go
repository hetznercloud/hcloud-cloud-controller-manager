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
	"strings"

	"github.com/hetznercloud/hcloud-go/hcloud"
	hrobot "github.com/syself/hrobot-go"
	"github.com/syself/hrobot-go/models"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

type zones struct {
	client      *hcloud.Client
	robotClient hrobot.RobotClient
	nodeName    string // name of the node the programm is running on
}

func newZones(hcloudClient *hcloud.Client, robotClient hrobot.RobotClient, nodeName string) *zones {
	return &zones{client: hcloudClient, robotClient: robotClient, nodeName: nodeName}
}

func (z zones) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	const op = "hcloud/zones.GetZone"

	isHCloudServer, err := isHCloudServerByName(z.nodeName)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, err := getHCloudServerByName(ctx, z.client, z.nodeName)
		if err != nil {
			return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
		}
		return zoneFromHCloudServer(server), nil
	}

	if z.robotClient == nil {
		return cloudprovider.Zone{}, errMissingRobotCredentials
	}

	server, err := getRobotServerByName(ctx, z.robotClient, z.nodeName)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}
	return zoneFromRobotServer(server), nil
}

func (z zones) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	const op = "hcloud/zones.GetZoneByProviderID"

	id, isHCloudServer, err := providerIDToServerID(providerID)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, err := getHCloudServerByID(ctx, z.client, id)
		if err != nil {
			return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
		}

		return zoneFromHCloudServer(server), nil
	}

	if z.robotClient == nil {
		return cloudprovider.Zone{}, errMissingRobotCredentials
	}

	server, err := getRobotServerByID(ctx, z.robotClient, id)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	return zoneFromRobotServer(server), nil
}

func (z zones) GetZoneByNodeName(ctx context.Context, nodeName types.NodeName) (cloudprovider.Zone, error) {
	const op = "hcloud/zones.GetZoneByNodeName"

	isHCloudServer, err := isHCloudServerByName(string(nodeName))
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	if isHCloudServer {
		server, err := getHCloudServerByName(ctx, z.client, string(nodeName))
		if err != nil {
			return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
		}
		return zoneFromHCloudServer(server), nil
	}

	if z.robotClient == nil {
		return cloudprovider.Zone{}, errMissingRobotCredentials
	}

	server, err := getRobotServerByName(ctx, z.robotClient, string(nodeName))
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("%s: %w", op, err)
	}

	return zoneFromRobotServer(server), nil
}

func zoneFromHCloudServer(server *hcloud.Server) cloudprovider.Zone {
	region := server.Datacenter.Location.Name
	return cloudprovider.Zone{
		Region:        region,
		FailureDomain: failureDomainFromRegion(region),
	}
}

func zoneFromRobotServer(server *models.Server) cloudprovider.Zone {
	region := strings.ToLower(server.Dc[:4])
	return cloudprovider.Zone{
		Region:        region,
		FailureDomain: failureDomainFromRegion(region),
	}
}

func failureDomainFromRegion(region string) string {
	return map[string]string{
		"nbg1": "eu-central",
		"fsn1": "eu-central",
		"hel1": "eu-central",
		"ash":  "us-east",
	}[region]
}
