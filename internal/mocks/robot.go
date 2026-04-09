package mocks

import (
	"github.com/stretchr/testify/mock"
	hrobot "github.com/syself/hrobot-go"
	hrobotmodels "github.com/syself/hrobot-go/models"
)

type RobotClient struct {
	mock.Mock
	hrobot.RobotClient // embedded for compile-time interface satisfaction; unmocked methods will panic
}

func (m *RobotClient) ServerGet(id int) (*hrobotmodels.Server, error) {
	args := m.Called()
	return getRobotServer(args, 0), args.Error(1)
}

func (m *RobotClient) ServerGetList() ([]hrobotmodels.Server, error) {
	args := m.Called()
	return getRobotServers(args, 0), args.Error(1)
}

func (m *RobotClient) ResetGet(id int) (*hrobotmodels.Reset, error) {
	args := m.Called(id)
	return getReset(args, 0), args.Error(1)
}
