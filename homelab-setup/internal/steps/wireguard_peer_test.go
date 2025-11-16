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

type fakeKeyGenerator struct {
	counter int
}

func (f *fakeKeyGenerator) GenerateKeyPair() (string, string, error) {
	f.counter++
	priv := fmt.Sprintf("priv-%d", f.counter)
	pub := fmt.Sprintf("pub-%d", f.counter)
	return priv, pub, nil
}

func (f *fakeKeyGenerator) GeneratePresharedKey() (string, error) {
	f.counter++
	return fmt.Sprintf("psk-%d", f.counter), nil
}

func (f *fakeKeyGenerator) DerivePublicKey(privateKey string) (string, error) {
	return "derived-pub", nil
}

func TestNextPeerAddressSequential(t *testing.T) {
	used := map[string]struct{}{
		"10.253.0.2/32": {},
		"10.253.0.3/32": {},
	}
	addr, err := nextPeerAddress("10.253.0.1/24", used)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "10.253.0.4/32" {
		t.Fatalf("expected 10.253.0.4/32, got %s", addr)
	}
}

func TestAddPeerWorkflowCreatesServerAndClientConfig(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.New(filepath.Join(tmp, "config.conf"))
	if err := cfg.Set("WIREGUARD_CONFIG_DIR", tmp); err != nil {
		t.Fatalf("failed to set config dir: %v", err)
	}
	if err := cfg.Set("WIREGUARD_INTERFACE", "wg0"); err != nil {
		t.Fatalf("failed to set interface: %v", err)
	}
	if err := cfg.Set("WIREGUARD_PUBLIC_KEY", "server-pub"); err != nil {
		t.Fatalf("failed to set public key: %v", err)
	}

	packages := system.NewPackageManager()
	services := system.NewServiceManager()
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmp)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)
	testUI.SetNonInteractive(true)

	setup := NewWireGuardSetup(packages, services, fs, network, cfg, testUI, markers)
	fakeGen := &fakeKeyGenerator{}
	setup.SetKeyGenerator(fakeGen)

	serverConfig := `[Interface]
Address = 10.253.0.1/24
PrivateKey = server-private
ListenPort = 51820
`
	configPath := filepath.Join(tmp, "wg0.conf")
	if err := os.WriteFile(configPath, []byte(serverConfig), 0600); err != nil {
		t.Fatalf("failed to write server config: %v", err)
	}

	routeAll := false
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "laptop",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  filepath.Join(tmp, "export"),
		PersistentKeepaliveSeconds: 30,
		GeneratePresharedKey:       boolPtr(true),
		NonInteractive:             true,
		SkipQRCode:                 true,
		SkipServiceRestart:         true,
	}

	if err := setup.AddPeerWorkflow(opts); err != nil {
		t.Fatalf("AddPeerWorkflow failed: %v", err)
	}

	secondOpts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "tablet",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  filepath.Join(tmp, "export"),
		PersistentKeepaliveSeconds: 30,
		GeneratePresharedKey:       boolPtr(true),
		NonInteractive:             true,
		SkipQRCode:                 true,
		SkipServiceRestart:         true,
	}
	if err := setup.AddPeerWorkflow(secondOpts); err != nil {
		t.Fatalf("second AddPeerWorkflow failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "AllowedIPs = 10.253.0.2/32") || !strings.Contains(content, "AllowedIPs = 10.253.0.3/32") {
		t.Fatalf("server config missing expected peer allocations: %s", content)
	}

	exportDir := filepath.Join(tmp, "export")
	files, err := os.ReadDir(exportDir)
	if err != nil {
		t.Fatalf("failed to list export dir: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 exports, got %d", len(files))
	}
	firstExport := filepath.Join(exportDir, files[0].Name())
	exportBytes, err := os.ReadFile(firstExport)
	if err != nil {
		t.Fatalf("failed to read export: %v", err)
	}
	info, err := os.Stat(firstExport)
	if err != nil {
		t.Fatalf("failed to stat export: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("export permissions should be 0600, got %v", info.Mode().Perm())
	}
	exportContent := string(exportBytes)
	if !strings.Contains(exportContent, "Endpoint = vpn.example.com:51820") {
		t.Fatalf("client config missing endpoint: %s", exportContent)
	}
	if !strings.Contains(exportContent, "DNS = 1.1.1.1") {
		t.Fatalf("client config missing DNS: %s", exportContent)
	}
	if !strings.Contains(exportContent, "AllowedIPs = 10.253.0.0/24") {
		t.Fatalf("client config missing allowed subnet: %s", exportContent)
	}
}

func TestAddPeerWorkflowRejectsBothClientAllowedIPsAndRouteAll(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.New(filepath.Join(tmp, "config.conf"))
	if err := cfg.Set("WIREGUARD_CONFIG_DIR", tmp); err != nil {
		t.Fatalf("failed to set config dir: %v", err)
	}
	if err := cfg.Set("WIREGUARD_INTERFACE", "wg0"); err != nil {
		t.Fatalf("failed to set interface: %v", err)
	}
	if err := cfg.Set("WIREGUARD_PUBLIC_KEY", "server-pub"); err != nil {
		t.Fatalf("failed to set public key: %v", err)
	}

	packages := system.NewPackageManager()
	services := system.NewServiceManager()
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmp)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)
	testUI.SetNonInteractive(true)

	setup := NewWireGuardSetup(packages, services, fs, network, cfg, testUI, markers)
	fakeGen := &fakeKeyGenerator{}
	setup.SetKeyGenerator(fakeGen)

	serverConfig := `[Interface]
Address = 10.253.0.1/24
PrivateKey = server-private
ListenPort = 51820
`
	configPath := filepath.Join(tmp, "wg0.conf")
	if err := os.WriteFile(configPath, []byte(serverConfig), 0600); err != nil {
		t.Fatalf("failed to write server config: %v", err)
	}

	// Test that providing both ClientAllowedIPs and RouteAll returns an error
	routeAll := true
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "laptop",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		ClientAllowedIPs:           "10.0.0.0/8",
		RouteAll:                   &routeAll,
		OutputDir:                  filepath.Join(tmp, "export"),
		PersistentKeepaliveSeconds: 30,
		NonInteractive:             true,
		SkipQRCode:                 true,
		SkipServiceRestart:         true,
	}

	err := setup.AddPeerWorkflow(opts)
	if err == nil {
		t.Fatal("expected error when both ClientAllowedIPs and RouteAll are set, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected error message to mention 'mutually exclusive', got: %v", err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
