package robot

import (
	hrobot "github.com/syself/hrobot-go"
	hrobotmodels "github.com/syself/hrobot-go/models"
)

// adapter wraps hrobot.RobotClient so it satisfies this package's Client interface.
type adapter struct {
	hrobot.RobotClient
}

func NewClient(robotClient hrobot.RobotClient) Client {
	if robotClient == nil {
		return nil
	}

	return &adapter{RobotClient: robotClient}
}

func (a *adapter) ServerGetListForceRefresh(_ string) ([]hrobotmodels.Server, error) {
	return a.ServerGetList()
}
