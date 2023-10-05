package mocks

import (
	"github.com/stretchr/testify/mock"
	hrobotmodels "github.com/syself/hrobot-go/models"
)

type RobotClient struct {
	mock.Mock
}

func (m *RobotClient) ServerGetList() ([]hrobotmodels.Server, error) {
	args := m.Called()
	return getRobotServers(args, 0), args.Error(1)
}

func (m *RobotClient) BootLinuxDelete(id int) (*hrobotmodels.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootLinuxGet(id int) (*hrobotmodels.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootLinuxSet(id int, input *hrobotmodels.LinuxSetInput) (*hrobotmodels.Linux, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueDelete(id int) (*hrobotmodels.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueGet(id int) (*hrobotmodels.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) BootRescueSet(id int, input *hrobotmodels.RescueSetInput) (*hrobotmodels.Rescue, error) {
	panic("this method should not be called")
}
func (m *RobotClient) FailoverGet(ip string) (*hrobotmodels.Failover, error) {
	panic("this method should not be called")
}
func (m *RobotClient) FailoverGetList() ([]hrobotmodels.Failover, error) {
	panic("this method should not be called")
}
func (m *RobotClient) GetVersion() string {
	panic("this method should not be called")
}
func (m *RobotClient) IPGetList() ([]hrobotmodels.IP, error) {
	panic("this method should not be called")
}
func (m *RobotClient) KeyGetList() ([]hrobotmodels.Key, error) {
	panic("this method should not be called")
}
func (m *RobotClient) KeySet(input *hrobotmodels.KeySetInput) (*hrobotmodels.Key, error) {
	panic("this method should not be called")
}
func (m *RobotClient) RDnsGet(ip string) (*hrobotmodels.Rdns, error) {
	panic("this method should not be called")
}
func (m *RobotClient) RDnsGetList() ([]hrobotmodels.Rdns, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ResetGet(id int) (*hrobotmodels.Reset, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ResetSet(id int, input *hrobotmodels.ResetSetInput) (*hrobotmodels.ResetPost, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerGet(id int) (*hrobotmodels.Server, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerReverse(id int) (*hrobotmodels.Cancellation, error) {
	panic("this method should not be called")
}
func (m *RobotClient) ServerSetName(id int, input *hrobotmodels.ServerSetNameInput) (*hrobotmodels.Server, error) {
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
