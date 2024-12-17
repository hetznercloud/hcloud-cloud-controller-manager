package client

import "github.com/syself/hrobot-go/models"

type Client interface {
	ServerGet(id int) (*models.Server, error)
	ServerGetList() ([]models.Server, error)
	SetCredentials(username, password string) error
}
