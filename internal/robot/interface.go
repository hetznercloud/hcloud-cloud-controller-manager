package robot

import (
	robotmodels "github.com/syself/hrobot-go/models"
)

type Client interface {
	ServerGet(id int) (*robotmodels.Server, error)
	ServerGetList() ([]robotmodels.Server, error)
	ResetGet(id int) (*robotmodels.Reset, error)
}
