package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config manages homelab setup configuration
type Config struct {
	filePath string
	data     map[string]string
	loaded   bool // Track if configuration has been loaded from disk
}

// New creates a new Config instance
func New(filePath string) *Config {
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/var/home/core" // Fallback for CoreOS
		}
		filePath = filepath.Join(home, ".homelab-setup.conf")
	}

	return &Config{
		filePath: filePath,
		data:     make(map[string]string),
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

// Save writes configuration to file
func (c *Config) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.OpenFile(c.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Write header
	fmt.Fprintln(file, "# UBlue uCore Homelab Setup Configuration")
	fmt.Fprintf(file, "# Generated: %s\n", os.Getenv("USER"))
	fmt.Fprintln(file, "")

	// Write key-value pairs
	for key, value := range c.data {
		fmt.Fprintf(file, "%s=%s\n", key, value)
	}

	// Explicitly check close error to prevent data loss
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}

	return nil
}

// Get retrieves a configuration value
func (c *Config) Get(key string) (string, error) {
	value, exists := c.data[key]
	if !exists {
		return "", fmt.Errorf("config key not found: %s", key)
	}
	return value, nil
}

// GetOrDefault retrieves a value or returns default if not found
func (c *Config) GetOrDefault(key, defaultValue string) string {
	if value, exists := c.data[key]; exists {
		return value
	}
	return defaultValue
}

// Set sets a configuration value
// Automatically loads existing configuration if not already loaded to prevent data loss
func (c *Config) Set(key, value string) error {
	// Load existing configuration first to avoid overwriting
	if !c.loaded {
		if err := c.Load(); err != nil {
			return fmt.Errorf("failed to load existing config before set: %w", err)
		}
	}

	c.data[key] = value
	return c.Save()
}

// Exists checks if a key exists
func (c *Config) Exists(key string) bool {
	_, exists := c.data[key]
	return exists
}

// GetAll returns all configuration data
func (c *Config) GetAll() map[string]string {
	// Return a copy to prevent external modification
	result := make(map[string]string, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// Delete removes a configuration key
// Automatically loads existing configuration if not already loaded to prevent data loss
func (c *Config) Delete(key string) error {
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
