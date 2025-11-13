package steps

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func TestWireGuardWriteConfigCreatesSecureFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.conf")
	cfg := config.New(cfgPath)
	if err := cfg.Set("WIREGUARD_CONFIG_DIR", tmpDir); err != nil {
		t.Fatalf("failed to set config dir: %v", err)
	}

	packages := system.NewPackageManager()
	services := system.NewServiceManager()
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	setup := NewWireGuardSetup(packages, services, fs, network, cfg, testUI, markers)
	wgCfg := &WireGuardConfig{
		InterfaceName: "wgtest",
		InterfaceIP:   "10.1.0.1/24",
		ListenPort:    "51820",
	}

	if err := setup.WriteConfig(wgCfg, "test-private-key"); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, "wgtest.conf")
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if !strings.Contains(string(data), "PrivateKey = test-private-key") {
		t.Fatalf("config file missing private key, content: %s", string(data))
	}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected permissions 0600, got %v", info.Mode().Perm())
	}
}
