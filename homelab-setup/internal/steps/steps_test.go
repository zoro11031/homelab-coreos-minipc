package steps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// TestPreflightChecker tests basic preflight checker functionality
func TestPreflightCheckerNew(t *testing.T) {
	packages := system.NewPackageManager()
	network := system.NewNetwork()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	checker := NewPreflightChecker(packages, network, testUI, markers, cfg)
	if checker == nil {
		t.Fatal("Expected non-nil PreflightChecker")
	}
}

// TestUserConfigurator tests basic user configurator functionality
func TestUserConfiguratorNew(t *testing.T) {
	users := system.NewUserManager()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	uc := NewUserConfigurator(users, cfg, testUI, markers)
	if uc == nil {
		t.Fatal("Expected non-nil UserConfigurator")
	}
}

// TestDirectorySetup tests basic directory setup functionality
func TestDirectorySetupNew(t *testing.T) {
	fs := system.NewFileSystem()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	ds := NewDirectorySetup(fs, cfg, testUI, markers)
	if ds == nil {
		t.Fatal("Expected non-nil DirectorySetup")
	}
}

// TestNFSConfigurator tests basic NFS configurator functionality
func TestNFSConfiguratorNew(t *testing.T) {
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	nfs := NewNFSConfigurator(fs, network, cfg, testUI, markers)
	if nfs == nil {
		t.Fatal("Expected non-nil NFSConfigurator")
	}
}

// TestContainerSetup tests basic container setup functionality
func TestContainerSetupNew(t *testing.T) {
	containers := system.NewContainerManager()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	cs := NewContainerSetup(containers, cfg, testUI, markers)
	if cs == nil {
		t.Fatal("Expected non-nil ContainerSetup")
	}
}

// TestDeployment tests basic deployment functionality
func TestDeploymentNew(t *testing.T) {
	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	services := system.NewServiceManager()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	d := NewDeployment(containers, fs, services, cfg, testUI, markers)
	if d == nil {
		t.Fatal("Expected non-nil Deployment")
	}
}

// TestWireGuardSetup tests basic WireGuard setup functionality
func TestWireGuardSetupNew(t *testing.T) {
	packages := system.NewPackageManager()
	services := system.NewServiceManager()
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	testUI := ui.New()
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)
	cfg := config.New(filepath.Join(tmpDir, "test.conf"))

	wg := NewWireGuardSetup(packages, services, fs, network, cfg, testUI, markers)
	if wg == nil {
		t.Fatal("Expected non-nil WireGuardSetup")
	}
}

// TestMarkerCreation tests marker creation in steps
func TestMarkerCreation(t *testing.T) {
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)

	// Test creating a marker
	err := markers.Create("test-marker")
	if err != nil {
		t.Fatalf("Failed to create marker: %v", err)
	}

	// Test checking if marker exists
	exists, err := markers.Exists("test-marker")
	if err != nil {
		t.Fatalf("Failed to check marker: %v", err)
	}
	if !exists {
		t.Fatal("Expected marker to exist")
	}
}

// TestConfigurationSaving tests configuration saving
func TestConfigurationSaving(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.conf")
	cfg := config.New(cfgPath)

	// Test setting a value
	err := cfg.Set("TEST_KEY", "test_value")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Test getting the value
	value := cfg.GetOrDefault("TEST_KEY", "")
	if value != "test_value" {
		t.Fatalf("Expected 'test_value', got '%s'", value)
	}

	// Test that config file was created
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("Expected config file to be created")
	}
}

// TestIdempotency tests that steps can be run multiple times safely
func TestIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	markers := config.NewMarkers(tmpDir)

	// Create a marker
	err := markers.Create("step-complete")
	if err != nil {
		t.Fatalf("Failed to create marker: %v", err)
	}

	// Check if exists (should return true)
	exists, err := markers.Exists("step-complete")
	if err != nil {
		t.Fatalf("Failed to check marker: %v", err)
	}
	if !exists {
		t.Fatal("Expected marker to exist")
	}

	// Try to create again (should not error)
	err = markers.Create("step-complete")
	if err != nil {
		t.Fatalf("Failed to create marker again: %v", err)
	}

	// Should still exist
	exists, err = markers.Exists("step-complete")
	if err != nil {
		t.Fatalf("Failed to check marker: %v", err)
	}
	if !exists {
		t.Fatal("Expected marker to still exist")
	}
}
