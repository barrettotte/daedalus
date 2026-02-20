package daedalus

import (
	"fmt"
	"strings"
)

// ValidateListName trims whitespace and validates a list directory name.
// Rejects empty names, path separators, traversal sequences, hidden dirs, and reserved names.
// Returns the cleaned name or an error.
func ValidateListName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("list name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid list name")
	}
	if strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("list name cannot start with '.'")
	}
	if name == "_assets" {
		return "", fmt.Errorf("list name cannot be '_assets'")
	}
	return name, nil
}
