package robot

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hrobotmodels "github.com/syself/hrobot-go/models"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
)

func TestRateLimit(t *testing.T) {
	mock := mocks.RobotClient{}
	mock.On("ServerGetList").Return([]hrobotmodels.Server{}, nil).Once()

	client := NewRateLimitedClient(5*time.Minute, &mock)

	servers, err := client.ServerGetList()
	assert.NoError(t, err)
	assert.Empty(t, servers)
	mock.AssertNumberOfCalls(t, "ServerGetList", 1)

	mock.On("ServerGetList").Return(nil, hrobotmodels.Error{Code: hrobotmodels.ErrorCodeRateLimitExceeded, Message: "Rate limit exceeded"}).Once()
	_, err = client.ServerGetList()
	assert.Error(t, err)
	mock.AssertNumberOfCalls(t, "ServerGetList", 2)

	// No further call should be made
	_, err = client.ServerGetList()
	assert.Error(t, err)
	mock.AssertNumberOfCalls(t, "ServerGetList", 2)
}

func TestRateLimitIsExceeded(t *testing.T) {
	client := rateLimitClient{
		waitTime:    5 * time.Minute,
		exceeded:    true,
		lastChecked: time.Now(),
	}
	// Just exceeded
	assert.True(t, client.isExceeded())

	// Exceeded longer than wait time ago
	client.lastChecked = time.Now().Add(-6 * time.Minute)
	assert.False(t, client.isExceeded())

	// Not exceeded ever
	client.exceeded = false
	client.lastChecked = time.Time{}
	assert.False(t, client.isExceeded())
}

func TestRateLimitGetRateLimitError(t *testing.T) {
	client := rateLimitClient{
		waitTime: 5 * time.Minute,
	}
	err := client.getRateLimitError()
	assert.NoError(t, err)

	client.exceeded = true
	client.lastChecked = time.Now()

	err = client.getRateLimitError()
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "rate limit exceeded, next try at "))
}

func TestRateLimitServerGetListForceRefresh(t *testing.T) {
	mock := mocks.RobotClient{}
	mock.On("ServerGetListForceRefresh", "robot-server1").Return([]hrobotmodels.Server{
		{ServerNumber: 321, Name: "robot-server1"},
	}, nil).Once()

	client := NewRateLimitedClient(5*time.Minute, &mock)

	servers, err := client.ServerGetListForceRefresh("robot-server1")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "robot-server1", servers[0].Name)
	mock.AssertNumberOfCalls(t, "ServerGetListForceRefresh", 1)
}

func TestRateLimitServerGetListForceRefreshStopsAfterRateLimit(t *testing.T) {
	mock := mocks.RobotClient{}
	mock.On("ServerGetListForceRefresh", "robot-server1").Return(nil, hrobotmodels.Error{
		Code:    hrobotmodels.ErrorCodeRateLimitExceeded,
		Message: "Rate limit exceeded",
	}).Once()

	client := NewRateLimitedClient(5*time.Minute, &mock)

	_, err := client.ServerGetListForceRefresh("robot-server1")
	require.Error(t, err)

	_, err = client.ServerGetListForceRefresh("robot-server1")
	require.Error(t, err)

	mock.AssertNumberOfCalls(t, "ServerGetListForceRefresh", 1)
}
