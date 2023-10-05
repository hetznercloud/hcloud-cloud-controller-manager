package mocks

import (
	"github.com/stretchr/testify/mock"
	robotmodels "github.com/syself/hrobot-go/models"
)

type RobotClient struct {
	mock.Mock
}

func (m *RobotClient) ServerGetList() ([]robotmodels.Server, error) {
	args := m.Called()
	return getRobotServers(args, 0), args.Error(1)
}

func (m *RobotClient) BootLinuxDelete(id int) (*robotmodels.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootLinuxGet(id int) (*robotmodels.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootLinuxSet(id int, input *robotmodels.LinuxSetInput) (*robotmodels.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueDelete(id int) (*robotmodels.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueGet(id int) (*robotmodels.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueSet(id int, input *robotmodels.RescueSetInput) (*robotmodels.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) FailoverGet(ip string) (*robotmodels.Failover, error) {
	panic("this method should not be called")
}
func (m *RobotClient) FailoverGetList() ([]robotmodels.Failover, error) {
	panic("this method should not be called")
}
func (m *RobotClient) GetVersion() string {
	panic("this method should not be called")
}
func (m *RobotClient) IPGetList() ([]robotmodels.IP, error) {
	panic("this method should not be called")
}
func (m *RobotClient) KeyGetList() ([]robotmodels.Key, error) {
	panic("this method should not be called")
}
func (m *RobotClient) KeySet(input *robotmodels.KeySetInput) (*robotmodels.Key, error) {
	panic("this method should not be called")
}
func (m *RobotClient) RDnsGet(ip string) (*robotmodels.Rdns, error) {
	panic("this method should not be called")
}
func (m *RobotClient) RDnsGetList() ([]robotmodels.Rdns, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ResetGet(id int) (*robotmodels.Reset, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ResetSet(id int, input *robotmodels.ResetSetInput) (*robotmodels.ResetPost, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerGet(id int) (*robotmodels.Server, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerReverse(id int) (*robotmodels.Cancellation, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerSetName(id int, input *robotmodels.ServerSetNameInput) (*robotmodels.Server, error) {
	panic("this method should not be called")
}
func (m *RobotClient) SetBaseURL(baseURL string) {
	panic("this method should not be called")
}
func (m *RobotClient) SetUserAgent(userAgent string) {
	panic("this method should not be called")
}
func (m *RobotClient) ValidateCredentials() error {
	panic("this method should not be called")
}
