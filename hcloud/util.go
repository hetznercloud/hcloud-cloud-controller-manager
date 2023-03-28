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
	"fmt"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
)

func providerIDToServerID(providerID string) (int, error) {
	const op = "hcloud/providerIDToServerID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	providerPrefix := providerName + "://"
	if !strings.HasPrefix(providerID, providerPrefix) {
		klog.Infof("%s: make sure your cluster configured for an external cloud provider", op)
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

func serverIDToProviderID(serverID int) string {
	return fmt.Sprintf("%s://%d", providerName, serverID)
}
