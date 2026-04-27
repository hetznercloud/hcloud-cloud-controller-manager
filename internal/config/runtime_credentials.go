package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/envutil"
)

// RuntimeCredentialFiles contains file-backed credential sources that can be
// watched for runtime reloads.
type RuntimeCredentialFiles struct {
	HCloudToken   string
	RobotUser     string
	RobotPassword string
}

// HasAnyFilePaths reports whether any runtime credential file path is
// configured.
func (f RuntimeCredentialFiles) HasAnyFilePaths() bool {
	return f.HCloudToken != "" || f.RobotUser != "" || f.RobotPassword != ""
}

// Directories returns the unique parent directories of the configured
// credential files.
func (f RuntimeCredentialFiles) Directories() []string {
	return uniqueDirectories(f.HCloudToken, f.RobotUser, f.RobotPassword)
}

// LookupHCloudToken reads the current HCLOUD_TOKEN / HCLOUD_TOKEN_FILE value.
func LookupHCloudToken() (string, error) {
	return envutil.LookupEnvWithFile(hcloudToken)
}

// LookupRobotCredentials reads the current ROBOT_USER / ROBOT_USER_FILE and
// ROBOT_PASSWORD / ROBOT_PASSWORD_FILE values.
func LookupRobotCredentials() (string, string, error) {
	user, userErr := envutil.LookupEnvWithFile(robotUser)
	password, passwordErr := envutil.LookupEnvWithFile(robotPassword)
	if userErr != nil || passwordErr != nil {
		return "", "", errors.Join(userErr, passwordErr)
	}
	if (user == "") != (password == "") {
		return "", "", fmt.Errorf("both %q and %q must be provided, or neither", robotUser, robotPassword)
	}
	return user, password, nil
}

// LookupRuntimeCredentialFiles returns the file-backed credential sources that
// can be watched for hot reloads. Plain environment variables take precedence
// over file-backed sources and are therefore not watched.
func LookupRuntimeCredentialFiles() RuntimeCredentialFiles {
	return RuntimeCredentialFiles{
		HCloudToken:   lookupCredentialFile(hcloudToken),
		RobotUser:     lookupCredentialFile(robotUser),
		RobotPassword: lookupCredentialFile(robotPassword),
	}
}

// ReadCredentialFile reads a mounted credential file and trims surrounding
// whitespace to match envutil.LookupEnvWithFile semantics.
func ReadCredentialFile(path string) (string, error) {
	valueBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(valueBytes)), nil
}

func lookupCredentialFile(key string) string {
	if _, ok := os.LookupEnv(key); ok {
		return ""
	}
	return os.Getenv(key + "_FILE")
}

func uniqueDirectories(paths ...string) []string {
	dirs := map[string]struct{}{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		dirs[filepath.Dir(path)] = struct{}{}
	}

	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}
	return result
}
