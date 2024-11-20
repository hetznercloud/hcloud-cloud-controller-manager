package providerid

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// prefixCloud is the prefix for Cloud Server provider IDs.
	//
	// It MUST not be changed, otherwise existing nodes will not be recognized anymore.
	prefixCloud = "hcloud://"

	// prefixRobot is the prefix for Robot Server provider IDs.
	//
	// It MUST not be changed, otherwise existing nodes will not be recognized anymore.
	prefixRobot = "hrobot://"

	// prefixRobot is the prefix used by the Syself Fork for Robot Server provider IDs.
	// This Prefix is no longer used for new nodes, instead [prefixRobot] should be used.
	//
	// It MUST not be changed, otherwise existing nodes will not be recognized anymore.
	prefixRobotLegacy = "hcloud://bm-"
)

type UnkownPrefixError struct {
	ProviderID string
}

func (e *UnkownPrefixError) Error() string {
	return fmt.Sprintf(
		"Provider ID does not have one of the the expected prefixes (%s, %s, %s): %s",
		prefixCloud,
		prefixRobot,
		prefixRobotLegacy,
		e.ProviderID,
	)
}

// ToServerID parses the Cloud or Robot Server ID from a ProviderID.
//
// This method supports all formats for the ProviderID that were ever used.
// If a format is ever dropped from this method the Nodes that still use that
// format will get abandoned and can no longer be processed by HCCM.
func ToServerID(providerID string) (id int64, isCloudServer bool, err error) {
	idString := ""
	switch {
	case strings.HasPrefix(providerID, prefixRobot):
		idString = strings.ReplaceAll(providerID, prefixRobot, "")

	case strings.HasPrefix(providerID, prefixRobotLegacy):
		// This case needs to be before [prefixCloud], as [prefixCloud] is a superset of [prefixRobotLegacy]
		idString = strings.ReplaceAll(providerID, prefixRobotLegacy, "")

	case strings.HasPrefix(providerID, prefixCloud):
		isCloudServer = true
		idString = strings.ReplaceAll(providerID, prefixCloud, "")

	default:
		return 0, false, &UnkownPrefixError{providerID}
	}

	if idString == "" {
		return 0, false, fmt.Errorf("providerID is missing a serverID: %s", providerID)
	}

	id, err = strconv.ParseInt(idString, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("unable to parse server id: %s", providerID)
	}
	return id, isCloudServer, nil
}

// FromCloudServerID generates the canonical ProviderID for a Cloud Server.
func FromCloudServerID(serverID int64) string {
	return fmt.Sprintf("%s%d", prefixCloud, serverID)
}

// FromRobotServerNumber generates the canonical ProviderID for a Robot Server.
func FromRobotServerNumber(serverNumber int) string {
	return fmt.Sprintf("%s%d", prefixRobot, serverNumber)
}
