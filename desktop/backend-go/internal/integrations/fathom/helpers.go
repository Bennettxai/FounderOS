package fathom

// syncStats tracks statistics from sync operations.
type syncStats struct {
	Created int
	Updated int
}

// containsString checks if a string slice contains a specific string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
