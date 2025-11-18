// Package common provides shared utilities and validation functions used across
// the homelab setup tool. This includes security-critical input validation
// (paths, usernames) that prevents command injection and path traversal attacks.
package common

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath validates that a path is absolute
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

// ValidateSafePath validates a path is absolute and contains no shell metacharacters
// This provides defense-in-depth against command injection when paths are used in system commands
func ValidateSafePath(path string) error {
	// First validate it's a valid absolute path
	if err := ValidatePath(path); err != nil {
		return err
	}

	// Check for shell metacharacters that could be exploited
	// Even though we use exec.Command which doesn't use a shell,
	// this provides defense-in-depth protection
	forbiddenChars := []string{
		";",  // Command separator
		"&",  // Background/AND operator
		"|",  // Pipe operator
		"$",  // Variable expansion
		"`",  // Command substitution
		"(",  // Subshell
		")",  // Subshell
		"<",  // Redirection
		">",  // Redirection
		"\n", // Newline
		"\r", // Carriage return
		"*",  // Glob wildcard
		"?",  // Glob wildcard
		"[",  // Glob wildcard
		"]",  // Glob wildcard
		"{",  // Brace expansion
		"}",  // Brace expansion
	}

	for _, char := range forbiddenChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains forbidden shell metacharacter '%s': %s", char, path)
		}
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	return nil
}

// ValidateUsername validates a Unix username
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Basic username validation (alphanumeric, underscore, hyphen, must start with letter or underscore)
	if len(username) > 32 {
		return fmt.Errorf("username too long (max 32 characters): %s", username)
	}

	firstChar := username[0]
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z') || firstChar == '_') {
		return fmt.Errorf("username must start with a letter or underscore: %s", username)
	}

	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return fmt.Errorf("username contains invalid character: %s", username)
		}
	}

	return nil
}
