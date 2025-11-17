package system

import (
	"os"
	"sync"
)

// MockFileSystem is a mock of the FileSystem for testing purposes.
// It captures written files in memory and implements FileSystemManager.
type MockFileSystem struct {
	FileSystem
	mu           sync.Mutex
	WrittenFiles map[string][]byte
}

// NewMockFileSystem creates a new MockFileSystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		WrittenFiles: make(map[string][]byte),
	}
}

// WriteFile captures the content that would be written to a file.
func (m *MockFileSystem) WriteFile(path string, content []byte, perms os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WrittenFiles[path] = content
	return nil
}

// EnsureDirectory is a mock implementation of FileSystemManager.EnsureDirectory.
func (m *MockFileSystem) EnsureDirectory(path string, owner string, perms os.FileMode) error {
	// In a mock, we might just log this or do nothing.
	// For this test, we don't need to capture directory creations, just file writes.
	return nil
}
