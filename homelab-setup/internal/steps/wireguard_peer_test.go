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

const baseServerConfig = `[Interface]
Address = 10.253.0.1/24
PrivateKey = server-private
ListenPort = 51820
`

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

func newWireGuardTestEnv(t *testing.T, serverConfig string) (*WireGuardSetup, string, string, *bytes.Buffer) {
	t.Helper()
	if serverConfig == "" {
		serverConfig = baseServerConfig
	}
	tmp := t.TempDir()
	cfg := config.New(filepath.Join(tmp, "config.conf"))
	mustSetConfig(t, cfg, "WIREGUARD_CONFIG_DIR", tmp)
	mustSetConfig(t, cfg, "WIREGUARD_INTERFACE", "wg0")
	mustSetConfig(t, cfg, "WIREGUARD_PUBLIC_KEY", "server-pub")
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
	configPath := filepath.Join(tmp, "wg0.conf")
	if err := os.WriteFile(configPath, []byte(serverConfig), 0600); err != nil {
		t.Fatalf("failed to write server config: %v", err)
	}
	return setup, configPath, tmp, buf
}

func mustSetConfig(t *testing.T, cfg *config.Config, key, value string) {
	t.Helper()
	if err := cfg.Set(key, value); err != nil {
		t.Fatalf("failed to set %s: %v", key, err)
	}
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

func TestCollectUsedPeerIPsHandlesInlineComments(t *testing.T) {
	config := `[Interface]
Address = 10.253.0.1/24

[Peer]
AllowedIPs = 10.253.0.2/32 # laptop

[Peer]
allowedips = 10.253.0.3/32 ; tablet
`
	parsed := parseWireGuardConfig(config)
	used := collectUsedPeerIPs(parsed)
	if _, ok := used["10.253.0.2/32"]; !ok {
		t.Fatalf("expected 10.253.0.2/32 to be marked as used")
	}
	if _, ok := used["10.253.0.3/32"]; !ok {
		t.Fatalf("expected 10.253.0.3/32 to be marked as used")
	}
	addr, err := nextPeerAddress("10.253.0.1/24", used)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "10.253.0.4/32" {
		t.Fatalf("expected 10.253.0.4/32, got %s", addr)
	}
}

func TestNextPeerAddressAllowsLargeSubnet(t *testing.T) {
	addr, err := nextPeerAddress("10.10.0.1/16", map[string]struct{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "10.10.0.2/32" {
		t.Fatalf("expected 10.10.0.2/32, got %s", addr)
	}
}

func TestNextPeerAddressDetectsExhaustionOnLargeNetwork(t *testing.T) {
	used := make(map[string]struct{})
	for host := 2; host < 65535; host++ {
		third := (host >> 8) & 0xff
		fourth := host & 0xff
		addr := fmt.Sprintf("10.10.%d.%d/32", third, fourth)
		used[addr] = struct{}{}
	}
	_, err := nextPeerAddress("10.10.0.1/16", used)
	if err == nil {
		t.Fatal("expected error when subnet is exhausted")
	}
	if !strings.Contains(err.Error(), "no available IPs") {
		t.Fatalf("expected exhaustion error, got %v", err)
	}
}

func TestAddPeerWorkflowCreatesServerAndClientConfig(t *testing.T) {
	setup, configPath, tmp, _ := newWireGuardTestEnv(t, "")
	routeAll := false
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "laptop",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  filepath.Join(tmp, "export"),
		PersistentKeepaliveSeconds: intPtr(30),
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
		PersistentKeepaliveSeconds: intPtr(30),
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
	setup, _, tmp, _ := newWireGuardTestEnv(t, "")
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
		PersistentKeepaliveSeconds: intPtr(30),
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

func TestAddPeerWorkflowExportDirectoryFailureDoesNotModifyServerConfig(t *testing.T) {
	setup, configPath, tmp, buf := newWireGuardTestEnv(t, "")
	badExport := filepath.Join(tmp, "export-file")
	if err := os.WriteFile(badExport, []byte("content"), 0600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}
	routeAll := false
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "blocked",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  badExport,
		PersistentKeepaliveSeconds: intPtr(30),
		GeneratePresharedKey:       boolPtr(true),
		NonInteractive:             true,
		SkipQRCode:                 true,
		SkipServiceRestart:         true,
	}
	if err := setup.AddPeerWorkflow(opts); err == nil {
		t.Fatal("expected export failure")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) != baseServerConfig {
		t.Fatalf("server config should remain unchanged on failure: %s", string(data))
	}
	output := buf.String()
	if !strings.Contains(output, "Client configuration (not exported)") {
		t.Fatalf("expected message about unexported client config, got %s", output)
	}
	if !strings.Contains(output, "[Interface]") {
		t.Fatalf("expected client config to be printed on failure")
	}
}

func TestAddPeerWorkflowExportFileFailureDoesNotModifyServerConfig(t *testing.T) {
	setup, configPath, tmp, buf := newWireGuardTestEnv(t, "")
	originalWriter := clientConfigFileWriter
	clientConfigFileWriter = func(path string, data []byte, perm os.FileMode) error {
		return fmt.Errorf("simulated write failure")
	}
	defer func() { clientConfigFileWriter = originalWriter }()
	exportDir := filepath.Join(tmp, "export")
	routeAll := false
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "laptop",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  exportDir,
		PersistentKeepaliveSeconds: intPtr(30),
		GeneratePresharedKey:       boolPtr(true),
		NonInteractive:             true,
		SkipQRCode:                 true,
		SkipServiceRestart:         true,
	}
	if err := setup.AddPeerWorkflow(opts); err == nil {
		t.Fatal("expected export file failure")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) != baseServerConfig {
		t.Fatalf("server config should remain unchanged on failure: %s", string(data))
	}
	output := buf.String()
	if !strings.Contains(output, "Client configuration (not exported)") {
		t.Fatalf("expected message about unexported client config, got %s", output)
	}
	if !strings.Contains(output, "[Interface]") {
		t.Fatalf("expected client config to be printed on failure")
	}
}

func TestAddPeerWorkflowWarnsWhenQRCodeFails(t *testing.T) {
	setup, configPath, tmp, buf := newWireGuardTestEnv(t, "")
	t.Setenv("PATH", "")
	routeAll := false
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "laptop",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  filepath.Join(tmp, "export"),
		PersistentKeepaliveSeconds: intPtr(30),
		GeneratePresharedKey:       boolPtr(true),
		NonInteractive:             true,
		SkipQRCode:                 false,
		SkipServiceRestart:         true,
	}
	if err := setup.AddPeerWorkflow(opts); err != nil {
		t.Fatalf("AddPeerWorkflow failed: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "AllowedIPs = 10.253.0.2/32") {
		t.Fatalf("server config missing peer: %s", string(data))
	}
	if !strings.Contains(buf.String(), "Failed to render QR code") {
		t.Fatalf("expected QR warning in output, got %s", buf.String())
	}
}

func TestAddPeerWorkflowAllowsDisablingKeepalive(t *testing.T) {
	setup, configPath, tmp, _ := newWireGuardTestEnv(t, "")
	routeAll := false
	opts := &WireGuardPeerWorkflowOptions{
		InterfaceName:              "wg0",
		PeerName:                   "node0",
		Endpoint:                   "vpn.example.com:51820",
		DNS:                        "1.1.1.1",
		RouteAll:                   &routeAll,
		OutputDir:                  filepath.Join(tmp, "export"),
		PersistentKeepaliveSeconds: intPtr(0),
		GeneratePresharedKey:       boolPtr(true),
		NonInteractive:             true,
		SkipQRCode:                 true,
		SkipServiceRestart:         true,
	}
	if err := setup.AddPeerWorkflow(opts); err != nil {
		t.Fatalf("AddPeerWorkflow failed: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "PersistentKeepalive = 0") {
		t.Fatalf("server config should record disabled keepalive: %s", string(data))
	}
	exportDir := filepath.Join(tmp, "export")
	files, err := os.ReadDir(exportDir)
	if err != nil {
		t.Fatalf("failed to read export dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 export file, got %d", len(files))
	}
	clientData, err := os.ReadFile(filepath.Join(exportDir, files[0].Name()))
	if err != nil {
		t.Fatalf("failed to read client config: %v", err)
	}
	if !strings.Contains(string(clientData), "PersistentKeepalive = 0") {
		t.Fatalf("client config should record disabled keepalive: %s", string(clientData))
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}
