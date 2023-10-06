package providerid

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// providerPrefix is the prefix for all provider IDs. It MUST not be changed,
	// otherwise existing nodes will not be recognized anymore.
	providerPrefix = "hcloud://"
)

// ToServerID converts a ProviderID to a server ID.
func ToServerID(providerID string) (int64, error) {
	if !strings.HasPrefix(providerID, providerPrefix) {
		return 0, fmt.Errorf("providerID does not have the expected prefix %s: %s", providerPrefix, providerID)
	}

	idString := strings.ReplaceAll(providerID, providerPrefix, "")
	if idString == "" {
		return 0, fmt.Errorf("providerID is missing a serverID: %s", providerID)
	}

	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse server id: %s", providerID)
	}
	return id, nil
}

// FromServerID converts a server ID to a ProviderID.
func FromServerID(serverID int64) string {
	return fmt.Sprintf("%s%d", providerPrefix, serverID)
}
