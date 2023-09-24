package hcops

import (
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/util"
	"github.com/syself/hrobot-go/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/scheme"
)

func init() {
	rateLimitWaitTimeRobot, err := util.GetEnvDuration("RATE_LIMIT_WAIT_TIME_ROBOT")
	if err != nil {
		panic(err)
	}

	if rateLimitWaitTimeRobot == 0 {
		rateLimitWaitTimeRobot = 5 * time.Minute
	}

	handler = rateLimitHandler{
		waitTime: rateLimitWaitTimeRobot,
	}

	eventBroadcaster := record.NewBroadcaster()
	recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "hetzner-ccm-ratelimit"})
}

var recorder record.EventRecorder

var handler rateLimitHandler

type rateLimitHandler struct {
	waitTime    time.Duration
	exceeded    bool
	lastChecked time.Time
}

func (rl *rateLimitHandler) set() {
	rl.exceeded = true
	rl.lastChecked = time.Now()
}

func (rl *rateLimitHandler) isExceeded() bool {
	if !rl.exceeded {
		return false
	}

	if time.Now().Before(rl.lastChecked.Add(rl.waitTime)) {
		return true
	}
	// Waiting time is over. Should try again
	rl.exceeded = false
	rl.lastChecked = time.Time{}
	return false
}

func (rl *rateLimitHandler) timeOfNextPossibleAPICall() time.Time {
	emptyTime := time.Time{}
	if rl.lastChecked == emptyTime {
		return emptyTime
	}
	return rl.lastChecked.Add(rl.waitTime)
}

// implement rate limit that is stored in memory

func IsRateLimitExceeded(obj runtime.Object) bool {
	if handler.isExceeded() {
		recorder.Event(obj, "Warning", "RobotRateLimitExceeded", "exceeded Hetzner Robot API rate limit")
		return true
	}
	return false
}

func SetRateLimit() {
	handler.set()
}

func TimeOfNextPossibleAPICall() time.Time {
	return handler.timeOfNextPossibleAPICall()
}

func HandleRateLimitExceededError(err error, obj runtime.Object) {
	if models.IsError(err, models.ErrorCodeRateLimitExceeded) || strings.Contains(err.Error(), "server responded with status code 403") {
		recorder.Event(obj, "Warning", "RobotRateLimitExceeded", "exceeded Hetzner Robot API rate limit")
		SetRateLimit()
	}
}
