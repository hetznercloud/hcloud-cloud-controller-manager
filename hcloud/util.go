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
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-go/hcloud"
	hrobot "github.com/syself/hrobot-go"
	"github.com/syself/hrobot-go/models"
	cloudprovider "k8s.io/cloud-provider"
)

func getHCloudServerByName(ctx context.Context, c *hcloud.Client, name string) (*hcloud.Server, error) {
	const op = "hcloud/getServerByName"

	server, _, err := c.Server.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if server == nil {
		klog.Infof("%s: server with name %s not found, are the name in the Hetzner Cloud and the node name identical?", op, name)
		return nil, cloudprovider.InstanceNotFound
	}
	return server, nil
}

func getHCloudServerByID(ctx context.Context, c *hcloud.Client, id int) (*hcloud.Server, error) {
	const op = "hcloud/getServerByName"

	server, _, err := c.Server.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if server == nil {
		return nil, cloudprovider.InstanceNotFound
	}
	return server, nil
}

func getRobotServerByName(ctx context.Context, c hrobot.RobotClient, name string) (server *models.Server, err error) {
	const op = "robot/getServerByName"

	if c == nil {
		return nil, errMissingRobotCredentials
	}

	serverList, err := c.ServerGetList()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i, s := range serverList {
		if s.Name == name {
			server = &serverList[i]
		}
	}

	if server == nil {
		klog.Infof("%s: server with name %s not found, are the name in Hetzner Robot and the node name identical?", op, name)
		return nil, cloudprovider.InstanceNotFound
	}
	return server, nil
}

func getRobotServerByID(ctx context.Context, c hrobot.RobotClient, id int) (*models.Server, error) {
	const op = "robot/getServerByID"

	if c == nil {
		return nil, errMissingRobotCredentials
	}

	server, err := c.ServerGet(id)
	if err != nil {
		if models.IsError(err, models.ErrorCodeNotFound) {
			return nil, cloudprovider.InstanceNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return server, nil
}

func providerIDToServerID(providerID string) (id int, isHCloudServer bool, err error) {
	const op = "hcloud/providerIDToServerID"

	providerPrefixHCloud := providerName + "://"
	providerPrefixRobot := providerNameRobot + "://"

	if !strings.HasPrefix(providerID, providerPrefixHCloud) && !strings.HasPrefix(providerID, providerPrefixRobot) {
		klog.Infof("%s: make sure your cluster configured for an external cloud provider", op)
		return 0, false, fmt.Errorf("%s: missing prefix %s or %s. %s", providerPrefixHCloud, providerPrefixRobot, op, providerID)
	}

	if strings.HasPrefix(providerID, providerPrefixHCloud) {
		isHCloudServer = true
	}

	idString := strings.ReplaceAll(providerID, providerPrefixHCloud, "")
	idString = strings.ReplaceAll(idString, providerPrefixRobot, "")
	if idString == "" {
		return 0, false, fmt.Errorf("%s: missing serverID: %s", op, providerID)
	}

	id, err = strconv.Atoi(idString)
	if err != nil {
		return 0, false, fmt.Errorf("%s: invalid serverID: %s", op, providerID)
	}
	return id, isHCloudServer, nil
}

func isHCloudServerByName(name string) (bool, error) {
	if strings.HasPrefix(name, hostNamePrefixHCloud) {
		return true, nil
	}
	if strings.HasPrefix(name, hostNamePrefixRobot) {
		return false, nil
	}
	return false, fmt.Errorf("server name has no host name prefix. Should be either %s or %s",
		hostNamePrefixHCloud, hostNamePrefixRobot)
}

func isRobotServerInCluster(name string) bool {
	return strings.HasPrefix(name, hostNamePrefixRobot)
}
