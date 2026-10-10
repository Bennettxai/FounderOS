package linear

// ============================================================================
// Helper Types
// ============================================================================

// syncStats tracks the number of items created and updated during a sync operation.
type syncStats struct {
	Created int
	Updated int
}

// ============================================================================
// Helper Functions
// ============================================================================

// containsString checks if a string slice contains a specific string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
