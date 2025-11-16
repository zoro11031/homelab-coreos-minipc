package steps

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func TestSanitizeConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean value",
			input:    "validkey123",
			expected: "validkey123",
		},
		{
			name:     "value with newline injection",
			input:    "validkey\n[Interface]\nPrivateKey = malicious",
			expected: "validkeyInterfacePrivateKey = malicious",
		},
		{
			name:     "value with carriage return",
			input:    "validkey\r\n[Peer]",
			expected: "validkeyPeer",
		},
		{
			name:     "value with brackets",
			input:    "valid[key]with[brackets]",
			expected: "validkeywithbrackets",
		},
		{
			name:     "value with hash comment injection",
			input:    "validkey # malicious comment\n[Interface]",
			expected: "validkey  malicious commentInterface",
		},
		{
			name:     "value with leading/trailing whitespace",
			input:    "  validkey  ",
			expected: "validkey",
		},
		{
			name:     "complex injection attempt",
			input:    "key123\n\n[Interface]\nAddress = 0.0.0.0/0\n# comment\n[Peer]\nPublicKey = evil",
			expected: "key123InterfaceAddress = 0.0.0.0/0 commentPeerPublicKey = evil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeConfigValue(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeConfigValue(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

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

	// WriteConfig creates the file with sudo, owned by root:root with 0600
	if err := setup.WriteConfig(wgCfg, "test-private-key"); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, "wgtest.conf")

	// File is owned by root:root with 0600, so we need sudo to read it
	output, err := exec.Command("sudo", "cat", configFile).Output()
	if err != nil {
		t.Fatalf("failed to read config file with sudo: %v", err)
	}

	if !strings.Contains(string(output), "PrivateKey = test-private-key") {
		t.Fatalf("config file missing private key, content: %s", string(output))
	}

	// Check permissions using fs.GetPermissions which handles sudo
	info, err := fs.GetPermissions(configFile)
	if err != nil {
		t.Fatalf("failed to get permissions: %v", err)
	}

	if info.Perm() != 0600 {
		t.Fatalf("expected permissions 0600, got %v", info.Perm())
	}
}

func TestAddPeerToConfigSanitizesFields(t *testing.T) {
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

	// Create a basic config first
	wgCfg := &WireGuardConfig{
		InterfaceName: "wgtest",
		InterfaceIP:   "10.1.0.1/24",
		ListenPort:    "51820",
	}
	if err := setup.WriteConfig(wgCfg, "test-private-key"); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Add peer with malicious content
	maliciousPeer := &WireGuardPeer{
		Name:       "TestPeer\n[Interface]",
		PublicKey:  "validkey\n[Peer]\nAllowedIPs = 0.0.0.0/0",
		AllowedIPs: "10.1.0.2/32\n# comment injection",
		Endpoint:   "example.com:51820\n[Interface]\nPrivateKey = stolen",
	}

	if err := setup.AddPeerToConfig("wgtest", maliciousPeer); err != nil {
		t.Fatalf("AddPeerToConfig failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, "wgtest.conf")
	output, err := exec.Command("sudo", "cat", configFile).Output()
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(output)

	// Verify that injected brackets are removed
	// Count only standalone [Interface] and [Peer] lines, not ones in comments
	lines := strings.Split(content, "\n")
	interfaceCount := 0
	peerCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[Interface]" {
			interfaceCount++
		}
		if trimmed == "[Peer]" {
			peerCount++
		}
	}

	if interfaceCount > 1 {
		t.Errorf("Config contains %d [Interface] sections, expected 1: %s", interfaceCount, content)
	}

	if peerCount > 1 {
		t.Errorf("Config contains %d [Peer] sections, expected 1: %s", peerCount, content)
	}

	// Verify the legitimate peer section exists with sanitized name
	if !strings.Contains(content, "Peer: TestPeerInterface") {
		t.Errorf("Config missing sanitized peer name, content: %s", content)
	}

	// Verify the sanitized values don't contain newlines or brackets
	if strings.Contains(content, "validkey\n") {
		t.Errorf("PublicKey was not sanitized properly: %s", content)
	}

	// Verify no raw injection strings remain
	if strings.Contains(content, "[Peer]\nAllowedIPs") {
		t.Errorf("PublicKey injection was not sanitized: %s", content)
	}
}
