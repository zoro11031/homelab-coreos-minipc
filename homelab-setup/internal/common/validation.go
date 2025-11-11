package common

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

// ValidateIP validates an IPv4 address
func ValidateIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	// Ensure it's IPv4
	if parsed.To4() == nil {
		return fmt.Errorf("not a valid IPv4 address: %s", ip)
	}

	return nil
}

// ValidatePort validates a port number (1-65535)
func ValidatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", port)
	}

	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", p)
	}

	return nil
}

// ValidatePath validates that a path is absolute
func ValidatePath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
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

// ValidateNotEmpty validates that a string is not empty
func ValidateNotEmpty(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

// ValidateDomain validates a domain name (basic validation)
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Basic domain validation - allow alphanumeric, dots, and hyphens
	if len(domain) > 253 {
		return fmt.Errorf("domain name too long: %s", domain)
	}

	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid domain (empty label): %s", domain)
		}
		if len(part) > 63 {
			return fmt.Errorf("domain label too long: %s", part)
		}

		for i, c := range part {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return fmt.Errorf("invalid character in domain: %s", domain)
			}
			// Hyphen cannot be at start or end
			if c == '-' && (i == 0 || i == len(part)-1) {
				return fmt.Errorf("domain label cannot start or end with hyphen: %s", part)
			}
		}
	}

	return nil
}

// ValidateTimezone validates a timezone string (basic check)
func ValidateTimezone(tz string) error {
	if tz == "" {
		return fmt.Errorf("timezone cannot be empty")
	}

	// Basic validation - should contain a slash and reasonable length
	if !strings.Contains(tz, "/") {
		return fmt.Errorf("invalid timezone format (should be Region/City): %s", tz)
	}

	if len(tz) > 64 {
		return fmt.Errorf("timezone string too long: %s", tz)
	}

	return nil
}
