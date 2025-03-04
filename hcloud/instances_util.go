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
	"regexp"
	"strings"

	hrobotmodels "github.com/syself/hrobot-go/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/robot"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type MockEventRecorder struct{}

func (er *MockEventRecorder) Event(_ runtime.Object, _, _, _ string) {
}

func (er *MockEventRecorder) Eventf(
	_ runtime.Object,
	_, _, _ string,
	_ ...interface{},
) {
}

func (er *MockEventRecorder) AnnotatedEventf(
	_ runtime.Object,
	_ map[string]string,
	_, _, _ string,
	_ ...interface{},
) {
}

func getCloudServerByName(ctx context.Context, c *hcloud.Client, name string) (*hcloud.Server, error) {
	const op = "hcloud/getCloudServerByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, _, err := c.Server.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return server, nil
}

func getCloudServerByID(ctx context.Context, c *hcloud.Client, id int64) (*hcloud.Server, error) {
	const op = "hcloud/getCloudServerByID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, _, err := c.Server.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return server, nil
}

func getRobotServerByName(c robot.Client, node *corev1.Node) (server *hrobotmodels.Server, err error) {
	const op = "hcloud/getRobotServerByName"

	if c == nil {
		return nil, errMissingRobotClient
	}

	serverList, err := c.ServerGetList()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i, s := range serverList {
		if s.Name == node.Name {
			server = &serverList[i]
		}
	}

	return server, nil
}

func getRobotServerByID(i *instances, id int, node *corev1.Node) (*hrobotmodels.Server, error) {
	const op = "hcloud/getRobotServerByID"

	if i.robotClient == nil {
		return nil, errMissingRobotClient
	}

	server, err := i.robotClient.ServerGet(id)
	if err != nil && !hrobotmodels.IsError(err, hrobotmodels.ErrorCodeServerNotFound) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if server == nil {
		return nil, nil
	}

	// check whether name matches - otherwise this server does not belong to the respective node anymore
	if server.Name != node.Name {
		i.recorder.Eventf(
			node,
			corev1.EventTypeWarning,
			"PossibleNodeDeletion",
			"Might be deleted by node-lifecycle-manager due to name mismatch; Node name %q differs from Robot name %q",
			node.ObjectMeta.Name,
			server.Name,
		)
		return nil, nil
	}

	// return nil, nil if server could not be found
	return server, nil
}

func getInstanceTypeOfRobotServer(server *hrobotmodels.Server) string {
	if server == nil {
		panic("getInstanceTypeOfRobotServer called with nil server")
	}
	productName := strings.ReplaceAll(server.Product, " ", "-")
	// Removes all characters that are invalid for a Kubernetes label
	return regexp.MustCompile(`[^a-zA-Z0-9_.-]+`).ReplaceAllString(productName, "")
}

func getZoneOfRobotServer(server *hrobotmodels.Server) string {
	return strings.ToLower(server.Dc)
}

func getRegionOfRobotServer(server *hrobotmodels.Server) string {
	zone := getZoneOfRobotServer(server)
	// zone is a Hetzner DC, e.g. "hel1-dc2"
	// the cloud location is equal to the first part of the zone, e.g. "hel1" and that is was has historically been used in the Region label.
	return strings.Split(zone, "-")[0]
}
