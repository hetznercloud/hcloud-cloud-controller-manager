package hcloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	hrobot "github.com/syself/hrobot-go"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func TestHCloudClientReloadsTokenFromMountedSecret(t *testing.T) {
	defer unsetEnv(t, "HCLOUD_TOKEN")()

	var mu sync.Mutex
	lastAuthorization := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		lastAuthorization = r.Header.Get("Authorization")
		mu.Unlock()
		assert.NoError(t, json.NewEncoder(w).Encode(schema.LocationListResponse{Locations: []schema.Location{}}))
	}))
	defer server.Close()

	tokenFile := filepath.Join(t.TempDir(), "hcloud-token")
	assert.NoError(t, os.WriteFile(tokenFile, []byte("token-1"), 0o600))

	resetEnv := testsupport.Setenv(t, "HCLOUD_TOKEN_FILE", tokenFile)
	defer resetEnv()

	credentials, err := newRuntimeCredentials()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, credentials.close())
	}()

	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithHTTPClient(newHCloudHTTPClient(0, credentials)),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	)

	_, _, err = client.Location.List(t.Context(), hcloud.LocationListOpts{})
	assert.NoError(t, err)

	mu.Lock()
	assert.Equal(t, "Bearer token-1", lastAuthorization)
	mu.Unlock()

	replaceFile(t, tokenFile, "token-2")

	assert.Eventually(t, func() bool {
		_, _, err = client.Location.List(t.Context(), hcloud.LocationListOpts{})
		if err != nil {
			return false
		}

		mu.Lock()
		defer mu.Unlock()
		return lastAuthorization == "Bearer token-2"
	}, 3*time.Second, 50*time.Millisecond)
}

func TestRobotClientReloadsCredentialsFromMountedSecret(t *testing.T) {
	defer unsetEnv(t, "ROBOT_USER")()
	defer unsetEnv(t, "ROBOT_PASSWORD")()

	var mu sync.Mutex
	lastUser := ""
	lastPassword := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		assert.True(t, ok)

		mu.Lock()
		lastUser = user
		lastPassword = password
		mu.Unlock()

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

	credentials, err := newRuntimeCredentials()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, credentials.close())
	}()

	client := hrobot.NewBasicAuthClientWithCustomHttpClient("stale-user", "stale-password", newRobotHTTPClient(0, credentials))
	client.SetBaseURL(server.URL)

	_, err = client.ServerGetList()
	assert.NoError(t, err)

	mu.Lock()
	assert.Equal(t, "robot-user-1", lastUser)
	assert.Equal(t, "robot-password-1", lastPassword)
	mu.Unlock()

	replaceFile(t, userFile, "robot-user-2")
	replaceFile(t, passwordFile, "robot-password-2")

	assert.Eventually(t, func() bool {
		_, err = client.ServerGetList()
		if err != nil {
			return false
		}

		mu.Lock()
		defer mu.Unlock()
		return lastUser == "robot-user-2" && lastPassword == "robot-password-2"
	}, 3*time.Second, 50*time.Millisecond)
}

func replaceFile(t *testing.T, path, content string) {
	t.Helper()

	tmpPath := path + ".tmp"
	assert.NoError(t, os.WriteFile(tmpPath, []byte(content), 0o600))
	assert.NoError(t, os.Rename(tmpPath, path))
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
