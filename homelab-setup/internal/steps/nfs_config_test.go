package steps

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

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

// fakePackageManager is a mock PackageManager for testing
type fakePackageManager struct {
	installedPackages map[string]bool
	checkError        error
}

func (f *fakePackageManager) IsInstalled(packageName string) (bool, error) {
	if f.checkError != nil {
		return false, f.checkError
	}
	installed, ok := f.installedPackages[packageName]
	if !ok {
		return false, nil
	}
	return installed, nil
}

func (f *fakePackageManager) CheckMultiple(packages []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, pkg := range packages {
		installed, err := f.IsInstalled(pkg)
		if err != nil {
			return nil, err
		}
		result[pkg] = installed
	}
	return result, nil
}

func (f *fakePackageManager) GetPackageVersion(packageName string) (string, error) {
	if f.checkError != nil {
		return "", f.checkError
	}
	if installed, ok := f.installedPackages[packageName]; ok && installed {
		return "1.0.0", nil
	}
	return "", fmt.Errorf("package not installed")
}

func TestCreateSystemdUnits(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.New(filepath.Join(tmpDir, "config.conf"))

	// Use the mock filesystem to capture output
	mockFS := system.NewMockFileSystem()

	network := system.NewNetwork()
	packages := system.NewPackageManager()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(mockFS, network, cfg, testUI, markers, packages)
	fakeRunner := &fakeCommandRunner{commandOutputs: map[string]string{}}
	nfs.runner = fakeRunner

	host := "192.168.1.10"
	export := "/mnt/storage/media"
	mountPoint := "/mnt/nas-media"

	err := nfs.CreateSystemdUnits(host, export, mountPoint)
	if err != nil {
		t.Fatalf("CreateSystemdUnits failed: %v", err)
	}

	// 1. Verify .mount file content
	mountUnitName := "mnt-nas-media.mount"
	mountUnitPath := filepath.Join("/etc/systemd/system", mountUnitName)
	mountContent, found := mockFS.WrittenFiles[mountUnitPath]
	if !found {
		t.Fatalf("expected mount unit file %s to be written, but it wasn't", mountUnitPath)
	}
	if !strings.Contains(string(mountContent), "What=192.168.1.10:/mnt/storage/media") {
		t.Errorf(".mount file has wrong 'What=' line: %s", string(mountContent))
	}
	if !strings.Contains(string(mountContent), "Where=/mnt/nas-media") {
		t.Errorf(".mount file has wrong 'Where=' line: %s", string(mountContent))
	}
	if !strings.Contains(string(mountContent), "Options=nofail,defaults,_netdev") {
		t.Errorf(".mount file has wrong 'Options=' line: %s", string(mountContent))
	}

	// 2. Verify .automount file content
	automountUnitName := "mnt-nas-media.automount"
	automountUnitPath := filepath.Join("/etc/systemd/system", automountUnitName)
	automountContent, found := mockFS.WrittenFiles[automountUnitPath]
	if !found {
		t.Fatalf("expected automount unit file %s to be written, but it wasn't", automountUnitPath)
	}
	if !strings.Contains(string(automountContent), "Where=/mnt/nas-media") {
		t.Errorf(".automount file has wrong 'Where=' line: %s", string(automountContent))
	}
	if !strings.Contains(string(automountContent), "[Automount]") {
		t.Errorf(".automount file is missing [Automount] section: %s", string(automountContent))
	}

	// 3. Verify systemd commands
	if !fakeRunner.ran("sudo -n systemctl daemon-reload") {
		t.Error("expected 'systemctl daemon-reload' to be run")
	}
	if !fakeRunner.ran("sudo -n systemctl enable --now mnt-nas-media.automount") {
		t.Error("expected 'systemctl enable --now' for the automount unit to be run")
	}
}

func TestMountPointToUnitBaseName(t *testing.T) {
	tests := []struct {
		name       string
		mountPoint string
		expected   string
	}{
		{name: "canonical path", mountPoint: "/mnt/nas-media", expected: "mnt-nas-media"},
		{name: "trailing slash", mountPoint: "/mnt/nas-media/", expected: "mnt-nas-media"},
		{name: "multiple trailing slashes", mountPoint: "/mnt/nas-media///", expected: "mnt-nas-media"},
		{name: "whitespace replaced", mountPoint: "/mnt/My Media", expected: "mnt-My-Media"},
		{name: "multiple whitespaces", mountPoint: "/mnt/My  Media", expected: "mnt-My-Media"},
		{name: "leading and trailing spaces", mountPoint: " /mnt/My Media ", expected: "mnt-My-Media"},
		{name: "path with multiple subdirs", mountPoint: "/srv/data/long/path", expected: "srv-data-long-path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mountPointToUnitBaseName(tt.mountPoint); got != tt.expected {
				t.Fatalf("mountPointToUnitBaseName(%q) = %q, want %q", tt.mountPoint, got, tt.expected)
			}
		})
	}
}

func TestCheckNFSUtilsWhenInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	// Create a fake package manager where nfs-utils is installed
	fakePackages := &fakePackageManager{
		installedPackages: map[string]bool{
			"nfs-utils": true,
		},
	}

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, fakePackages)

	err := nfs.CheckNFSUtils()
	if err != nil {
		t.Errorf("CheckNFSUtils() returned error when nfs-utils is installed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "nfs-utils package is installed") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestCheckNFSUtilsWhenNotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	// Create a fake package manager where nfs-utils is not installed
	fakePackages := &fakePackageManager{
		installedPackages: map[string]bool{
			"nfs-utils": false,
		},
	}

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, fakePackages)

	err := nfs.CheckNFSUtils()
	if err == nil {
		t.Error("CheckNFSUtils() should return error when nfs-utils is not installed")
	}

	if !strings.Contains(err.Error(), "nfs-utils package is not installed") {
		t.Errorf("Expected error about nfs-utils not installed, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "rpm-ostree install nfs-utils") {
		t.Errorf("Expected installation instructions, got: %s", output)
	}
}

func TestCheckNFSUtilsWhenCheckFails(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.New(filepath.Join(tmpDir, "config.conf"))
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	// Create a fake package manager that returns an error when checking
	fakePackages := &fakePackageManager{
		checkError: fmt.Errorf("rpm command failed"),
	}

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers, fakePackages)

	err := nfs.CheckNFSUtils()
	// When check fails, the function should return nil and proceed with a warning
	if err != nil {
		t.Errorf("CheckNFSUtils() should return nil when package check fails, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Could not verify nfs-utils package") {
		t.Errorf("Expected warning about unable to verify package, got: %s", output)
	}
	if !strings.Contains(output, "Proceeding anyway") {
		t.Errorf("Expected message about proceeding anyway, got: %s", output)
	}
}
