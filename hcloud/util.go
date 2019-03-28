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
	"k8s.io/kubernetes/pkg/cloudprovider"
)

func getServerByName(c *hcloud.Client, name string, ctx context.Context) (server *hcloud.Server, err error) {
	server, _, err = c.Server.GetByName(ctx, name)
	if err != nil {
		return
	}
	if server == nil {
		err = cloudprovider.InstanceNotFound
		return
	}
	return
}

func getServerByID(c *hcloud.Client, id int, ctx context.Context) (server *hcloud.Server, err error) {
	server, _, err = c.Server.GetByID(ctx, id)
	if err != nil {
		return
	}
	if server == nil {
		err = cloudprovider.InstanceNotFound
		return
	}
	return
}

func providerIDToServerID(providerID string) (id int, err error) {
	providerPrefix := providerName + "://"
	if !strings.HasPrefix(providerID, providerPrefix) {
		err = fmt.Errorf("providerID should start with hcloud://: %s", providerID)
		return
	}

	idString := strings.ReplaceAll(providerID, providerPrefix, "")
	if idString == "" {
		err = fmt.Errorf("missing server id in providerID: %s", providerID)
		return
	}

	id, err = strconv.Atoi(idString)
	return
}
