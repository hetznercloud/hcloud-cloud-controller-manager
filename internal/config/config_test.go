package config

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
)

func TestRead(t *testing.T) {
	tests := []struct {
		name    string
		env     []string
		files   map[string]string
		want    HCCMConfiguration
		wantErr error
	}{
		{
			name: "minimal",
			env:  []string{},
			want: HCCMConfiguration{
				Robot:        RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
			},
			wantErr: nil,
		},
		{
			// Matches the default env from deploy/ccm-networks.yaml
			name: "default deployment",
			env: []string{
				"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
				"HCLOUD_NETWORK", "foobar",
			},
			want: HCCMConfiguration{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Robot:        RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				Network: NetworkConfiguration{
					NameOrID: "foobar",
				},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
				Route:        RouteConfiguration{Enabled: true},
			},
			wantErr: nil,
		},
		{
			name: "secrets from file",
			env: []string{
				"HCLOUD_TOKEN_FILE", "/tmp/hetzner-token",
				"ROBOT_USER_FILE", "/tmp/hetzner-user",
				"ROBOT_PASSWORD_FILE", "/tmp/hetzner-password",
			},
			files: map[string]string{
				"hetzner-token":    "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
				"hetzner-user":     "foobar",
				"hetzner-password": `secret-password`,
			},
			want: HCCMConfiguration{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Robot: RobotConfiguration{
					Enabled:           false,
					User:              "foobar",
					Password:          "secret-password",
					CacheTimeout:      5 * time.Minute,
					RateLimitWaitTime: 0,
				},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
				Route:        RouteConfiguration{Enabled: false},
			},
			wantErr: nil,
		},
		{
			name: "secrets from unknown file",
			env: []string{
				"HCLOUD_TOKEN_FILE", "/tmp/hetzner-token",
				"ROBOT_USER_FILE", "/tmp/hetzner-user",
				"ROBOT_PASSWORD_FILE", "/tmp/hetzner-password",
			},
			files: map[string]string{}, // don't create files
			want: HCCMConfiguration{
				HCloudClient: HCloudClientConfiguration{Token: ""},
				Robot:        RobotConfiguration{User: "", Password: "", CacheTimeout: 0},
				Metrics:      MetricsConfiguration{Enabled: false},
				Instance:     InstanceConfiguration{},
				LoadBalancer: LoadBalancerConfiguration{Enabled: false},
				Route:        RouteConfiguration{Enabled: false},
			},
			wantErr: errors.New(`failed to read HCLOUD_TOKEN_FILE: open /tmp/hetzner-token: no such file or directory
failed to read ROBOT_USER_FILE: open /tmp/hetzner-user: no such file or directory
failed to read ROBOT_PASSWORD_FILE: open /tmp/hetzner-password: no such file or directory`),
		},
		{
			name: "client",
			env: []string{
				"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
				"HCLOUD_ENDPOINT", "https://api.example.com",
				"HCLOUD_DEBUG", "true",
			},
			want: HCCMConfiguration{
				HCloudClient: HCloudClientConfiguration{
					Token:    "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
					Endpoint: "https://api.example.com",
					Debug:    true,
				},
				Robot:        RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
			},
			wantErr: nil,
		},
		{
			name: "robot",
			env: []string{
				"ROBOT_ENABLED", "true",
				"ROBOT_USER", "foobar",
				"ROBOT_PASSWORD", "secret-password",
				"ROBOT_RATE_LIMIT_WAIT_TIME", "5m",
				"ROBOT_CACHE_TIMEOUT", "1m",
			},
			want: HCCMConfiguration{
				Robot: RobotConfiguration{
					Enabled:           true,
					User:              "foobar",
					Password:          "secret-password",
					CacheTimeout:      1 * time.Minute,
					RateLimitWaitTime: 5 * time.Minute,
				},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
			},
			wantErr: nil,
		},
		{
			name: "instance",
			env: []string{
				"HCLOUD_INSTANCES_ADDRESS_FAMILY", "ipv6",
			},
			want: HCCMConfiguration{
				Robot:        RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv6},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
			},
			wantErr: nil,
		},
		{
			name: "network",
			env: []string{
				"HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK", "true",
				"HCLOUD_NETWORK", "foobar",
			},
			want: HCCMConfiguration{
				Robot:        RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
				Network: NetworkConfiguration{
					NameOrID:             "foobar",
					DisableAttachedCheck: true,
				},
				Route: RouteConfiguration{Enabled: true},
			},
			wantErr: nil,
		},
		{
			name: "route",
			env: []string{
				"HCLOUD_NETWORK", "foobar",
				"HCLOUD_NETWORK_ROUTES_ENABLED", "false",
			},
			want: HCCMConfiguration{
				Robot:        RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
				Network: NetworkConfiguration{
					NameOrID: "foobar",
				},
				Route: RouteConfiguration{Enabled: false},
			},
			wantErr: nil,
		},
		{
			name: "load balancer",
			env: []string{
				"HCLOUD_LOAD_BALANCERS_ENABLED", "false",
				"HCLOUD_LOAD_BALANCERS_LOCATION", "nbg1",
				"HCLOUD_LOAD_BALANCERS_NETWORK_ZONE", "eu-central",
				"HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS", "true",
				"HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP", "true",
				"HCLOUD_LOAD_BALANCERS_DISABLE_IPV6", "true",
			},
			want: HCCMConfiguration{
				Robot:    RobotConfiguration{CacheTimeout: 5 * time.Minute},
				Metrics:  MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance: InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{
					Enabled:               false,
					Location:              "nbg1",
					NetworkZone:           "eu-central",
					DisablePrivateIngress: true,
					UsePrivateIP:          true,
					DisableIPv6:           true,
				},
			},
			wantErr: nil,
		},
		{
			name: "error parsing bool values",
			env: []string{
				// Required to parse HCLOUD_NETWORK_ROUTES_ENABLED
				"HCLOUD_NETWORK", "foobar",

				"ROBOT_ENABLED", "no",
				"HCLOUD_DEBUG", "foo",
				"HCLOUD_METRICS_ENABLED", "bar",
				"HCLOUD_LOAD_BALANCERS_ENABLED", "nej",
				"HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS", "nyet",
				"HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP", "nein",
				"HCLOUD_LOAD_BALANCERS_DISABLE_IPV6", "ja",
				"HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK", "oui",
				"HCLOUD_NETWORK_ROUTES_ENABLED", "si",
			},
			wantErr: errors.New(`failed to parse HCLOUD_DEBUG: strconv.ParseBool: parsing "foo": invalid syntax
failed to parse ROBOT_ENABLED: strconv.ParseBool: parsing "no": invalid syntax
failed to parse HCLOUD_METRICS_ENABLED: strconv.ParseBool: parsing "bar": invalid syntax
failed to parse HCLOUD_LOAD_BALANCERS_ENABLED: strconv.ParseBool: parsing "nej": invalid syntax
failed to parse HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS: strconv.ParseBool: parsing "nyet": invalid syntax
failed to parse HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP: strconv.ParseBool: parsing "nein": invalid syntax
failed to parse HCLOUD_LOAD_BALANCERS_DISABLE_IPV6: strconv.ParseBool: parsing "ja": invalid syntax
failed to parse HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK: strconv.ParseBool: parsing "oui": invalid syntax
failed to parse HCLOUD_NETWORK_ROUTES_ENABLED: strconv.ParseBool: parsing "si": invalid syntax`),
		},
		{
			name: "error parsing duration values",
			env: []string{
				"ROBOT_CACHE_TIMEOUT", "biweekly",
				"ROBOT_RATE_LIMIT_WAIT_TIME", "42fortnights",
			},
			wantErr: errors.New(`failed to parse ROBOT_CACHE_TIMEOUT: time: invalid duration "biweekly"
failed to parse ROBOT_RATE_LIMIT_WAIT_TIME: time: unknown unit "fortnights" in duration "42fortnights"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetEnv := testsupport.Setenv(t, tt.env...)
			defer resetEnv()
			resetFiles := testsupport.SetFiles(t, tt.files)
			defer resetFiles()

			got, err := Read()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHCCMConfiguration_Validate(t *testing.T) {
	type fields struct {
		HCloudClient HCloudClientConfiguration
		Robot        RobotConfiguration
		Metrics      MetricsConfiguration
		Instance     InstanceConfiguration
		LoadBalancer LoadBalancerConfiguration
		Network      NetworkConfiguration
		Route        RouteConfiguration
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "minimal",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
			},
			wantErr: nil,
		},
		{
			// Matches the default env from deploy/ccm-networks.yaml
			name: "default deployment",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Metrics:      MetricsConfiguration{Enabled: true, Address: ":8233"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				Network: NetworkConfiguration{
					NameOrID: "foobar",
				},
				LoadBalancer: LoadBalancerConfiguration{Enabled: true},
				Route:        RouteConfiguration{Enabled: true},
			},
			wantErr: nil,
		},

		{
			name: "token missing",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: ""},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
			},
			wantErr: errors.New("environment variable \"HCLOUD_TOKEN\" is required"),
		},
		{
			name: "token invalid length",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: "abc"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
			},
			wantErr: errors.New("entered token is invalid (must be exactly 64 characters long)"),
		},
		{
			name: "address family invalid",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamily("foobar")},
			},
			wantErr: errors.New("invalid value for \"HCLOUD_INSTANCES_ADDRESS_FAMILY\", expect one of: ipv4,ipv6,dualstack"),
		},
		{
			name: "LB location and network zone set",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				LoadBalancer: LoadBalancerConfiguration{
					Location:    "nbg1",
					NetworkZone: "eu-central",
				},
			},
			wantErr: errors.New("invalid value for \"HCLOUD_LOAD_BALANCERS_LOCATION\"/\"HCLOUD_LOAD_BALANCERS_NETWORK_ZONE\", only one of them can be set"),
		},
		{
			name: "robot enabled but missing credentials",
			fields: fields{
				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},

				Robot: RobotConfiguration{
					Enabled: true,
				},
			},
			wantErr: errors.New(`environment variable "ROBOT_USER" is required if Robot support is enabled
environment variable "ROBOT_PASSWORD" is required if Robot support is enabled`),
		},
		{
			name: "robot & routes activated",
			fields: fields{

				HCloudClient: HCloudClientConfiguration{Token: "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq"},
				Instance:     InstanceConfiguration{AddressFamily: AddressFamilyIPv4},
				Route:        RouteConfiguration{Enabled: true},
				Robot: RobotConfiguration{
					Enabled: true,

					User:     "foo",
					Password: "bar",
				},
			},
			wantErr: errors.New("using Routes with Robot is not supported"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := HCCMConfiguration{
				HCloudClient: tt.fields.HCloudClient,
				Robot:        tt.fields.Robot,
				Metrics:      tt.fields.Metrics,
				Instance:     tt.fields.Instance,
				LoadBalancer: tt.fields.LoadBalancer,
				Network:      tt.fields.Network,
				Route:        tt.fields.Route,
			}
			err := c.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
