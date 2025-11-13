package steps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func TestNewContainerSetup(t *testing.T) {
	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	cfg := config.New("")
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	if setup == nil {
		t.Fatal("NewContainerSetup returned nil")
	}

	if setup.containers == nil {
		t.Error("ContainerSetup.containers is nil")
	}
	if setup.fs == nil {
		t.Error("ContainerSetup.fs is nil")
	}
	if setup.config == nil {
		t.Error("ContainerSetup.config is nil")
	}
	if setup.ui == nil {
		t.Error("ContainerSetup.ui is nil")
	}
	if setup.markers == nil {
		t.Error("ContainerSetup.markers is nil")
	}
}

func TestCountYAMLFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "container-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	files := []string{
		"test1.yml",
		"test2.yaml",
		"test3.txt",
		".hidden.yml",
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test counting
	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	cfg := config.New("")
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	count, err := setup.countYAMLFiles(tmpDir)
	if err != nil {
		t.Fatalf("countYAMLFiles failed: %v", err)
	}

	// Should count 3 YAML files (test1.yml, test2.yaml, .hidden.yml)
	expected := 3
	if count != expected {
		t.Errorf("Expected %d YAML files, got %d", expected, count)
	}
}

func TestGenerateEnvContent(t *testing.T) {
	cfg := config.New("")
	cfg.Set("ENV_PUID", "1001")
	cfg.Set("ENV_PGID", "1002")
	cfg.Set("ENV_TZ", "America/New_York")
	cfg.Set("ENV_APPDATA_PATH", "/custom/path")

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	// Test generic service
	content := setup.generateEnvContent("generic")

	if content == "" {
		t.Error("generateEnvContent returned empty string")
	}

	// Check that base variables are present
	if !contains(content, "PUID=1001") {
		t.Error("Content missing PUID=1001")
	}
	if !contains(content, "PGID=1002") {
		t.Error("Content missing PGID=1002")
	}
	if !contains(content, "TZ=America/New_York") {
		t.Error("Content missing TZ=America/New_York")
	}
	if !contains(content, "APPDATA_PATH=/custom/path") {
		t.Error("Content missing APPDATA_PATH=/custom/path")
	}
}

func TestGenerateEnvContent_Media(t *testing.T) {
	cfg := config.New("")
	cfg.Set("ENV_PUID", "1000")
	cfg.Set("ENV_PGID", "1000")
	cfg.Set("ENV_TZ", "UTC")
	cfg.Set("ENV_APPDATA_PATH", "/data")
	cfg.Set("PLEX_CLAIM_TOKEN", "claim-test-token")
	cfg.Set("JELLYFIN_PUBLIC_URL", "https://jellyfin.example.com")

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	content := setup.generateEnvContent("media")

	// Check media-specific variables
	if !contains(content, "PLEX_CLAIM_TOKEN=claim-test-token") {
		t.Error("Content missing PLEX_CLAIM_TOKEN")
	}
	if !contains(content, "JELLYFIN_PUBLIC_URL=https://jellyfin.example.com") {
		t.Error("Content missing JELLYFIN_PUBLIC_URL")
	}
	if !contains(content, "TRANSCODE_DEVICE=/dev/dri") {
		t.Error("Content missing TRANSCODE_DEVICE")
	}
}

func TestGetServiceInfo(t *testing.T) {
	cfg := config.New("")
	cfg.Set("HOMELAB_BASE_DIR", "/test/containers")

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	services := system.NewServiceManager()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	deployment := NewDeployment(containers, fs, services, cfg, uiInstance, markers)

	info := deployment.GetServiceInfo("media")

	if info.Name != "media" {
		t.Errorf("Expected Name=media, got %s", info.Name)
	}
	if info.DisplayName != "Media" {
		t.Errorf("Expected DisplayName=Media, got %s", info.DisplayName)
	}
	if info.Directory != "/test/containers/media" {
		t.Errorf("Expected Directory=/test/containers/media, got %s", info.Directory)
	}
	if info.UnitName != "podman-compose-media.service" {
		t.Errorf("Expected UnitName=podman-compose-media.service, got %s", info.UnitName)
	}
}

func TestContainerSetupServiceDirectoryUsesHomelabBase(t *testing.T) {
	cfg := config.New("")
	cfg.Set("HOMELAB_BASE_DIR", "/mnt/homelab")

	setup := NewContainerSetup(system.NewContainerManager(), system.NewFileSystem(), cfg, ui.New(), config.NewMarkers(""))

	dir := setup.serviceDirectory("media")
	expected := filepath.Join("/mnt/homelab", "media")
	if dir != expected {
		t.Fatalf("expected %s, got %s", expected, dir)
	}
}

func TestContainerSetupServiceDirectoryFallback(t *testing.T) {
	cfg := config.New("")
	cfg.Set("CONTAINERS_BASE", "/legacy")

	setup := NewContainerSetup(system.NewContainerManager(), system.NewFileSystem(), cfg, ui.New(), config.NewMarkers(""))

	dir := setup.serviceDirectory("web")
	expected := filepath.Join("/legacy", "web")
	if dir != expected {
		t.Fatalf("expected %s, got %s", expected, dir)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
