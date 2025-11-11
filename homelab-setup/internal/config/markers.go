package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Markers manages completion marker files
type Markers struct {
	dir string
}

// NewMarkers creates a new Markers instance
func NewMarkers(dir string) *Markers {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/var/home/core" // Fallback for CoreOS
		}
		dir = filepath.Join(home, ".local", "homelab-setup")
	}

	return &Markers{
		dir: dir,
	}
}

// Create creates a marker file
func (m *Markers) Create(name string) error {
	// Ensure directory exists
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return fmt.Errorf("failed to create marker directory: %w", err)
	}

	markerPath := filepath.Join(m.dir, name)
	file, err := os.Create(markerPath)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}
	defer file.Close()

	return nil
}

// Exists checks if a marker file exists
// Returns (exists, error) where error indicates a problem checking (e.g., permission denied)
// If error is not nil, the exists value should not be trusted
func (m *Markers) Exists(name string) (bool, error) {
	markerPath := filepath.Join(m.dir, name)
	_, err := os.Stat(markerPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Other error (permission denied, I/O error, etc.)
	return false, fmt.Errorf("failed to check marker existence: %w", err)
}

// Remove deletes a marker file
func (m *Markers) Remove(name string) error {
	markerPath := filepath.Join(m.dir, name)
	err := os.Remove(markerPath)
	if os.IsNotExist(err) {
		return nil // Not an error if it doesn't exist
	}
	return err
}

// RemoveAll removes all marker files
func (m *Markers) RemoveAll() error {
	if _, err := os.Stat(m.dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to remove
	}

	return os.RemoveAll(m.dir)
}

// List returns all marker names
func (m *Markers) List() ([]string, error) {
	if _, err := os.Stat(m.dir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(m.dir)
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

// Dir returns the marker directory path
func (m *Markers) Dir() string {
	return m.dir
}
