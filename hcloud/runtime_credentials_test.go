package hcloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	hrobot "github.com/syself/hrobot-go"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func TestHCloudClientReloadsTokenFromFile(t *testing.T) {
	defer unsetEnv(t, "HCLOUD_TOKEN")()

	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		assert.NoError(t, json.NewEncoder(w).Encode(schema.LocationListResponse{Locations: []schema.Location{}}))
	}))
	defer server.Close()

	tokenFile := filepath.Join(t.TempDir(), "hcloud-token")
	assert.NoError(t, os.WriteFile(tokenFile, []byte("token-1"), 0o600))

	resetEnv := testsupport.Setenv(t, "HCLOUD_TOKEN_FILE", tokenFile)
	defer resetEnv()

	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithHTTPClient(newHCloudHTTPClient(0)),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	)

	_, _, err := client.Location.List(t.Context(), hcloud.LocationListOpts{})
	assert.NoError(t, err)

	assert.NoError(t, os.WriteFile(tokenFile, []byte("token-2"), 0o600))

	_, _, err = client.Location.List(t.Context(), hcloud.LocationListOpts{})
	assert.NoError(t, err)

	assert.Equal(t, []string{"Bearer token-1", "Bearer token-2"}, authorizations)
}

func TestRobotClientReloadsCredentialsFromFile(t *testing.T) {
	defer unsetEnv(t, "ROBOT_USER")()
	defer unsetEnv(t, "ROBOT_PASSWORD")()

	var users []string
	var passwords []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		assert.True(t, ok)
		users = append(users, user)
		passwords = append(passwords, password)
		assert.NoError(t, json.NewEncoder(w).Encode([]map[string]any{
			{
				"server": map[string]any{
					"server_number": 1,
					"server_name":   "node-1",
					"server_ip":     "192.0.2.1",
				},
			},
		}))
	}))
	defer server.Close()

	dir := t.TempDir()
	userFile := filepath.Join(dir, "robot-user")
	passwordFile := filepath.Join(dir, "robot-password")
	assert.NoError(t, os.WriteFile(userFile, []byte("robot-user-1"), 0o600))
	assert.NoError(t, os.WriteFile(passwordFile, []byte("robot-password-1"), 0o600))

	resetEnv := testsupport.Setenv(t,
		"ROBOT_USER_FILE", userFile,
		"ROBOT_PASSWORD_FILE", passwordFile,
	)
	defer resetEnv()

	client := hrobot.NewBasicAuthClientWithCustomHttpClient("stale-user", "stale-password", newRobotHTTPClient(0))
	client.SetBaseURL(server.URL)

	_, err := client.ServerGetList()
	assert.NoError(t, err)

	assert.NoError(t, os.WriteFile(userFile, []byte("robot-user-2"), 0o600))
	assert.NoError(t, os.WriteFile(passwordFile, []byte("robot-password-2"), 0o600))

	_, err = client.ServerGetList()
	assert.NoError(t, err)

	assert.Equal(t, []string{"robot-user-1", "robot-user-2"}, users)
	assert.Equal(t, []string{"robot-password-1", "robot-password-2"}, passwords)
}

func unsetEnv(t *testing.T, key string) func() {
	t.Helper()

	value, ok := os.LookupEnv(key)
	assert.NoError(t, os.Unsetenv(key))

	return func() {
		if !ok {
			assert.NoError(t, os.Unsetenv(key))
			return
		}
		assert.NoError(t, os.Setenv(key, value))
	}
}
