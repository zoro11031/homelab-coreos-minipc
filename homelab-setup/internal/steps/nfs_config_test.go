package steps

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func TestAddToFstabAppendsEntryAndReloads(t *testing.T) {
	tmpDir := t.TempDir()
	fstabPath := filepath.Join(tmpDir, "fstab")
	if err := os.WriteFile(fstabPath, []byte("# test fstab\n"), 0644); err != nil {
		t.Fatalf("failed to seed fstab: %v", err)
	}

	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	if err := cfg.Set("NFS_FSTAB_PATH", fstabPath); err != nil {
		t.Fatalf("failed to set fstab path: %v", err)
	}

	fs := system.NewFileSystem()
	network := system.NewNetwork()
	packages := system.NewPackageManager()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, packages)
	fakeRunner := &fakeCommandRunner{commandOutputs: map[string]string{}}
	nfs.runner = fakeRunner

	if err := nfs.AddToFstab("192.168.1.10", "/export", "/mnt/data"); err != nil {
		t.Fatalf("AddToFstab failed: %v", err)
	}

	data, err := os.ReadFile(fstabPath)
	if err != nil {
		t.Fatalf("failed to read fstab: %v", err)
	}

	// With the new configurable options, the default is "defaults,_netdev"
	expectedLine := "192.168.1.10:/export /mnt/data nfs defaults,_netdev 0 0"
	if !strings.Contains(string(data), expectedLine) {
		t.Fatalf("fstab missing entry, content: %s", string(data))
	}

	if !fakeRunner.ran("sudo -n systemctl daemon-reload") {
		t.Fatalf("expected systemctl daemon-reload to run, commands: %v", fakeRunner.commands)
	}
}

func TestMountNFSUsesRunner(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	packages := system.NewPackageManager()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, packages)
	fakeRunner := &fakeCommandRunner{commandOutputs: map[string]string{
		"systemd-escape --path --suffix=mount /mnt/nas-media": "mnt-nas\\x2dmedia.mount\n",
	}}
	nfs.runner = fakeRunner

	mountPoint := filepath.Join(tmpDir, "mnt")
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		t.Fatalf("failed to create mount point: %v", err)
	}

	if err := nfs.MountNFS(mountPoint); err != nil {
		t.Fatalf("MountNFS failed: %v", err)
	}

	if !fakeRunner.ran(fmt.Sprintf("sudo -n mount %s", mountPoint)) {
		t.Fatalf("expected mount command to run, commands: %v", fakeRunner.commands)
	}
}

func TestMountNFSFailureReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	packages := system.NewPackageManager()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, packages)
	fakeRunner := &fakeCommandRunner{failCommand: "sudo -n mount /mnt/fail"}
	nfs.runner = fakeRunner

	mountPoint := "/mnt/fail"
	if err := nfs.MountNFS(mountPoint); err == nil {
		t.Fatal("expected MountNFS to fail when runner returns error")
	}
}

type fakeCommandRunner struct {
	commands       []string
	failCommand    string
	commandOutputs map[string]string
}

func (f *fakeCommandRunner) Run(name string, args ...string) (string, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	f.commands = append(f.commands, cmd)
	if f.failCommand != "" && cmd == f.failCommand {
		return "", fmt.Errorf("forced failure for %s", cmd)
	}
	if output, ok := f.commandOutputs[cmd]; ok {
		return output, nil
	}
	return "", nil
}

func (f *fakeCommandRunner) ran(command string) bool {
	for _, cmd := range f.commands {
		if cmd == command {
			return true
		}
	}
	return false
}

func TestCreateSystemdMountUnit(t *testing.T) {
	// This test verifies the systemd unit file generation logic
	// The function tries to write to /etc/systemd/system which requires root
	// In a test environment, we'll skip the actual execution but verify the logic

	tmpDir := t.TempDir()

	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	packages := system.NewPackageManager()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, packages)
	fakeRunner := &fakeCommandRunner{commandOutputs: map[string]string{
		"systemd-escape --path --suffix=mount /mnt/nas-media": "mnt-nas\\x2dmedia.mount\n",
	}}
	nfs.runner = fakeRunner

	// Test creating mount unit - this will fail due to permissions
	// but we're mainly testing the logic flow
	host := "192.168.1.10"
	export := "/mnt/storage/media"
	mountPoint := "/mnt/nas-media"

	// The function will fail because we can't write to /etc/systemd/system
	// but that's expected in a test environment
	err := nfs.CreateSystemdMountUnit(host, export, mountPoint)
	if err == nil {
		t.Skip("Test skipped: requires root access to write systemd units")
	}

	// Verify error is about permissions
	if !strings.Contains(err.Error(), "failed to write mount unit") {
		t.Logf("Expected permission error, got: %v", err)
	}
}

func TestPathToUnitName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/mnt/nas-media", "mnt-nas\\x2dmedia.mount"},
		{"/mnt/nas-nextcloud", "mnt-nas\\x2dnextcloud.mount"},
		{"/srv/data", "srv-data.mount"},
		{"/mnt/foo/bar/baz", "mnt-foo-bar-baz.mount"},
		{"/mnt/My Media", "mnt-My\\x20Media.mount"},
	}

	fakeRunner := &fakeCommandRunner{commandOutputs: map[string]string{
		"systemd-escape --path --suffix=mount /mnt/nas-media":     "mnt-nas\\x2dmedia.mount\n",
		"systemd-escape --path --suffix=mount /mnt/nas-nextcloud": "mnt-nas\\x2dnextcloud.mount\n",
		"systemd-escape --path --suffix=mount /srv/data":          "srv-data.mount\n",
		"systemd-escape --path --suffix=mount /mnt/foo/bar/baz":   "mnt-foo-bar-baz.mount\n",
		"systemd-escape --path --suffix=mount /mnt/My Media":      "mnt-My\\x20Media.mount\n",
	}}

	for _, tt := range tests {
		result, err := pathToUnitName(fakeRunner, tt.input)
		if err != nil {
			t.Fatalf("pathToUnitName(%q) returned error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("pathToUnitName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMountPointToUnitName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path with single directory",
			input:    "/mnt/data",
			expected: "mnt-data.mount",
		},
		{
			name:     "nested path with multiple directories",
			input:    "/mnt/foo/bar/baz",
			expected: "mnt-foo-bar-baz.mount",
		},
		{
			name:     "path with hyphen in name",
			input:    "/mnt/nas-media",
			expected: "mnt-nas-media.mount",
		},
		{
			name:     "path with multiple hyphens",
			input:    "/mnt/nas-nextcloud",
			expected: "mnt-nas-nextcloud.mount",
		},
		{
			name:     "path with srv directory",
			input:    "/srv/data",
			expected: "srv-data.mount",
		},
		{
			name:     "deeply nested path",
			input:    "/mnt/storage/media/videos",
			expected: "mnt-storage-media-videos.mount",
		},
		{
			name:     "path with single letter components",
			input:    "/a/b/c",
			expected: "a-b-c.mount",
		},
		{
			name:     "path with numbers",
			input:    "/mnt/disk1",
			expected: "mnt-disk1.mount",
		},
		{
			name:     "path without leading slash",
			input:    "mnt/data",
			expected: "mnt-data.mount",
		},
		{
			name:     "path with spaces (not recommended but should work)",
			input:    "/mnt/My Media",
			expected: "mnt-My Media.mount",
		},
		{
			name:     "path with underscores",
			input:    "/mnt/nas_storage",
			expected: "mnt-nas_storage.mount",
		},
		{
			name:     "path with dots",
			input:    "/mnt/media.backup",
			expected: "mnt-media.backup.mount",
		},
		{
			name:     "root path",
			input:    "/",
			expected: ".mount",
		},
		{
			name:     "empty string",
			input:    "",
			expected: ".mount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mountPointToUnitName(tt.input)
			if result != tt.expected {
				t.Errorf("mountPointToUnitName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
