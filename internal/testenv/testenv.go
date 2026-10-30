package testenv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	hrobot "github.com/syself/hrobot-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/cache"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/events"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// TestEnv provides a test server that mocks the Hetzner Cloud and Hetzner Robot
// APIs, together with clients that are configured to talk to it.
type TestEnv struct {
	Server      *httptest.Server
	Mux         *http.ServeMux
	Client      *hcloud.Client
	RobotClient hrobot.RobotClient
	ServerCache *cache.Cache[hcloud.Server]
	Recorder    *events.Recorder
	Cfg         config.HCCMConfiguration
}

func (env *TestEnv) Teardown() {
	env.Server.Close()
	env.Server = nil
	env.Mux = nil
	env.Client = nil
	env.RobotClient = nil
	env.ServerCache = nil
	env.Recorder = nil
}

// New returns a ready to use [TestEnv]. The caller is responsible for calling
// [TestEnv.Teardown].
func New() TestEnv {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithToken("jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jNZXCeTYQ4uArypFM3nh75"),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
		hcloud.WithDebugWriter(os.Stdout),
	)
	robotClient := hrobot.NewBasicAuthClient("", "")
	robotClient.SetBaseURL(server.URL + "/robot")
	serverCache := cache.NewServerCache(client, cache.ModeOne, 10*time.Second)
	recorder := events.NewRecorder(record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: "hcloud-cloud-controller-manager"}))

	cfg := config.HCCMConfiguration{}
	cfg.Instance.AddressFamily = config.AddressFamilyIPv4
	cfg.Instance.ZoneLabelEnabled = true

	return TestEnv{
		Server:      server,
		Mux:         mux,
		Client:      client,
		RobotClient: robotClient,
		ServerCache: serverCache,
		Recorder:    recorder,
		Cfg:         cfg,
	}
}
