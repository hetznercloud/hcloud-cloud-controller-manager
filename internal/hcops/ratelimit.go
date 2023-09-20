package hcops

import (
	"strings"
	"time"

	"github.com/syself/hetzner-cloud-controller-manager/internal/util"
	"github.com/syself/hrobot-go/models"
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
}

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

func IsRateLimitExceeded() bool {
	return handler.isExceeded()
}

func SetRateLimit() {
	handler.set()
}

func TimeOfNextPossibleAPICall() time.Time {
	return handler.timeOfNextPossibleAPICall()
}

func HandleRateLimitExceededError(err error) {
	if models.IsError(err, models.ErrorCodeRateLimitExceeded) || strings.Contains(err.Error(), "server responded with status code 403") {
		SetRateLimit()
	}
}
