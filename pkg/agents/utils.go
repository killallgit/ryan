package agents

import (
	"fmt"
	"time"
)

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
