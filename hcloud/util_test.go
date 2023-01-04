package hcloud

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitSet(t *testing.T) {
	rl := rateLimit{}
	now := time.Now()
	rl.set()
	require.Equal(t, true, rl.exceeded)
	require.NotEqual(t, time.Time{}, rl.lastChecked)
	require.Equal(t, true, now.Before(rl.lastChecked))
}

func TestRateLimitIsExceeded(t *testing.T) {
	now := time.Now()

	rateLimitNotExceeded := rateLimit{}

	require.Equal(t, false, rateLimitNotExceeded.isExceeded())

	rateLimitExceeded := rateLimit{
		exceeded:    true,
		lastChecked: now.Add(-3 * time.Minute),
	}

	require.Equal(t, true, rateLimitExceeded.isExceeded())

	rateLimitWaitingTimeOver := rateLimit{
		exceeded:    true,
		lastChecked: now.Add(-10 * time.Minute),
	}

	require.Equal(t, false, rateLimitWaitingTimeOver.isExceeded())
	require.Equal(t, time.Time{}, rateLimitWaitingTimeOver.lastChecked)
	require.Equal(t, false, rateLimitWaitingTimeOver.exceeded)
}

func TestRateLimitTimeOfNextPossibleAPICall(t *testing.T) {
	now := time.Now()
	lastChecked := now.Add(-3 * time.Minute)
	rateLimitExceeded := rateLimit{
		exceeded:    true,
		lastChecked: lastChecked,
	}

	require.Equal(t, lastChecked.Add(rateLimitWaitingTime), rateLimitExceeded.timeOfNextPossibleAPICall())

	rateLimitNotExceeded := rateLimit{}

	require.Equal(t, time.Time{}, rateLimitNotExceeded.timeOfNextPossibleAPICall())
}
