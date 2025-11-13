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
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers)
	fakeRunner := &fakeCommandRunner{}
	nfs.runner = fakeRunner

	if err := nfs.AddToFstab("192.168.1.10", "/export", "/mnt/data"); err != nil {
		t.Fatalf("AddToFstab failed: %v", err)
	}

	data, err := os.ReadFile(fstabPath)
	if err != nil {
		t.Fatalf("failed to read fstab: %v", err)
	}

	expectedLine := "192.168.1.10:/export /mnt/data nfs defaults,nfsvers=4.2,_netdev 0 0"
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
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers)
	fakeRunner := &fakeCommandRunner{}
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
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers)
	fakeRunner := &fakeCommandRunner{failCommand: "sudo -n mount /mnt/fail"}
	nfs.runner = fakeRunner

	mountPoint := "/mnt/fail"
	if err := nfs.MountNFS(mountPoint); err == nil {
		t.Fatal("expected MountNFS to fail when runner returns error")
	}
}

type fakeCommandRunner struct {
	commands    []string
	failCommand string
}

func (f *fakeCommandRunner) Run(name string, args ...string) (string, error) {
	cmd := strings.Join(append([]string{name}, args...), " ")
	f.commands = append(f.commands, cmd)
	if f.failCommand != "" && cmd == f.failCommand {
		return "", fmt.Errorf("forced failure for %s", cmd)
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
