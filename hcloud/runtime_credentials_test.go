package hcloud

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestNewRuntimeCredentialsWithoutFileSources(t *testing.T) {
	defer unsetEnv(t, "HCLOUD_TOKEN_FILE")()
	defer unsetEnv(t, "ROBOT_USER_FILE")()
	defer unsetEnv(t, "ROBOT_PASSWORD_FILE")()

	resetEnv := testsupport.Setenv(t,
		"HCLOUD_TOKEN", "token-1",
		"ROBOT_USER", "robot-user-1",
		"ROBOT_PASSWORD", "robot-password-1",
	)
	defer resetEnv()

	credentials, err := newRuntimeCredentials()
	assert.NoError(t, err)
	assert.Nil(t, credentials.watcher)
	assert.Equal(t, "Bearer token-1", credentials.hcloudAuthorization())
	assert.Equal(t, "robot-user-1", credentials.robotUser)
	assert.Equal(t, "robot-password-1", credentials.robotPass)
	assert.NoError(t, credentials.close())
}

func TestNewRuntimeCredentialsRejectsInvalidAuthorizationToken(t *testing.T) {
	defer unsetEnv(t, "HCLOUD_TOKEN_FILE")()
	defer unsetEnv(t, "ROBOT_USER")()
	defer unsetEnv(t, "ROBOT_PASSWORD")()
	defer unsetEnv(t, "ROBOT_USER_FILE")()
	defer unsetEnv(t, "ROBOT_PASSWORD_FILE")()

	resetEnv := testsupport.Setenv(t, "HCLOUD_TOKEN", "token\ninvalid")
	defer resetEnv()

	_, err := newRuntimeCredentials()
	assert.EqualError(t, err, invalidAuthorizationTokenError)
}

func TestNewRuntimeCredentialsRejectsMissingMountedSecret(t *testing.T) {
	defer unsetEnv(t, "HCLOUD_TOKEN")()
	defer unsetEnv(t, "ROBOT_USER")()
	defer unsetEnv(t, "ROBOT_PASSWORD")()
	defer unsetEnv(t, "ROBOT_USER_FILE")()
	defer unsetEnv(t, "ROBOT_PASSWORD_FILE")()

	resetEnv := testsupport.Setenv(t, "HCLOUD_TOKEN_FILE", filepath.Join(t.TempDir(), "missing"))
	defer resetEnv()

	_, err := newRuntimeCredentials()
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestRuntimeCredentialsLoadRobotCredentialsErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		userFile := filepath.Join(dir, "robot-user")
		assert.NoError(t, os.WriteFile(userFile, []byte("robot-user-1"), 0o600))

		credentials := &runtimeCredentials{
			robotUserPath: userFile,
			robotPassPath: filepath.Join(dir, "missing"),
		}

		_, _, err := credentials.loadRobotCredentials()
		assert.ErrorContains(t, err, "no such file or directory")
	})

	t.Run("partial credentials", func(t *testing.T) {
		dir := t.TempDir()
		userFile := filepath.Join(dir, "robot-user")
		passwordFile := filepath.Join(dir, "robot-password")
		assert.NoError(t, os.WriteFile(userFile, []byte("robot-user-1"), 0o600))
		assert.NoError(t, os.WriteFile(passwordFile, []byte(""), 0o600))

		credentials := &runtimeCredentials{
			robotUserPath: userFile,
			robotPassPath: passwordFile,
		}

		_, _, err := credentials.loadRobotCredentials()
		assert.EqualError(t, err, `both "ROBOT_USER" and "ROBOT_PASSWORD" must be provided, or neither`)
	})
}

func TestRuntimeCredentialsReloadKeepsPreviousValuesOnErrors(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "hcloud-token")
	userFile := filepath.Join(dir, "robot-user")
	passwordFile := filepath.Join(dir, "robot-password")

	assert.NoError(t, os.WriteFile(tokenFile, []byte("token\ninvalid"), 0o600))
	assert.NoError(t, os.WriteFile(userFile, []byte("robot-user-2"), 0o600))
	assert.NoError(t, os.WriteFile(passwordFile, []byte(""), 0o600))

	credentials := &runtimeCredentials{
		hcloudToken:     "token-1",
		robotUser:       "robot-user-1",
		robotPass:       "robot-password-1",
		hcloudTokenPath: tokenFile,
		robotUserPath:   userFile,
		robotPassPath:   passwordFile,
	}

	credentials.reload()

	assert.Equal(t, "Bearer token-1", credentials.hcloudAuthorization())
	user, password := credentials.robotCredentials()
	assert.Equal(t, "robot-user-1", user)
	assert.Equal(t, "robot-password-1", password)
}

func TestCredentialReloadersClearAuthorizationHeadersWhenCredentialsAreEmpty(t *testing.T) {
	captured := make(chan string, 2)
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		captured <- req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "stale")

	_, err = newHCloudCredentialReloader(&runtimeCredentials{}, next).RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, "", <-captured)

	req, err = http.NewRequest(http.MethodGet, "https://example.com", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "stale")

	_, err = newRobotCredentialReloader(&runtimeCredentials{}, next).RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, "", <-captured)
}

func TestTransportOrDefault(t *testing.T) {
	custom := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("created")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	assert.Same(t, http.DefaultTransport, transportOrDefault(nil))

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	assert.NoError(t, err)

	resp, err := transportOrDefault(custom).RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
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
