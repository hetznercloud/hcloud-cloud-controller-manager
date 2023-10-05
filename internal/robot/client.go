package robot

import (
	"time"

	hrobot "github.com/syself/hrobot-go"
	robotmodels "github.com/syself/hrobot-go/models"
)

type cacheRobotClient struct {
	robotClient hrobot.RobotClient
	timeout     time.Duration

	lastUpdate time.Time

	// cache
	l []robotmodels.Server
	m map[int]*robotmodels.Server
}

func NewClient(robotClient hrobot.RobotClient, cacheTimeout time.Duration) Client {
	return &cacheRobotClient{
		timeout:     cacheTimeout,
		robotClient: robotClient,
	}
}

func (c *cacheRobotClient) ServerGet(id int) (*robotmodels.Server, error) {
	if c.shouldSync() {
		list, err := c.robotClient.ServerGetList()
		if err != nil {
			return nil, err
		}

		// populate list
		c.l = list

		// remove all entries from map and populate it freshly
		c.m = make(map[int]*robotmodels.Server)
		for i, server := range list {
			c.m[server.ServerNumber] = &list[i]
		}

		// set time of last update
		c.lastUpdate = time.Now()
	}

	server, found := c.m[id]
	if !found {
		// return not found error
		return nil, robotmodels.Error{Code: robotmodels.ErrorCodeServerNotFound, Message: "server not found"}
	}

	return server, nil
}

func (c *cacheRobotClient) ServerGetList() ([]robotmodels.Server, error) {
	if c.shouldSync() {
		list, err := c.robotClient.ServerGetList()
		if err != nil {
			return list, err
		}

		// populate list
		c.l = list

		// remove all entries from map and populate it freshly
		c.m = make(map[int]*robotmodels.Server)
		for i, server := range list {
			c.m[server.ServerNumber] = &list[i]
		}

		// set time of last update
		c.lastUpdate = time.Now()
	}

	return c.l, nil
}

func (c *cacheRobotClient) ResetGet(id int) (*robotmodels.Reset, error) {
	return c.robotClient.ResetGet(id)
}

func (c *cacheRobotClient) shouldSync() bool {
	// map is nil means we have no cached value yet
	if c.m == nil {
		c.m = make(map[int]*robotmodels.Server)
		return true
	}
	if time.Now().After(c.lastUpdate.Add(c.timeout)) {
		return true
	}
	return false
}
