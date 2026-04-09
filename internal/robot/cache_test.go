package robot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hrobotmodels "github.com/syself/hrobot-go/models"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
)

// TestCacheClientServerGetListForceRefreshSkipsRepeatedNameWithinTimeout verifies that repeated forced refreshes for the same node name reuse the cached result until the timeout expires.
func TestCacheClientServerGetListForceRefreshSkipsRepeatedNameWithinTimeout(t *testing.T) {
	now := time.Date(2026, time.April, 7, 13, 0, 0, 0, time.UTC)
	mockClient := &mocks.RobotClient{}
	mockClient.On("ServerGetList").Return([]hrobotmodels.Server{
		{ServerNumber: 321, Name: "robot-server1"},
	}, nil).Once()

	client := &cacheRobotClient{
		robotClient:              mockClient,
		timeout:                  time.Hour,
		now:                      func() time.Time { return now },
		serversByID:              make(map[int]*hrobotmodels.Server),
		forcedRefreshServerNames: make(map[string]time.Time),
	}

	servers, err := client.ServerGetListForceRefresh("robot-server2")
	assert.NoError(t, err)
	assert.Len(t, servers, 1)

	servers, err = client.ServerGetListForceRefresh("robot-server2")
	assert.NoError(t, err)
	assert.Len(t, servers, 1)

	mockClient.AssertNumberOfCalls(t, "ServerGetList", 1)
}

// TestCacheClientServerGetListForceRefreshExpiresPerName verifies that a node name can trigger another forced refresh after its timeout window has expired.
func TestCacheClientServerGetListForceRefreshExpiresPerName(t *testing.T) {
	now := time.Date(2026, time.April, 7, 13, 0, 0, 0, time.UTC)
	mockClient := &mocks.RobotClient{}
	mockClient.On("ServerGetList").Return([]hrobotmodels.Server{
		{ServerNumber: 321, Name: "robot-server1"},
	}, nil).Twice()

	client := &cacheRobotClient{
		robotClient:              mockClient,
		timeout:                  time.Hour,
		now:                      func() time.Time { return now },
		serversByID:              make(map[int]*hrobotmodels.Server),
		forcedRefreshServerNames: make(map[string]time.Time),
	}

	_, err := client.ServerGetListForceRefresh("robot-server2")
	assert.NoError(t, err)

	now = now.Add(time.Hour + time.Second)

	_, err = client.ServerGetListForceRefresh("robot-server2")
	assert.NoError(t, err)

	mockClient.AssertNumberOfCalls(t, "ServerGetList", 2)
}

type cacheTestRobotClient struct {
	servers    []hrobotmodels.Server
	listCalls  int
	reset      *hrobotmodels.Reset
	resetCalls int
}

func (c *cacheTestRobotClient) ServerGet(int) (*hrobotmodels.Server, error) {
	panic("this method should not be called")
}

func (c *cacheTestRobotClient) ServerGetList() ([]hrobotmodels.Server, error) {
	c.listCalls++
	return c.servers, nil
}

func (c *cacheTestRobotClient) ServerGetListForceRefresh(string) ([]hrobotmodels.Server, error) {
	panic("this method should not be called")
}

func (c *cacheTestRobotClient) ResetGet(int) (*hrobotmodels.Reset, error) {
	c.resetCalls++
	return c.reset, nil
}

func TestNewCachedClientCachesServerListAndLookupByID(t *testing.T) {
	backend := &cacheTestRobotClient{
		servers: []hrobotmodels.Server{
			{ServerNumber: 321, Name: "robot-server1"},
		},
	}

	client := NewCachedClient(time.Hour, backend)

	servers, err := client.ServerGetList()
	require.NoError(t, err)
	require.Len(t, servers, 1)

	server, err := client.ServerGet(321)
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "robot-server1", server.Name)

	missingServer, err := client.ServerGet(999)
	require.Error(t, err)
	assert.Nil(t, missingServer)
	assert.True(t, hrobotmodels.IsError(err, hrobotmodels.ErrorCodeServerNotFound))
	assert.Equal(t, 1, backend.listCalls)
}

func TestCacheClientCurrentTimeFallsBackToTimeNow(t *testing.T) {
	client := &cacheRobotClient{}

	assert.WithinDuration(t, time.Now(), client.currentTime(), time.Second)
}

func TestNewCachedClientResetGetBypassesCache(t *testing.T) {
	expectedReset := &hrobotmodels.Reset{OperatingStatus: "running"}
	backend := &cacheTestRobotClient{reset: expectedReset}

	client := NewCachedClient(time.Hour, backend)

	reset, err := client.ResetGet(321)
	require.NoError(t, err)
	assert.Same(t, expectedReset, reset)
	assert.Equal(t, 1, backend.resetCalls)
}
