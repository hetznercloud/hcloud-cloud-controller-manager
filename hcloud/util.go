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
	"time"

	"github.com/syself/hetzner-cloud-controller-manager/internal/metrics"
	hrobot "github.com/syself/hrobot-go"
	"github.com/syself/hrobot-go/models"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

const rateLimitWaitingTime = 5 * time.Minute

type rateLimit struct {
	exceeded    bool
	lastChecked time.Time
}

var (
	robotRateLimitServerGetList rateLimit
	robotRateLimitServerGet     rateLimit
)

func (rl *rateLimit) set() {
	rl.exceeded = true
	rl.lastChecked = time.Now()
}

func (rl *rateLimit) isExceeded() bool {
	if !rl.exceeded {
		return false
	}

	if time.Now().Before(rl.lastChecked.Add(rateLimitWaitingTime)) {
		return true
	} else {
		// Waiting time is over. Should try again
		rl.exceeded = false
		rl.lastChecked = time.Time{}
		return false
	}
}

func (rl *rateLimit) timeOfNextPossibleAPICall() time.Time {
	emptyTime := time.Time{}
	if rl.lastChecked == emptyTime {
		return emptyTime
	}
	return rl.lastChecked.Add(rateLimitWaitingTime)
}

func getHCloudServerByName(ctx context.Context, c *hcloud.Client, name string) (*hcloud.Server, error) {
	const op = "hcloud/getServerByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, _, err := c.Server.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return server, nil
}

func getHCloudServerByID(ctx context.Context, c *hcloud.Client, id int) (*hcloud.Server, error) {
	const op = "hcloud/getServerByID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, _, err := c.Server.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return server, nil
}

func getRobotServerByName(c hrobot.RobotClient, name string) (server *models.Server, err error) {
	const op = "robot/getServerByName"

	if c == nil {
		return nil, errMissingRobotCredentials
	}

	// Check for rate limit
	if robotRateLimitServerGetList.isExceeded() {
		return nil, fmt.Errorf("%s: rate limit exceeded. Next try at %v", op, robotRateLimitServerGetList.timeOfNextPossibleAPICall().String())
	}

	serverList, err := c.ServerGetList()
	if err != nil {
		if models.IsError(err, models.ErrorCodeRateLimitExceeded) {
			robotRateLimitServerGetList.set()
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i, s := range serverList {
		if s.Name == name {
			server = &serverList[i]
		}
	}

	return server, nil
}

func getRobotServerByID(c hrobot.RobotClient, id int) (*models.Server, error) {
	const op = "robot/getServerByID"

	if c == nil {
		return nil, errMissingRobotCredentials
	}

	// Check for rate limit
	if robotRateLimitServerGet.isExceeded() {
		return nil, fmt.Errorf("%s: rate limit exceeded. Next try at %v", op, robotRateLimitServerGet.timeOfNextPossibleAPICall().String())
	}

	server, err := c.ServerGet(id)
	if err != nil && !models.IsError(err, models.ErrorCodeServerNotFound) {
		if models.IsError(err, models.ErrorCodeRateLimitExceeded) {
			robotRateLimitServerGet.set()
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return server, nil
}

func providerIDToServerID(providerID string) (id int, isHCloudServer bool, err error) {
	const op = "hcloud/providerIDToServerID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	providerPrefixHCloud := providerName + "://"
	providerPrefixRobot := providerName + "://" + hostNamePrefixRobot

	if !strings.HasPrefix(providerID, providerPrefixHCloud) && !strings.HasPrefix(providerID, providerPrefixRobot) {
		klog.Infof("%s: make sure your cluster configured for an external cloud provider", op)
		return 0, false, fmt.Errorf("%s: missing prefix %s or %s. %s", providerPrefixHCloud, providerPrefixRobot, op, providerID)
	}

	isHCloudServer = true
	idString := providerID
	if strings.HasPrefix(providerID, providerPrefixRobot) {
		isHCloudServer = false
		idString = strings.ReplaceAll(idString, providerPrefixRobot, "")
	} else {
		idString = strings.ReplaceAll(providerID, providerPrefixHCloud, "")
	}

	if idString == "" {
		return 0, false, fmt.Errorf("%s: missing serverID: %s", op, providerID)
	}

	id, err = strconv.Atoi(idString)
	if err != nil {
		return 0, false, fmt.Errorf("%s: invalid serverID: %s", op, providerID)
	}
	return id, isHCloudServer, nil
}

func isHCloudServerByName(name string) bool {
	return !strings.HasPrefix(name, hostNamePrefixRobot)
}

func serverIDToProviderIDRobot(serverID int) string {
	return fmt.Sprintf("%s://%s%d", providerName, hostNamePrefixRobot, serverID)
}

func serverIDToProviderIDHCloud(serverID int) string {
	return fmt.Sprintf("%s://%d", providerName, serverID)
}

func getInstanceTypeOfRobotServer(bmServer *models.Server) string {
	if bmServer == nil {
		panic("getInstanceTypeOfRobotServer called with nil server")
	}
	return strings.ReplaceAll(bmServer.Product, " ", "-")
}

func getZoneOfRobotServer(bmServer *models.Server) string {
	if bmServer == nil {
		panic("getZoneOfRobotServer called with nil server")
	}
	return strings.ToLower(bmServer.Dc[:4])
}

func getRegionOfRobotServer(bmServer *models.Server) string {
	if bmServer == nil {
		panic("getZoneOfRobotServer called with nil server")
	}
	zoneToRegionMap := map[string]string{
		"nbg1": "eu-central",
		"fsn1": "eu-central",
		"hel1": "eu-central",
		"ash":  "us-east",
	}
	zone := getZoneOfRobotServer(bmServer)
	region, found := zoneToRegionMap[zone]
	if !found {
		panic("zoneToRegionMap: unknown zone")
	}
	return region
}
