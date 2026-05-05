package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
)

func TestRuntimeCredentialFilesHelpers(t *testing.T) {
	files := RuntimeCredentialFiles{}
	assert.False(t, files.HasAnyFilePaths())
	assert.Empty(t, files.Directories())

	files = RuntimeCredentialFiles{
		HCloudToken:   "/tmp/one/token",
		RobotUser:     "/tmp/two/user",
		RobotPassword: "/tmp/two/password",
	}

	assert.True(t, files.HasAnyFilePaths())
	assert.ElementsMatch(t, []string{"/tmp/one", "/tmp/two"}, files.Directories())
}

func TestLookupRuntimeCredentialFiles(t *testing.T) {
	defer unsetEnv(t, hcloudToken)()
	defer unsetEnv(t, robotUser)()
	defer unsetEnv(t, robotPassword)()

	resetEnv := testsupport.Setenv(t,
		hcloudToken+"_FILE", "/tmp/hcloud-token",
		robotUser+"_FILE", "/tmp/robot-user",
		robotPassword+"_FILE", "/tmp/robot-password",
	)
	defer resetEnv()

	files := LookupRuntimeCredentialFiles()
	assert.Equal(t, RuntimeCredentialFiles{
		HCloudToken:   "/tmp/hcloud-token",
		RobotUser:     "/tmp/robot-user",
		RobotPassword: "/tmp/robot-password",
	}, files)
}

func TestLookupRuntimeCredentialFilesIgnoresPlainEnvironmentVariables(t *testing.T) {
	resetEnv := testsupport.Setenv(t,
		hcloudToken, "token",
		hcloudToken+"_FILE", "/tmp/hcloud-token",
		robotUser, "robot-user",
		robotUser+"_FILE", "/tmp/robot-user",
		robotPassword, "robot-password",
		robotPassword+"_FILE", "/tmp/robot-password",
	)
	defer resetEnv()

	files := LookupRuntimeCredentialFiles()
	assert.Equal(t, RuntimeCredentialFiles{}, files)
	assert.Empty(t, lookupCredentialFile(hcloudToken))
}

func TestLookupRobotCredentials(t *testing.T) {
	resetEnv := testsupport.Setenv(t,
		robotUser, "robot-user-1",
		robotPassword, "robot-password-1",
	)
	defer resetEnv()

	user, password, err := LookupRobotCredentials()
	assert.NoError(t, err)
	assert.Equal(t, "robot-user-1", user)
	assert.Equal(t, "robot-password-1", password)
}

func TestLookupHCloudToken(t *testing.T) {
	resetEnv := testsupport.Setenv(t, hcloudToken, "token-1")
	defer resetEnv()

	token, err := LookupHCloudToken()
	assert.NoError(t, err)
	assert.Equal(t, "token-1", token)
}

func TestLookupRobotCredentialsRejectsPartialCredentials(t *testing.T) {
	defer unsetEnv(t, robotPassword)()

	resetEnv := testsupport.Setenv(t, robotUser, "robot-user-1")
	defer resetEnv()

	_, _, err := LookupRobotCredentials()
	assert.EqualError(t, err, `both "ROBOT_USER" and "ROBOT_PASSWORD" must be provided, or neither`)
}

func TestLookupRobotCredentialsJoinsFileErrors(t *testing.T) {
	defer unsetEnv(t, robotUser)()
	defer unsetEnv(t, robotPassword)()

	resetEnv := testsupport.Setenv(t,
		robotUser+"_FILE", filepath.Join(t.TempDir(), "missing-user"),
		robotPassword+"_FILE", filepath.Join(t.TempDir(), "missing-password"),
	)
	defer resetEnv()

	_, _, err := LookupRobotCredentials()
	assert.ErrorContains(t, err, "missing-user")
	assert.ErrorContains(t, err, "missing-password")
}

func TestReadCredentialFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credential")
	assert.NoError(t, os.WriteFile(path, []byte("  token-1\n"), 0o600))

	value, err := ReadCredentialFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "token-1", value)
}

func TestReadCredentialFileReturnsReadError(t *testing.T) {
	_, err := ReadCredentialFile(filepath.Join(t.TempDir(), "missing"))
	assert.ErrorContains(t, err, "no such file or directory")
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
