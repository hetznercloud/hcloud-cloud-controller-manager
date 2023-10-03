package cache

import (
	"time"

	"github.com/syself/hetzner-cloud-controller-manager/internal/robot/client"
	hrobot "github.com/syself/hrobot-go"
	"github.com/syself/hrobot-go/models"
)

var handler = &cacheRobotClient{}

type cacheRobotClient struct {
	robotClient hrobot.RobotClient
	timeout     time.Duration

	lastUpdate time.Time

	// cache
	l []models.Server
	m map[int]*models.Server
}

func NewClient(robotClient hrobot.RobotClient, cacheTimeout time.Duration) client.Client {
	handler.timeout = cacheTimeout
	handler.robotClient = robotClient
	return handler
}

func (c *cacheRobotClient) ServerGet(id int) (*models.Server, error) {
	if c.shouldSync() {
		list, err := c.robotClient.ServerGetList()
		if err != nil {
			return nil, err
		}

		// populate list
		c.l = list

		// remove all entries from map and populate it freshly
		c.m = make(map[int]*models.Server)
		for i, server := range list {
			c.m[server.ServerNumber] = &list[i]
		}

		// set time of last update
		c.lastUpdate = time.Now()
	}

	server, found := c.m[id]
	if !found {
		// return not found error
		return nil, models.Error{Code: models.ErrorCodeServerNotFound, Message: "server not found"}
	}

	return server, nil
}

func (c *cacheRobotClient) ServerGetList() ([]models.Server, error) {
	if c.shouldSync() {
		list, err := c.robotClient.ServerGetList()
		if err != nil {
			return list, err
		}

		// populate list
		c.l = list

		// remove all entries from map and populate it freshly
		c.m = make(map[int]*models.Server)
		for i, server := range list {
			c.m[server.ServerNumber] = &list[i]
		}

		// set time of last update
		c.lastUpdate = time.Now()
	}

	return c.l, nil
}

func (c *cacheRobotClient) shouldSync() bool {
	// map is nil means we have no cached value yet
	if c.m == nil {
		c.m = make(map[int]*models.Server)
		return true
	}
	if time.Now().After(c.lastUpdate.Add(c.timeout)) {
		return true
	}
	return false
}
