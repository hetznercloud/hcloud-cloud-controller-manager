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

	"github.com/hetznercloud/hcloud-go/hcloud"
	cloudprovider "k8s.io/cloud-provider"
)

func getServerByName(ctx context.Context, c *hcloud.Client, name string) (*hcloud.Server, error) {
	const op = "hcloud/getServerByName"

	server, _, err := c.Server.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if server == nil {
		return nil, cloudprovider.InstanceNotFound
	}
	return server, nil
}

func getServerByID(ctx context.Context, c *hcloud.Client, id int) (*hcloud.Server, error) {
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

func providerIDToServerID(providerID string) (int, error) {
	const op = "hcloud/providerIDToServerID"

	providerPrefix := providerName + "://"
	if !strings.HasPrefix(providerID, providerPrefix) {
		return 0, fmt.Errorf("%s: missing prefix hcloud://: %s", op, providerID)
	}

	idString := strings.ReplaceAll(providerID, providerPrefix, "")
	if idString == "" {
		return 0, fmt.Errorf("%s: missing serverID: %s", op, providerID)
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid serverID: %s", op, providerID)
	}
	return id, nil
}
