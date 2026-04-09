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
	now        func() time.Time
	// mutex is necessary to synchronize parallel access to the cache
	mutex sync.Mutex

	// cache
	servers     []hrobotmodels.Server
	serversByID map[int]*hrobotmodels.Server

	// forcedRefreshServerNames tracks which node names already triggered a forced refresh within the current cache timeout window.
	forcedRefreshServerNames map[string]time.Time
}

func NewCachedClient(cacheTimeout time.Duration, robotClient Client) Client {
	return &cacheRobotClient{
		timeout:     cacheTimeout,
		robotClient: robotClient,
		now:         time.Now,

		serversByID:              make(map[int]*hrobotmodels.Server),
		forcedRefreshServerNames: make(map[string]time.Time),
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

// ServerGetListForceRefresh refreshes the server list immediately, unless the same node already forced a refresh within the current cache timeout window.
func (c *cacheRobotClient) ServerGetListForceRefresh(nodeName string) ([]hrobotmodels.Server, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if nodeName != "" && c.nodeHasAlreadyForcedRefresh(nodeName) {
		if err := c.updateCacheIfNecessary(); err != nil {
			return nil, err
		}
		return c.servers, nil
	}

	if err := c.refreshCache(); err != nil {
		return nil, err
	}

	if nodeName != "" {
		c.forcedRefreshServerNames[nodeName] = c.currentTime()
	}

	return c.servers, nil
}

// Make sure to lock the mutext before calling updateCacheIfNecessary.
func (c *cacheRobotClient) updateCacheIfNecessary() error {
	nextUpdate := c.lastUpdate.Add(c.timeout)
	if c.currentTime().Before(nextUpdate) {
		return nil
	}

	return c.refreshCache()
}

func (c *cacheRobotClient) refreshCache() error {
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
	c.lastUpdate = c.currentTime()
	return nil
}

// nodeHasAlreadyForcedRefresh reports whether this node already triggered a forced refresh within the current cache timeout window and drops expired entries.
func (c *cacheRobotClient) nodeHasAlreadyForcedRefresh(nodeName string) bool {
	forcedAt, found := c.forcedRefreshServerNames[nodeName]
	if !found {
		return false
	}

	if c.currentTime().After(forcedAt.Add(c.timeout)) {
		delete(c.forcedRefreshServerNames, nodeName)
		return false
	}

	return true
}

// currentTime centralizes access to the clock so tests can inject a deterministic time source via c.now.
func (c *cacheRobotClient) currentTime() time.Time {
	if c.now == nil {
		return time.Now()
	}

	return c.now()
}

// ResetGet does not use the cache, as we need up to date information for its function.
func (c *cacheRobotClient) ResetGet(id int) (*hrobotmodels.Reset, error) {
	return c.robotClient.ResetGet(id)
}
