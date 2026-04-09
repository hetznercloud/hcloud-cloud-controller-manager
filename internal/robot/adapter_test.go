package robot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hrobot "github.com/syself/hrobot-go"
	hrobotmodels "github.com/syself/hrobot-go/models"
)

func TestNewClientNil(t *testing.T) {
	assert.Nil(t, NewClient(nil))
}

func TestAdapterServerGetListForceRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/robot/server", r.URL.Path)
		err := json.NewEncoder(w).Encode([]hrobotmodels.ServerResponse{
			{
				Server: hrobotmodels.Server{
					ServerNumber: 321,
					Name:         "robot-server1",
				},
			},
		})
		assert.NoError(t, err)
	}))
	defer server.Close()

	robotClient := hrobot.NewBasicAuthClient("", "")
	robotClient.SetBaseURL(server.URL + "/robot")

	client := NewClient(robotClient)
	require.NotNil(t, client)

	servers, err := client.ServerGetListForceRefresh("robot-server1")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "robot-server1", servers[0].Name)
}
