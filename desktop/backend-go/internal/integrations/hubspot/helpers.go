package hubspot

// syncStats tracks sync operation results.
type syncStats struct {
	Created int
	Updated int
}

// containsString checks if a string exists in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
