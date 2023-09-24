package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/syself/hrobot-go/models"
)

type RobotClient struct {
	mock.Mock
}

func (m *RobotClient) ServerGetList() ([]models.Server, error) {
	args := m.Called()
	return getRobotServers(args, 0), args.Error(1)
}

func (m *RobotClient) BootLinuxDelete(id int) (*models.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootLinuxGet(id int) (*models.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootLinuxSet(id int, input *models.LinuxSetInput) (*models.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueDelete(id int) (*models.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueGet(id int) (*models.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueSet(id int, input *models.RescueSetInput) (*models.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) FailoverGet(ip string) (*models.Failover, error) {
	panic("this method should not be called")
}
func (m *RobotClient) FailoverGetList() ([]models.Failover, error) {
	panic("this method should not be called")
}
func (m *RobotClient) GetVersion() string {
	panic("this method should not be called")
}
func (m *RobotClient) IPGetList() ([]models.IP, error) {
	panic("this method should not be called")
}
func (m *RobotClient) KeyGetList() ([]models.Key, error) {
	panic("this method should not be called")
}
func (m *RobotClient) KeySet(input *models.KeySetInput) (*models.Key, error) {
	panic("this method should not be called")
}
func (m *RobotClient) RDnsGet(ip string) (*models.Rdns, error) {
	panic("this method should not be called")
}
func (m *RobotClient) RDnsGetList() ([]models.Rdns, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ResetGet(id int) (*models.Reset, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ResetSet(id int, input *models.ResetSetInput) (*models.ResetPost, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerGet(id int) (*models.Server, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerReverse(id int) (*models.Cancellation, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerSetName(id int, input *models.ServerSetNameInput) (*models.Server, error) {
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
