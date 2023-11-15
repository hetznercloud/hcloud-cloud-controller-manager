package robot

import (
	"sync"
	"time"

	hrobotmodels "github.com/syself/hrobot-go/models"
)

type cacheRobotClient struct {
	robotClient Client
	timeout     time.Duration

	lastUpdate time.Time
	// mutex is necessary to synchronize parallel access to the cache
	mutex sync.Mutex

	// cache
	servers     []hrobotmodels.Server
	serversByID map[int]*hrobotmodels.Server
}

func NewCachedClient(cacheTimeout time.Duration, robotClient Client) Client {
	return &cacheRobotClient{
		timeout:     cacheTimeout,
		robotClient: robotClient,

		serversByID: make(map[int]*hrobotmodels.Server),
	}
}

func (c *cacheRobotClient) ServerGet(id int) (*hrobotmodels.Server, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.updateCacheIfNecessary(); err != nil {
		return nil, err
	}

	server, found := c.serversByID[id]
	if !found {
		// return not found error
		return nil, hrobotmodels.Error{Code: hrobotmodels.ErrorCodeServerNotFound, Message: "server not found"}
	}

	return server, nil
}

func (c *cacheRobotClient) ServerGetList() ([]hrobotmodels.Server, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.updateCacheIfNecessary(); err != nil {
		return nil, err
	}

	return c.servers, nil
}

// Make sure to lock the mutext before calling updateCacheIfNecessary.
func (c *cacheRobotClient) updateCacheIfNecessary() error {
	nextUpdate := c.lastUpdate.Add(c.timeout)
	if time.Now().Before(nextUpdate) {
		return nil
	}

	servers, err := c.robotClient.ServerGetList()
	if err != nil {
		return err
	}

	// populate servers
	c.servers = servers

	// remove all entries from map and populate it freshly
	c.serversByID = make(map[int]*hrobotmodels.Server)
	for i, server := range servers {
		c.serversByID[server.ServerNumber] = &servers[i]
	}

	// set time of last update
	c.lastUpdate = time.Now()
	return nil
}

// ResetGet does not use the cache, as we need up to date information for its function.
func (c *cacheRobotClient) ResetGet(id int) (*hrobotmodels.Reset, error) {
	return c.robotClient.ResetGet(id)
}
