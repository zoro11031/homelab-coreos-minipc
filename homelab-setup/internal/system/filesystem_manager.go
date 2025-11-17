package system

import "os"

// FileSystemManager defines the interface for file system operations.
// This allows for mocking the file system in tests.
type FileSystemManager interface {
	WriteFile(path string, content []byte, perms os.FileMode) error
	EnsureDirectory(path string, owner string, perms os.FileMode) error
}
