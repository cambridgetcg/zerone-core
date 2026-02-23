package types

// IsValidServiceType returns true if the service type is recognized.
func IsValidServiceType(s string) bool {
	switch s {
	case "inference", "verification", "storage":
		return true
	}
	return false
}
