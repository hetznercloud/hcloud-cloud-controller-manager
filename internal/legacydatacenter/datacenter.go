package legacydatacenter

// NameFromLocation maps location name to legacy cloud datacenter names, which
// are deprecated in the API. For new locations we will return the location
// name as a datacenter name (`topology.kubernetes.io/zone`).
func NameFromLocation(location string) string {
	switch location {
	case "nbg1":
		return "nbg1-dc3"
	case "hel1":
		return "hel1-dc2"
	case "fsn1":
		return "fsn1-dc14"
	case "ash":
		return "ash-dc1"
	case "hil":
		return "hil-dc1"
	case "sin":
		return "sin-dc1"
	default:
		return location
	}
}
