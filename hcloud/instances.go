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
	"errors"
	"fmt"
	"net"

	hrobotmodels "github.com/syself/hrobot-go/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/providerid"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/robot"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	ProvidedBy              = "instance.hetzner.cloud/provided-by"
	MisconfiguredInternalIP = "MisconfiguredInternalIP"
)

type instances struct {
	client      *hcloud.Client
	robotClient robot.Client
	recorder    record.EventRecorder
	networkID   int64
	cfg         config.HCCMConfiguration
}

var (
	errServerNotFound     = errors.New("server not found")
	errMissingRobotClient = errors.New("no robot client configured, make sure to enable Robot support in the configuration")
)

func newInstances(
	client *hcloud.Client,
	robotClient robot.Client,
	recorder record.EventRecorder,
	networkID int64,
	cfg config.HCCMConfiguration,
) *instances {
	return &instances{
		client,
		robotClient,
		recorder,
		networkID,
		cfg,
	}
}

// lookupServer attempts to locate the corresponding [*hcloud.Server] or [*hrobotmodels.Server] for a given [*corev1.Node].
// It returns an error if the Node has an invalid provider ID or if API requests failed.
// It can return nil server if neither the ProviderID nor the Name matches an existing server.
func (i *instances) lookupServer(
	ctx context.Context,
	node *corev1.Node,
) (genericServer, error) {
	if node.Spec.ProviderID != "" {
		var serverID int64
		serverID, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)

		if err != nil {
			return nil, fmt.Errorf("failed to convert provider id to server id: %w", err)
		}

		if isCloudServer {
			server, err := getCloudServerByID(ctx, i.client, serverID)
			if err != nil {
				return nil, fmt.Errorf("failed to get hcloud server \"%d\": %w", serverID, err)
			}

			if server == nil {
				return nil, nil
			}

			return hcloudServer{server}, nil
		}

		if i.robotClient == nil {
			return nil, errMissingRobotClient
		}
		server, err := getRobotServerByID(i, int(serverID), node)
		if err != nil {
			return nil, fmt.Errorf("failed to get robot server \"%d\": %w", serverID, err)
		}

		if server == nil {
			return nil, nil
		}

		return robotServer{server, i.robotClient, i.recorder}, nil
	}

	// If the node has no provider ID we try to find the server by name from
	// both sources. In case we find two servers, we return an error.
	cloudServer, err := getCloudServerByName(ctx, i.client, node.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get hcloud server %q: %w", node.Name, err)
	}

	var hrobotServer *hrobotmodels.Server
	if i.robotClient != nil {
		hrobotServer, err = getRobotServerByName(i.robotClient, node)
		if err != nil {
			return nil, fmt.Errorf("failed to get robot server %q: %w", node.Name, err)
		}
	}

	if cloudServer != nil && hrobotServer != nil {
		i.recorder.Eventf(
			node,
			corev1.EventTypeWarning,
			"InstanceLookupFailed",
			"Node %s could not be uniquely associated with a Cloud or Robot server, as a server with this name exists in both APIs",
			node.Name,
		)
		return nil, fmt.Errorf("found both a cloud & robot server for name %q", node.Name)
	}

	switch {
	case cloudServer != nil:
		return hcloudServer{cloudServer}, nil
	case hrobotServer != nil:
		return robotServer{hrobotServer, i.robotClient, i.recorder}, nil
	default:
		// Both nil
		return nil, nil
	}
}

func (i *instances) InstanceExists(ctx context.Context, node *corev1.Node) (bool, error) {
	const op = "hcloud/instancesv2.InstanceExists"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupServer(ctx, node)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return server != nil, nil
}

func (i *instances) InstanceShutdown(ctx context.Context, node *corev1.Node) (bool, error) {
	const op = "hcloud/instancesv2.InstanceShutdown"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupServer(ctx, node)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if server == nil {
		return false, fmt.Errorf(
			"%s: failed to get instance metadata: no matching server found for node '%s': %w",
			op, node.Name, errServerNotFound)
	}

	isShutdown, err := server.IsShutdown()
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isShutdown, nil
}

func (i *instances) InstanceMetadata(ctx context.Context, node *corev1.Node) (*cloudprovider.InstanceMetadata, error) {
	const op = "hcloud/instancesv2.InstanceMetadata"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupServer(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if server == nil {
		return nil, fmt.Errorf(
			"%s: failed to get instance metadata: no matching server found for node '%s': %w",
			op, node.Name, errServerNotFound)
	}

	metadata, err := server.Metadata(i.networkID, node, i.cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return metadata, nil
}

func hcloudNodeAddresses(
	networkID int64,
	server *hcloud.Server,
	cfg config.HCCMConfiguration,
) []corev1.NodeAddress {
	var addresses []corev1.NodeAddress
	addresses = append(
		addresses,
		corev1.NodeAddress{Type: corev1.NodeHostName, Address: server.Name},
	)

	dualStack := cfg.Instance.AddressFamily == config.AddressFamilyDualStack
	ipv4 := cfg.Instance.AddressFamily == config.AddressFamilyIPv4 || dualStack
	ipv6 := cfg.Instance.AddressFamily == config.AddressFamilyIPv6 || dualStack

	if ipv4 && !server.PublicNet.IPv4.IsUnspecified() {
		addresses = append(
			addresses,
			corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: server.PublicNet.IPv4.IP.String()},
		)
	}

	if ipv6 && !server.PublicNet.IPv6.IsUnspecified() {
		// For a given IPv6 network of 2001:db8:1234::/64, the instance address is 2001:db8:1234::1
		hostAddress := server.PublicNet.IPv6.IP
		hostAddress[len(hostAddress)-1] |= 0x01

		addresses = append(
			addresses,
			corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: hostAddress.String()},
		)
	}

	// Add private IP from network if network is specified
	if networkID > 0 {
		for _, privateNet := range server.PrivateNet {
			if privateNet.Network.ID == networkID {
				addresses = append(
					addresses,
					corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: privateNet.IP.String()},
				)
			}
		}
	}

	return addresses
}

func robotNodeAddresses(
	server *hrobotmodels.Server,
	node *corev1.Node,
	cfg config.HCCMConfiguration,
	recorder record.EventRecorder,
) []corev1.NodeAddress {
	var addresses []corev1.NodeAddress
	addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeHostName, Address: server.Name})

	dualStack := cfg.Instance.AddressFamily == config.AddressFamilyDualStack
	ipv4 := cfg.Instance.AddressFamily == config.AddressFamilyIPv4 || dualStack
	ipv6 := cfg.Instance.AddressFamily == config.AddressFamilyIPv6 || dualStack

	if ipv6 {
		// For a given IPv6 network of 2a01:f48:111:4221::, the instance address is 2a01:f48:111:4221::1
		hostAddress := server.ServerIPv6Net + "1"
		addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: hostAddress})
	}

	if ipv4 {
		addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeExternalIP, Address: server.ServerIP})
	}

	if cfg.Robot.ForwardInternalIPs {
	OUTER:
		for _, currentAddress := range node.Status.Addresses {
			if currentAddress.Type != corev1.NodeInternalIP {
				continue
			}

			ip := net.ParseIP(currentAddress.Address)
			isIPv4 := ip.To4() != nil

			var warnMsg string
			if isIPv4 && ipv6 && !dualStack {
				warnMsg = fmt.Sprintf(
					"Configured InternalIP is IPv4 even though IPv6 only is configured. As a result, %s is not added as an InternalIP",
					currentAddress.Address,
				)
			} else if !isIPv4 && ipv4 && !dualStack {
				warnMsg = fmt.Sprintf(
					"Configured InternalIP is IPv6 even though IPv4 only is configured. As a result, %s is not added as an InternalIP",
					currentAddress.Address,
				)
			}

			if warnMsg != "" {
				recorder.Event(node, corev1.EventTypeWarning, MisconfiguredInternalIP, warnMsg)
				klog.Warning(warnMsg)
				continue
			}

			for _, address := range addresses {
				if currentAddress.Address == address.Address {
					warnMsg := fmt.Sprintf(
						"Configured InternalIP already exists as an ExternalIP. As a result, %s is not added as an InternalIP",
						currentAddress.Address,
					)
					recorder.Event(node, corev1.EventTypeWarning, MisconfiguredInternalIP, warnMsg)
					klog.Warning(warnMsg)
					continue OUTER
				}
			}

			addresses = append(addresses, currentAddress)
		}
	}

	return addresses
}

type genericServer interface {
	IsShutdown() (bool, error)
	Metadata(
		networkID int64,
		node *corev1.Node,
		cfg config.HCCMConfiguration,
	) (*cloudprovider.InstanceMetadata, error)
}

type hcloudServer struct {
	*hcloud.Server
}

func (s hcloudServer) IsShutdown() (bool, error) {
	return s.Status == hcloud.ServerStatusOff, nil
}

func (s hcloudServer) Metadata(networkID int64, _ *corev1.Node, cfg config.HCCMConfiguration) (*cloudprovider.InstanceMetadata, error) {
	return &cloudprovider.InstanceMetadata{
		ProviderID:    providerid.FromCloudServerID(s.ID),
		InstanceType:  s.ServerType.Name,
		NodeAddresses: hcloudNodeAddresses(networkID, s.Server, cfg),
		Zone:          s.Datacenter.Name,
		Region:        s.Datacenter.Location.Name,
		AdditionalLabels: map[string]string{
			ProvidedBy: "cloud",
		},
	}, nil
}

type robotServer struct {
	*hrobotmodels.Server
	robotClient robot.Client
	recorder    record.EventRecorder
}

func (s robotServer) IsShutdown() (bool, error) {
	resetStatus, err := s.robotClient.ResetGet(s.ServerNumber)
	if err != nil {
		return false, err
	}

	// OperationStatus is not supported for server models using the tower case, in that case the value is "not supported"
	// When the server is powered off, the OperatingStatus is "shut off"
	return resetStatus.OperatingStatus == "shut off", nil
}

func (s robotServer) Metadata(_ int64, node *corev1.Node, cfg config.HCCMConfiguration) (*cloudprovider.InstanceMetadata, error) {
	return &cloudprovider.InstanceMetadata{
		ProviderID:    providerid.FromRobotServerNumber(s.ServerNumber),
		InstanceType:  getInstanceTypeOfRobotServer(s.Server),
		NodeAddresses: robotNodeAddresses(s.Server, node, cfg, s.recorder),
		Zone:          getZoneOfRobotServer(s.Server),
		Region:        getRegionOfRobotServer(s.Server),
		AdditionalLabels: map[string]string{
			ProvidedBy: "robot",
		},
	}, nil
}
