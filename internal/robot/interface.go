package robot

import (
	hrobotmodels "github.com/syself/hrobot-go/models"
)

type Client interface {
	ServerGet(id int) (*hrobotmodels.Server, error)
	ServerGetList() ([]hrobotmodels.Server, error)
	ServerGetListForceRefresh(nodeName string) ([]hrobotmodels.Server, error)
	ResetGet(id int) (*hrobotmodels.Reset, error)
}
