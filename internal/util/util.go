package util

import (
	"fmt"
	"os"
	"time"
)

// GetEnvDuration returns the duration parsed from the environment variable with the given key and a potential error
// parsing the var. Returns false if the env var is unset.
func GetEnvDuration(key string) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return 0, nil
	}

	b, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %v", key, err)
	}

	return b, nil
}
