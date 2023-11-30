package robot

import (
	"fmt"
	"strings"
	"time"

	hrobotmodels "github.com/syself/hrobot-go/models"
)

type rateLimitClient struct {
	robotClient Client

	waitTime    time.Duration
	exceeded    bool
	lastChecked time.Time
}

func NewRateLimitedClient(rateLimitWaitTime time.Duration, robotClient Client) Client {
	return &rateLimitClient{
		robotClient: robotClient,

		waitTime: rateLimitWaitTime,
	}
}

func (c *rateLimitClient) ServerGet(id int) (*hrobotmodels.Server, error) {
	if c.isExceeded() {
		return nil, c.getRateLimitError()
	}

	server, err := c.robotClient.ServerGet(id)
	c.handleError(err)
	return server, err
}

func (c *rateLimitClient) ServerGetList() ([]hrobotmodels.Server, error) {
	if c.isExceeded() {
		return nil, c.getRateLimitError()
	}

	servers, err := c.robotClient.ServerGetList()
	c.handleError(err)
	return servers, err
}

func (c *rateLimitClient) ResetGet(id int) (*hrobotmodels.Reset, error) {
	if c.isExceeded() {
		return nil, c.getRateLimitError()
	}

	reset, err := c.robotClient.ResetGet(id)
	c.handleError(err)
	return reset, err
}

func (c *rateLimitClient) set() {
	c.exceeded = true
	c.lastChecked = time.Now()
}

func (c *rateLimitClient) isExceeded() bool {
	if !c.exceeded {
		return false
	}

	if time.Now().Before(c.lastChecked.Add(c.waitTime)) {
		return true
	}
	// Waiting time is over. Should try again
	c.exceeded = false
	c.lastChecked = time.Time{}
	return false
}

func (c *rateLimitClient) handleError(err error) {
	if err == nil {
		return
	}

	if hrobotmodels.IsError(err, hrobotmodels.ErrorCodeRateLimitExceeded) || strings.Contains(err.Error(), "server responded with status code 403") {
		c.set()
	}
}

func (c *rateLimitClient) getRateLimitError() error {
	if !c.isExceeded() {
		return nil
	}

	nextPossibleCall := c.lastChecked.Add(c.waitTime)
	return fmt.Errorf("rate limit exceeded, next try at %q", nextPossibleCall.String())
}
