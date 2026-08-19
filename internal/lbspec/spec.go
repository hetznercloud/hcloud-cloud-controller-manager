// Package lbspec resolves the desired state of a Hetzner Cloud Load Balancer
// from the annotations of a Kubernetes Service and the cluster-wide
// configuration.
//
// Resolution is pure: it makes no API calls, emits no events and reads no
// clock. Anything that needs the Hetzner Cloud API - looking up a Load Balancer
// type by name, turning certificate names into IDs - is left to the caller,
// which receives the parsed intent and nothing else.
//
// Every invalid annotation is reported, not just the first one, so a user with
// two typos does not need two reconciles to learn about both.
package lbspec

import (
	"net"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const DefaultType = "lb11"

// LabelServiceUID is a label added to the Hetzner Cloud backend to uniquely
// identify a load balancer managed by Hetzner Cloud Cloud Controller Manager.
const LabelServiceUID = "hcloud-ccm/service-uid"

// Spec is the desired state of the Hetzner Cloud Load Balancer for a Service.
//
// Pointer fields distinguish "not configured anywhere" (nil) from a configured
// value, so we never ask the API to change something the user never set.
type Spec struct {
	// Name is the effective Load Balancer name, never empty.
	Name string
	// NameUnset reports that Name is derived from the Service UID because no
	// name was requested. An unrequested name is not applied to an existing
	// Load Balancer, so that one created by other means keeps its own.
	NameUnset bool
	// Labels are the labels the Load Balancer must carry.
	Labels map[string]string
	// Hostname is published as the ingress address instead of the IPs when set.
	Hostname string
	// NodeSelector restricts which Nodes become targets. Never nil.
	NodeSelector labels.Selector

	// Type is the Load Balancer type name, never empty.
	Type string
	// TypeUnset reports that Type holds [DefaultType] because nothing
	// configured it. An unconfigured type is not applied to an existing Load
	// Balancer, to avoid downgrading it.
	TypeUnset bool

	Location    string
	NetworkZone hcloud.NetworkZone

	// Algorithm is empty when unconfigured, in which case it is left alone.
	Algorithm hcloud.LoadBalancerAlgorithmType
	// PublicInterface reports whether the public interface should be enabled.
	// Nil when unconfigured, in which case it is left alone.
	PublicInterface *bool
	// IPv4RDNS and IPv6RDNS are nil when unconfigured. The empty string is a
	// valid value and resets the record.
	IPv4RDNS *string
	IPv6RDNS *string

	// PrivateIPv4 is the address the Load Balancer should have in the private
	// network. Nil when unconfigured.
	PrivateIPv4 net.IP
	// PrivateSubnetIPRange is the existing subnet to attach to. Nil when
	// unconfigured.
	PrivateSubnetIPRange *net.IPNet
	// UsePrivateIP makes server targets use their private IP.
	UsePrivateIP bool
	// PrivateIngress publishes the private IPs as ingress addresses.
	PrivateIngress bool
	// IPv6 publishes the public IPv6 address as an ingress address.
	IPv6 bool

	// Service applies to every port of the Kubernetes Service. Every HTTP and
	// health check annotation is read from the Service, so only the listen and
	// destination ports differ per port.
	Service ServiceSpec

	// ManagedCertificate is set when the Service asks for a managed
	// certificate. The certificate itself is created and looked up by the
	// caller.
	ManagedCertificate *ManagedCertificate
}

// ServiceSpec is the desired state of the services exposed by the Load
// Balancer.
type ServiceSpec struct {
	Protocol      hcloud.LoadBalancerServiceProtocol
	ProxyProtocol *bool

	// HTTP is nil unless at least one HTTP option is configured, in which case
	// no HTTP block is sent to the API at all.
	HTTP *HTTPSpec
	// HealthCheck is nil unless at least one health check option is
	// configured, in which case a TCP check against the destination port is
	// used.
	HealthCheck *HealthCheckSpec
}

// HTTPSpec holds the HTTP options of a Load Balancer service.
type HTTPSpec struct {
	CookieName     *string
	CookieLifetime *time.Duration
	TimeoutIdle    *time.Duration
	RedirectHTTP   *bool
	StickySessions *bool

	// Certificates reference certificates either by ID or by name, exactly as
	// the annotation spelled them. Resolving names to IDs needs the API and is
	// left to the caller. Empty for a managed certificate, which the caller
	// looks up by label.
	Certificates []*hcloud.Certificate
}

// HealthCheckSpec holds the health check options of a Load Balancer service.
type HealthCheckSpec struct {
	// Protocol defaults to the service protocol when no health check protocol
	// is configured.
	Protocol hcloud.LoadBalancerServiceProtocol
	// Port is nil when unconfigured, in which case the destination port is
	// checked.
	Port     *int
	Interval *time.Duration
	Timeout  *time.Duration
	Retries  *int

	// HTTP is only applied for the HTTP and HTTPS protocols, so it is a value:
	// its options can all be unset while the block itself is still sent.
	HTTP HealthCheckHTTPSpec
}

// HealthCheckHTTPSpec holds the HTTP options of a health check.
type HealthCheckHTTPSpec struct {
	Domain      *string
	Path        *string
	TLS         *bool
	StatusCodes []string
}

// ManagedCertificate is a certificate the cloud controller manager creates and
// renews on behalf of the Service.
type ManagedCertificate struct {
	Name    string
	Labels  map[string]string
	Domains []string
	// UseACMEStaging is for Hetzner internal testing only.
	UseACMEStaging bool
}
