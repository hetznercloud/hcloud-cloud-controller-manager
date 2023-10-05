package robot

import (
	"strings"
	"time"

	robotmodels "github.com/syself/hrobot-go/models"
)

func InitRateLimit(rateLimitWaitTime time.Duration) {
	rateLimit = rateLimitHandler{
		waitTime: rateLimitWaitTime,
	}
}

var rateLimit rateLimitHandler

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
	return rateLimit.isExceeded()
}

func SetRateLimit() {
	rateLimit.set()
}

func TimeOfNextPossibleAPICall() time.Time {
	return rateLimit.timeOfNextPossibleAPICall()
}

func HandleRateLimitExceededError(err error) {
	if robotmodels.IsError(err, robotmodels.ErrorCodeRateLimitExceeded) || strings.Contains(err.Error(), "server responded with status code 403") {
		SetRateLimit()
	}
}
