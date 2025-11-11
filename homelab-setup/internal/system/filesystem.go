package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// FileSystem handles file system operations
type FileSystem struct{}

// NewFileSystem creates a new FileSystem instance
func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

// EnsureDirectory creates a directory with specified owner and permissions
// If the directory already exists, it does nothing
func (fs *FileSystem) EnsureDirectory(path string, owner string, perms os.FileMode) error {
	// Check if directory exists
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", path)
		}
		// Directory exists, nothing to do
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory %s: %w", path, err)
	}

	// Create directory with sudo
	cmd := exec.Command("sudo", "-n", "mkdir", "-p", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create directory %s: %w\nOutput: %s", path, err, string(output))
	}

	// Set ownership if specified
	if owner != "" {
		if err := fs.Chown(path, owner); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", path, err)
		}
	}

	// Set permissions
	if err := fs.Chmod(path, perms); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", path, err)
	}

	return nil
}

// Chown changes the owner of a file or directory
// owner should be in format "user:group" or just "user"
func (fs *FileSystem) Chown(path string, owner string) error {
	cmd := exec.Command("sudo", "-n", "chown", owner, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chown %s to %s: %w\nOutput: %s", path, owner, err, string(output))
	}
	return nil
}

// ChownRecursive changes the owner of a file or directory recursively
func (fs *FileSystem) ChownRecursive(path string, owner string) error {
	cmd := exec.Command("sudo", "-n", "chown", "-R", owner, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chown -R %s to %s: %w\nOutput: %s", path, owner, err, string(output))
	}
	return nil
}

// Chmod changes the permissions of a file or directory
func (fs *FileSystem) Chmod(path string, perms os.FileMode) error {
	permStr := fmt.Sprintf("%o", perms)
	cmd := exec.Command("sudo", "-n", "chmod", permStr, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chmod %s to %s: %w\nOutput: %s", path, permStr, err, string(output))
	}
	return nil
}

// ChmodRecursive changes permissions recursively
func (fs *FileSystem) ChmodRecursive(path string, perms os.FileMode) error {
	permStr := fmt.Sprintf("%o", perms)
	cmd := exec.Command("sudo", "-n", "chmod", "-R", permStr, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to chmod -R %s to %s: %w\nOutput: %s", path, permStr, err, string(output))
	}
	return nil
}

// FileExists checks if a file exists
func (fs *FileSystem) FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check if file exists %s: %w", path, err)
}

// DirectoryExists checks if a directory exists
func (fs *FileSystem) DirectoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check if directory exists %s: %w", path, err)
}

// GetOwner returns the owner (user:group) of a file or directory
func (fs *FileSystem) GetOwner(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat %s: %w", path, err)
	}

	// Use type assertion with check to prevent panic
	sysInfo := info.Sys()
	if sysInfo == nil {
		return "", fmt.Errorf("failed to get system info for %s", path)
	}

	stat, ok := sysInfo.(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("failed to get stat info for %s: not a Unix filesystem", path)
	}

	uid := stat.Uid
	gid := stat.Gid

	return fmt.Sprintf("%d:%d", uid, gid), nil
}

// GetPermissions returns the permissions of a file or directory
func (fs *FileSystem) GetPermissions(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	return info.Mode().Perm(), nil
}

// RemoveDirectory removes a directory and all its contents
// Security note: This uses sudo rm -rf which is dangerous.
// Safety checks are in place to prevent accidental deletion of critical directories.
func (fs *FileSystem) RemoveDirectory(path string) error {
	// Safety checks to prevent accidental deletion of critical directories
	if path == "" {
		return fmt.Errorf("refusing to remove empty path")
	}

	// Ensure path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing to remove relative path: %s (must be absolute)", path)
	}

	// Block critical system directories
	criticalPaths := []string{
		"/",
		"/bin",
		"/boot",
		"/dev",
		"/etc",
		"/home",
		"/lib",
		"/lib64",
		"/proc",
		"/root",
		"/sbin",
		"/sys",
		"/usr",
		"/var",
	}

	for _, critical := range criticalPaths {
		if path == critical || strings.HasPrefix(path, critical+"/") {
			return fmt.Errorf("refusing to remove critical system path: %s", path)
		}
	}

	cmd := exec.Command("sudo", "-n", "rm", "-rf", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w\nOutput: %s", path, err, string(output))
	}
	return nil
}

// RemoveFile removes a file
func (fs *FileSystem) RemoveFile(path string) error {
	cmd := exec.Command("sudo", "-n", "rm", "-f", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove file %s: %w\nOutput: %s", path, err, string(output))
	}
	return nil
}

// CopyFile copies a file from src to dst
func (fs *FileSystem) CopyFile(src, dst string) error {
	cmd := exec.Command("sudo", "-n", "cp", src, dst)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w\nOutput: %s", src, dst, err, string(output))
	}
	return nil
}

// BackupFile creates a backup of a file with timestamp suffix
func (fs *FileSystem) BackupFile(path string) (string, error) {
	exists, err := fs.FileExists(path)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil // Nothing to backup
	}

	timestamp := exec.Command("date", "+%Y%m%d_%H%M%S")
	output, err := timestamp.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate timestamp: %w", err)
	}

	// Validate output before slicing to prevent out-of-bounds panic
	timestampStr := strings.TrimSpace(string(output))
	if timestampStr == "" {
		return "", fmt.Errorf("failed to generate timestamp: empty output")
	}

	backupPath := fmt.Sprintf("%s.backup.%s", path, timestampStr)

	if err := fs.CopyFile(path, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// GetDiskUsage returns disk usage information for a path
func (fs *FileSystem) GetDiskUsage(path string) (total, used, free uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get disk usage for %s: %w", path, err)
	}

	// Available blocks * size per block = available space in bytes
	free = stat.Bavail * uint64(stat.Bsize)
	total = stat.Blocks * uint64(stat.Bsize)
	used = total - (stat.Bfree * uint64(stat.Bsize))

	return total, used, free, nil
}

// GetDiskUsageHuman returns human-readable disk usage for a path
func (fs *FileSystem) GetDiskUsageHuman(path string) (string, error) {
	cmd := exec.Command("df", "-h", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get disk usage: %w", err)
	}

	return string(output), nil
}

// CountFiles counts the number of files in a directory (non-recursive)
func (fs *FileSystem) CountFiles(path string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}

	return count, nil
}

// ListDirectory lists all entries in a directory
func (fs *FileSystem) ListDirectory(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

// CreateSymlink creates a symbolic link
func (fs *FileSystem) CreateSymlink(target, linkPath string) error {
	cmd := exec.Command("sudo", "-n", "ln", "-sf", target, linkPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w\nOutput: %s", linkPath, target, err, string(output))
	}
	return nil
}

// WriteFile writes content to a file with sudo
func (fs *FileSystem) WriteFile(path string, content []byte, perms os.FileMode) error {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "homelab-setup-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	// Move temp file to target with sudo
	cmd := exec.Command("sudo", "-n", "mv", tmpPath, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to move file to %s: %w\nOutput: %s", path, err, string(output))
	}

	// Set ownership to root:root for security
	// (temp file was created by unprivileged user)
	if err := fs.Chown(path, "root:root"); err != nil {
		return fmt.Errorf("failed to set ownership on %s: %w", path, err)
	}

	// Set permissions
	return fs.Chmod(path, perms)
}

// GetFileSize returns the size of a file in bytes
func (fs *FileSystem) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	return info.Size(), nil
}

// IsMount checks if a path is a mount point
func (fs *FileSystem) IsMount(path string) (bool, error) {
	// Get stat of the path
	pathStat, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	// Get stat of the parent directory
	parentPath := filepath.Dir(path)
	parentStat, err := os.Stat(parentPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat parent %s: %w", parentPath, err)
	}

	// Use type assertions with checks to prevent panic
	pathSys := pathStat.Sys()
	if pathSys == nil {
		return false, fmt.Errorf("failed to get system info for %s", path)
	}
	pathStatT, ok := pathSys.(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to get stat info for %s: not a Unix filesystem", path)
	}

	parentSys := parentStat.Sys()
	if parentSys == nil {
		return false, fmt.Errorf("failed to get system info for %s", parentPath)
	}
	parentStatT, ok := parentSys.(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("failed to get stat info for %s: not a Unix filesystem", parentPath)
	}

	// If the device IDs are different, it's a mount point
	return pathStatT.Dev != parentStatT.Dev, nil
}
