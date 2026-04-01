package config

import (
	"errors"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/envutil"
)

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
