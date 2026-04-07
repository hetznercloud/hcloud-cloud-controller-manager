package robot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	hrobotmodels "github.com/syself/hrobot-go/models"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
)

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
