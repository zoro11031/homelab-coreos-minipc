// Package config provides thread-safe configuration management for the homelab
// setup tool. It handles both persistent configuration storage (key-value pairs
// in a config file) and completion markers (files indicating completed setup steps).
// All operations are atomic and safe for concurrent use.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config manages homelab setup configuration and completion markers with thread-safe operations
type Config struct {
	filePath  string
	markerDir string
	data      map[string]string
	loaded    bool // Track if configuration has been loaded from disk
	mu        sync.RWMutex
}

// ensureLoaded loads configuration data from disk once before read operations.
// This method must only be called while holding c.mu.RLock or c.mu.Lock.
// The c.loaded check happens inside the caller's lock to prevent race conditions.
func (c *Config) ensureLoaded() error {
	if c.loaded {
		return nil
	}
	return c.Load()
}

// New creates a new Config instance
func New(filePath string) *Config {
	var markerDir string
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/var/home/core" // Fallback for CoreOS
		}
		filePath = filepath.Join(home, ".homelab-setup.conf")
		markerDir = filepath.Join(home, ".local", "homelab-setup")
	} else {
		// If custom config path provided, use adjacent directory for markers
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/var/home/core"
		}
		markerDir = filepath.Join(home, ".local", "homelab-setup")
	}

	return &Config{
		filePath:  filePath,
		markerDir: markerDir,
		data:      make(map[string]string),
	}
}

// Load reads configuration from file
func (c *Config) Load() error {
	// If file doesn't exist, that's okay - we'll create it on Save
	if _, err := os.Stat(c.filePath); os.IsNotExist(err) {
		c.loaded = true
		return nil
	}

	file, err := os.Open(c.filePath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			c.data[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	c.loaded = true
	return nil
}

// Save writes configuration to file using atomic write pattern
// This prevents data loss if the write operation fails midway
func (c *Config) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create temporary file in the same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, ".homelab-setup.conf.tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Cleanup on error

	// Set proper permissions on temp file
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Write header
	fmt.Fprintln(tmpFile, "# UBlue uCore Homelab Setup Configuration")
	fmt.Fprintf(tmpFile, "# Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(tmpFile, "")

	// Write key-value pairs
	for key, value := range c.data {
		fmt.Fprintf(tmpFile, "%s=%s\n", key, value)
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Explicitly check close error to prevent data loss
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename - if this succeeds, the old config is replaced atomically
	if err := os.Rename(tmpPath, c.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file to config: %w", err)
	}

	return nil
}

// Get retrieves a configuration value (thread-safe)
func (c *Config) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := c.ensureLoaded(); err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	value, exists := c.data[key]
	if !exists {
		return "", fmt.Errorf("config key not found: %s", key)
	}
	return value, nil
}

// GetOrDefault retrieves a value or returns default if not found (thread-safe)
// First checks the config, then the Defaults table, then the provided fallback
func (c *Config) GetOrDefault(key, defaultValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := c.ensureLoaded(); err != nil {
		return defaultValue
	}
	if value, exists := c.data[key]; exists {
		return value
	}
	// Check the defaults table
	if tableDefault, exists := Defaults[key]; exists {
		return tableDefault
	}
	return defaultValue
}

// Set sets a configuration value (thread-safe)
// Automatically loads existing configuration if not already loaded to prevent data loss
func (c *Config) Set(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Load existing configuration first to avoid overwriting
	// Note: We're holding c.mu.Lock, so calling c.Load() directly is safe
	if !c.loaded {
		if err := c.Load(); err != nil {
			return fmt.Errorf("failed to load existing config before set: %w", err)
		}
	}

	c.data[key] = value
	return c.Save()
}

// Exists checks if a key exists (thread-safe)
func (c *Config) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := c.ensureLoaded(); err != nil {
		return false
	}
	_, exists := c.data[key]
	return exists
}

// GetAll returns all configuration data (thread-safe)
func (c *Config) GetAll() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := c.ensureLoaded(); err != nil {
		return map[string]string{}
	}
	// Return a copy to prevent external modification
	result := make(map[string]string, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// Delete removes a configuration key (thread-safe)
// Automatically loads existing configuration if not already loaded to prevent data loss
func (c *Config) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Load existing configuration first to avoid overwriting
	if !c.loaded {
		if err := c.Load(); err != nil {
			return fmt.Errorf("failed to load existing config before delete: %w", err)
		}
	}

	delete(c.data, key)
	return c.Save()
}

// FilePath returns the configuration file path
func (c *Config) FilePath() string {
	return c.filePath
}

// ===== Marker Management Methods =====

// validateMarkerName ensures the marker name is safe and doesn't contain path traversal characters
func validateMarkerName(name string) error {
	if name == "" {
		return fmt.Errorf("marker name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("marker name cannot contain path separators: %s", name)
	}
	if name == ".." || name == "." {
		return fmt.Errorf("marker name cannot be '.' or '..': %s", name)
	}
	return nil
}

// MarkComplete creates a completion marker file (idempotent)
func (c *Config) MarkComplete(name string) error {
	if err := validateMarkerName(name); err != nil {
		return err
	}

	if err := os.MkdirAll(c.markerDir, 0755); err != nil {
		return fmt.Errorf("failed to create marker directory: %w", err)
	}

	markerPath := filepath.Join(c.markerDir, name)
	file, err := os.Create(markerPath)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}
	defer file.Close()

	return nil
}

// MarkCompleteIfNotExists atomically creates a marker only if it doesn't exist
// Returns (wasCreated, error) where wasCreated indicates if this call created the marker
func (c *Config) MarkCompleteIfNotExists(name string) (bool, error) {
	if err := validateMarkerName(name); err != nil {
		return false, err
	}

	if err := os.MkdirAll(c.markerDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create marker directory: %w", err)
	}

	markerPath := filepath.Join(c.markerDir, name)
	file, err := os.OpenFile(markerPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to create marker file: %w", err)
	}
	defer file.Close()

	return true, nil
}

// IsComplete checks if a step completion marker exists
func (c *Config) IsComplete(name string) bool {
	if err := validateMarkerName(name); err != nil {
		return false
	}

	markerPath := filepath.Join(c.markerDir, name)
	_, err := os.Stat(markerPath)
	return err == nil
}

// ClearMarker removes a completion marker
func (c *Config) ClearMarker(name string) error {
	if err := validateMarkerName(name); err != nil {
		return err
	}

	markerPath := filepath.Join(c.markerDir, name)
	err := os.Remove(markerPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ClearAllMarkers removes all marker files
func (c *Config) ClearAllMarkers() error {
	if _, err := os.Stat(c.markerDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(c.markerDir)
}

// ListMarkers returns all marker names
func (c *Config) ListMarkers() ([]string, error) {
	if _, err := os.Stat(c.markerDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(c.markerDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read marker directory: %w", err)
	}

	var markers []string
	for _, entry := range entries {
		if !entry.IsDir() {
			markers = append(markers, entry.Name())
		}
	}

	return markers, nil
}

// MarkerDir returns the marker directory path
func (c *Config) MarkerDir() string {
	return c.markerDir
}
