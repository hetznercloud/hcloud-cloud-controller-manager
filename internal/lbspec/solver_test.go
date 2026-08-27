package lbspec_test

import (
	"maps"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/lbspec"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func service(uid string, annotations map[string]string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:         types.UID(uid),
			Annotations: make(map[string]string, len(annotations)),
		},
	}
	maps.Copy(svc.Annotations, annotations)
	return svc
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		cfg         config.LoadBalancerConfiguration
		check       func(t *testing.T, spec lbspec.Spec)
	}{
		{
			name: "defaults without annotations or configuration",
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, "asomeuid", spec.Name, "derived from the service uid")
				assert.True(t, spec.NameUnset)
				assert.Empty(t, spec.Hostname)
				assert.True(t, spec.NodeSelector.Empty(), "selects every Node")
				assert.Equal(t, map[string]string{lbspec.LabelServiceUID: "some-uid"}, spec.Labels)
				assert.Equal(t, lbspec.DefaultType, spec.Type)
				assert.True(t, spec.TypeUnset)
				assert.Empty(t, spec.Location)
				assert.Empty(t, spec.NetworkZone)
				assert.Empty(t, spec.Algorithm, "an unconfigured algorithm is left alone")
				assert.Nil(t, spec.PublicInterface, "an unconfigured public interface is left alone")
				assert.Nil(t, spec.IPv4RDNS)
				assert.Nil(t, spec.IPv6RDNS)
				assert.Nil(t, spec.PrivateIPv4)
				assert.Nil(t, spec.PrivateSubnetIPRange)
				assert.False(t, spec.UsePrivateIP)
				assert.Nil(t, spec.ManagedCertificate)
				assert.Equal(t, hcloud.LoadBalancerServiceProtocolTCP, spec.Service.Protocol)
				assert.Nil(t, spec.Service.ProxyProtocol)
				assert.Nil(t, spec.Service.HTTP, "no HTTP block without HTTP options")
				assert.Nil(t, spec.Service.HealthCheck, "no health check without health check options")
			},
		},
		{
			name: "name and hostname are separate settings",
			annotations: map[string]string{
				string(annotation.LBName):     "my-lb",
				string(annotation.LBHostname): "lb.example.com",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, "my-lb", spec.Name)
				assert.False(t, spec.NameUnset)
				assert.Equal(t, "lb.example.com", spec.Hostname)
			},
		},
		{
			name: "annotations take precedence over the configuration",
			cfg: config.LoadBalancerConfiguration{
				Type:             "lb21",
				Location:         "hel1",
				AlgorithmType:    hcloud.LoadBalancerAlgorithmTypeRoundRobin,
				PrivateIPEnabled: true,
			},
			annotations: map[string]string{
				string(annotation.LBType):          "lb31",
				string(annotation.LBLocation):      "fsn1",
				string(annotation.LBAlgorithmType): "least_connections",
				string(annotation.LBUsePrivateIP):  "false",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, "lb31", spec.Type)
				assert.False(t, spec.TypeUnset)
				assert.Equal(t, "fsn1", spec.Location)
				assert.Equal(t, hcloud.LoadBalancerAlgorithmTypeLeastConnections, spec.Algorithm)
				assert.False(t, spec.UsePrivateIP)
			},
		},
		{
			name: "configuration applies without annotations",
			cfg: config.LoadBalancerConfiguration{
				Type:                 "lb21",
				NetworkZone:          "eu-central",
				AlgorithmType:        hcloud.LoadBalancerAlgorithmTypeLeastConnections,
				DisablePublicNetwork: new(true),
				PrivateSubnetIPRange: "10.0.0.0/24",
				PrivateIPEnabled:     true,
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, "lb21", spec.Type)
				assert.False(t, spec.TypeUnset)
				assert.Equal(t, hcloud.NetworkZone("eu-central"), spec.NetworkZone)
				assert.Equal(t, hcloud.LoadBalancerAlgorithmTypeLeastConnections, spec.Algorithm)
				assert.Equal(t, new(false), spec.PublicInterface, "DISABLE_PUBLIC_NETWORK=true means disabled")
				assert.Equal(t, "10.0.0.0/24", spec.PrivateSubnetIPRange.String())
				assert.True(t, spec.UsePrivateIP)
			},
		},
		{
			name: "an empty annotation resets a configured location",
			cfg: config.LoadBalancerConfiguration{
				Location: "hel1",
			},
			annotations: map[string]string{
				string(annotation.LBLocation):    "",
				string(annotation.LBNetworkZone): "eu-central",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Empty(t, spec.Location)
				assert.Equal(t, hcloud.NetworkZone("eu-central"), spec.NetworkZone)
			},
		},
		{
			name: "a location wins over a network zone",
			annotations: map[string]string{
				string(annotation.LBLocation):    "nbg1",
				string(annotation.LBNetworkZone): "eu-central",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, "nbg1", spec.Location)
				assert.Empty(t, spec.NetworkZone, "the API accepts only one of them")
			},
		},
		{
			name: "disable annotations are resolved as enabled settings",
			cfg: config.LoadBalancerConfiguration{
				PrivateIngressEnabled: true,
				IPv6Enabled:           true,
			},
			annotations: map[string]string{
				string(annotation.LBDisablePublicNetwork):  "true",
				string(annotation.LBDisablePrivateIngress): "true",
				string(annotation.LBIPv6Disabled):          "true",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, new(false), spec.PublicInterface)
				assert.False(t, spec.PrivateIngress)
				assert.False(t, spec.IPv6)
			},
		},
		{
			name: "reverse DNS records tell an empty value from an unset one",
			annotations: map[string]string{
				string(annotation.LBPublicIPv4RDNS): "",
				string(annotation.LBPublicIPv6RDNS): "lb.example.com",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, new(""), spec.IPv4RDNS, "an empty record is a value, not an omission")
				assert.Equal(t, new("lb.example.com"), spec.IPv6RDNS)
			},
		},
		{
			name: "private network addressing is parsed",
			annotations: map[string]string{
				string(annotation.LBPrivateIPv4):        "10.0.1.5",
				string(annotation.PrivateSubnetIPRange): "10.0.1.0/24",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.True(t, net.ParseIP("10.0.1.5").Equal(spec.PrivateIPv4))
				assert.Equal(t, "10.0.1.0/24", spec.PrivateSubnetIPRange.String())
			},
		},
		{
			name: "node selector is parsed",
			annotations: map[string]string{
				string(annotation.LBNodeSelector): "environment=production",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, "environment=production", spec.NodeSelector.String())
			},
		},
		{
			name: "HTTP options configure the HTTP block",
			annotations: map[string]string{
				string(annotation.LBSvcProtocol):           "http",
				string(annotation.LBSvcHTTPCookieName):     "my-cookie",
				string(annotation.LBSvcHTTPCookieLifetime): "1h",
				string(annotation.LBSvcHTTPTimeoutIdle):    "30s",
				string(annotation.LBSvcRedirectHTTP):       "true",
				string(annotation.LBSvcHTTPStickySessions): "true",
				string(annotation.LBSvcHTTPCertificates):   "1,some-cert",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Equal(t, hcloud.LoadBalancerServiceProtocolHTTP, spec.Service.Protocol)
				require.NotNil(t, spec.Service.HTTP)
				assert.Equal(t, new("my-cookie"), spec.Service.HTTP.CookieName)
				assert.Equal(t, new(time.Hour), spec.Service.HTTP.CookieLifetime)
				assert.Equal(t, new(30*time.Second), spec.Service.HTTP.TimeoutIdle)
				assert.Equal(t, new(true), spec.Service.HTTP.RedirectHTTP)
				assert.Equal(t, new(true), spec.Service.HTTP.StickySessions)
				assert.Equal(t, []*hcloud.Certificate{{ID: 1}, {Name: "some-cert"}},
					spec.Service.HTTP.Certificates, "references are kept as written")
			},
		},
		{
			name: "a managed certificate defaults its name to the service uid",
			annotations: map[string]string{
				string(annotation.LBSvcHTTPCertificateType):           "managed",
				string(annotation.LBSvcHTTPManagedCertificateDomains): "example.com,*.example.com",
				string(annotation.LBSvcHTTPCertificates):              "ignored",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				require.NotNil(t, spec.ManagedCertificate)
				assert.Equal(t, "ccm-managed-certificate-some-uid", spec.ManagedCertificate.Name)
				assert.Equal(t, []string{"example.com", "*.example.com"}, spec.ManagedCertificate.Domains)
				assert.False(t, spec.ManagedCertificate.UseACMEStaging)

				require.NotNil(t, spec.Service.HTTP, "the certificate is attached to an HTTP service")
				assert.Empty(t, spec.Service.HTTP.Certificates,
					"the managed certificate is looked up by label, and the list annotation is ignored")
			},
		},
		{
			name: "a managed certificate can be named and use ACME staging",
			annotations: map[string]string{
				string(annotation.LBSvcHTTPCertificateType):                  "managed",
				string(annotation.LBSvcHTTPManagedCertificateName):           "my-cert",
				string(annotation.LBSvcHTTPManagedCertificateDomains):        "example.com",
				string(annotation.LBSvcHTTPManagedCertificateUseACMEStaging): "true",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				require.NotNil(t, spec.ManagedCertificate)
				assert.Equal(t, "my-cert", spec.ManagedCertificate.Name)
				assert.True(t, spec.ManagedCertificate.UseACMEStaging)
			},
		},
		{
			name: "an uploaded certificate type is not a managed certificate",
			annotations: map[string]string{
				string(annotation.LBSvcHTTPCertificateType): "uploaded",
				string(annotation.LBSvcHTTPCertificates):    "1",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Nil(t, spec.ManagedCertificate)
				require.NotNil(t, spec.Service.HTTP)
				assert.Equal(t, []*hcloud.Certificate{{ID: 1}}, spec.Service.HTTP.Certificates)
			},
		},
		{
			name: "health check options configure a health check",
			annotations: map[string]string{
				string(annotation.LBSvcHealthCheckProtocol): "http",
				string(annotation.LBSvcHealthCheckPort):     "8080",
				string(annotation.LBSvcHealthCheckInterval): "1h",
				string(annotation.LBSvcHealthCheckTimeout):  "30s",
				string(annotation.LBSvcHealthCheckRetries):  "5",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				require.NotNil(t, spec.Service.HealthCheck)
				assert.Equal(t, hcloud.LoadBalancerServiceProtocolHTTP, spec.Service.HealthCheck.Protocol)
				assert.Equal(t, new(8080), spec.Service.HealthCheck.Port)
				assert.Equal(t, new(time.Hour), spec.Service.HealthCheck.Interval)
				assert.Equal(t, new(30*time.Second), spec.Service.HealthCheck.Timeout)
				assert.Equal(t, new(5), spec.Service.HealthCheck.Retries)
			},
		},
		{
			name: "cluster-wide health check defaults configure a health check",
			cfg: config.LoadBalancerConfiguration{
				HealthCheckInterval: 30 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
				HealthCheckRetries:  5,
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				require.NotNil(t, spec.Service.HealthCheck)
				assert.Equal(t, hcloud.LoadBalancerServiceProtocolTCP, spec.Service.HealthCheck.Protocol)
				assert.Equal(t, new(30*time.Second), spec.Service.HealthCheck.Interval)
				assert.Equal(t, new(5*time.Second), spec.Service.HealthCheck.Timeout)
				assert.Equal(t, new(5), spec.Service.HealthCheck.Retries)
			},
		},
		{
			name: "an unset health check protocol follows the service protocol",
			annotations: map[string]string{
				string(annotation.LBSvcProtocol):        "https",
				string(annotation.LBSvcHealthCheckPort): "8080",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				require.NotNil(t, spec.Service.HealthCheck)
				assert.Equal(t, hcloud.LoadBalancerServiceProtocolHTTPS, spec.Service.HealthCheck.Protocol)
			},
		},
		{
			name: "health check HTTP options alone do not configure a health check",
			annotations: map[string]string{
				string(annotation.LBSvcHealthCheckHTTPPath): "/healthz",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				assert.Nil(t, spec.Service.HealthCheck,
					"a TCP health check has no HTTP options, so the path alone is not enough")
			},
		},
		{
			name: "health check HTTP options are read for HTTP health checks",
			annotations: map[string]string{
				string(annotation.LBSvcHealthCheckProtocol):                "http",
				string(annotation.LBSvcHealthCheckHTTPDomain):              "example.com",
				string(annotation.LBSvcHealthCheckHTTPPath):                "/healthz",
				string(annotation.LBSvcHealthCheckHTTPValidateCertificate): "true",
				string(annotation.LBSvcHealthCheckHTTPStatusCodes):         "200,202",
			},
			check: func(t *testing.T, spec lbspec.Spec) {
				require.NotNil(t, spec.Service.HealthCheck)
				assert.Equal(t, new("example.com"), spec.Service.HealthCheck.HTTP.Domain)
				assert.Equal(t, new("/healthz"), spec.Service.HealthCheck.HTTP.Path)
				assert.Equal(t, new(true), spec.Service.HealthCheck.HTTP.TLS)
				assert.Equal(t, []string{"200", "202"}, spec.Service.HealthCheck.HTTP.StatusCodes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := lbspec.Resolve(record.NewFakeRecorder(10), service("some-uid", tt.annotations), tt.cfg)
			require.NoError(t, err)
			tt.check(t, spec)
		})
	}
}

func TestResolveReportsEveryInvalidAnnotation(t *testing.T) {
	svc := service("some-uid", map[string]string{
		string(annotation.LBAlgorithmType):         "sideways",
		string(annotation.LBUsePrivateIP):          "maybe",
		string(annotation.LBSvcHealthCheckRetries): "many",
		string(annotation.LBPrivateIPv4):           "not-an-ip",
		string(annotation.PrivateSubnetIPRange):    "not-a-cidr",
	})

	logs := testsupport.CaptureKlog(t)
	recorder := record.NewFakeRecorder(10)

	_, err := lbspec.Resolve(recorder, svc, config.LoadBalancerConfiguration{})

	// The error only counts the invalid annotations, the details are reported
	// through the logs and the Kubernetes events.
	require.EqualError(t, err, "5 Load Balancer annotation(s) are invalid")

	// One reconcile tells the user about all of their typos, not just the first.
	for _, want := range []string{"sideways", "maybe", "many", "not-an-ip", "not-a-cidr"} {
		assert.Contains(t, logs.String(), want)
	}

	// Every invalid annotation gets its own event, so that a single event stays
	// readable.
	assert.Len(t, recorder.Events, 5)
}

func TestResolveErrors(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantErr     string
	}{
		{
			name:        "invalid algorithm",
			annotations: map[string]string{string(annotation.LBAlgorithmType): "sideways"},
			wantErr:     "invalid algorithm type: sideways",
		},
		{
			name:        "invalid service protocol",
			annotations: map[string]string{string(annotation.LBSvcProtocol): "smtp"},
			wantErr:     "invalid protocol: smtp",
		},
		{
			name:        "invalid node selector",
			annotations: map[string]string{string(annotation.LBNodeSelector): "environment=production=staging"},
			wantErr:     "unable to parse the node-selector annotation",
		},
		{
			name:        "invalid duration",
			annotations: map[string]string{string(annotation.LBSvcHTTPTimeoutIdle): "30 fortnights"},
			wantErr:     "30 fortnights",
		},
		{
			name: "managed certificate without domains",
			annotations: map[string]string{
				string(annotation.LBSvcHTTPCertificateType): "managed",
			},
			wantErr: "no domains for managed certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := testsupport.CaptureKlog(t)
			recorder := record.NewFakeRecorder(10)

			_, err := lbspec.Resolve(recorder, service("some-uid", tt.annotations), config.LoadBalancerConfiguration{})

			require.EqualError(t, err, "1 Load Balancer annotation(s) are invalid")

			// The reason is only reported through the logs and the Kubernetes
			// event.
			assert.Contains(t, logs.String(), tt.wantErr)
			require.Len(t, recorder.Events, 1)
			event := <-recorder.Events
			assert.Contains(t, event, "InvalidLoadBalancerAnnotation")
			assert.Contains(t, event, tt.wantErr)
		})
	}
}

func TestName(t *testing.T) {
	t.Run("derived from the service uid", func(t *testing.T) {
		assert.Equal(t, "a0000000000000000000000000000000", lbspec.Name(service("0000000-0000-0000-0000-000000000000", nil)))
	})

	t.Run("from the annotation", func(t *testing.T) {
		svc := service("some-uid", map[string]string{string(annotation.LBName): "my-lb"})
		assert.Equal(t, "my-lb", lbspec.Name(svc))
	})
}
