package steps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// DirectorySetup handles directory structure creation
type DirectorySetup struct {
	fs      *system.FileSystem
	config  *config.Config
	ui      *ui.UI
	markers *config.Markers
}

// NewDirectorySetup creates a new DirectorySetup instance
func NewDirectorySetup(fs *system.FileSystem, cfg *config.Config, ui *ui.UI, markers *config.Markers) *DirectorySetup {
	return &DirectorySetup{
		fs:      fs,
		config:  cfg,
		ui:      ui,
		markers: markers,
	}
}

// PromptForBaseDir prompts for the base directory
func (d *DirectorySetup) PromptForBaseDir() (string, error) {
	d.ui.Info("The base directory will contain all homelab services and data")
	d.ui.Info("Recommended locations: /mnt/homelab, /srv/homelab, or /var/homelab")
	d.ui.Print("")

	// Check if already configured
	existingBaseDir := d.config.GetOrDefault("HOMELAB_BASE_DIR", "")
	if existingBaseDir != "" {
		d.ui.Infof("Previously configured: %s", existingBaseDir)
		useExisting, err := d.ui.PromptYesNo("Use this directory?", true)
		if err != nil {
			return "", fmt.Errorf("failed to prompt: %w", err)
		}
		if useExisting {
			return existingBaseDir, nil
		}
	}

	// Prompt for base directory
	baseDir, err := d.ui.PromptInput("Enter base directory path", "/mnt/homelab")
	if err != nil {
		return "", fmt.Errorf("failed to prompt for base directory: %w", err)
	}

	// Validate path
	if err := common.ValidatePath(baseDir); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	return baseDir, nil
}

// CreateBaseStructure creates the base directory structure
func (d *DirectorySetup) CreateBaseStructure(baseDir, owner string) error {
	d.ui.Infof("Creating base directory structure in %s...", baseDir)
	d.ui.Print("")

	// Define base structure
	baseDirs := []struct {
		path        string
		description string
	}{
		{baseDir, "Base homelab directory"},
		{filepath.Join(baseDir, "config"), "Service configurations"},
		{filepath.Join(baseDir, "data"), "Service data"},
		{filepath.Join(baseDir, "compose"), "Docker Compose files"},
		{filepath.Join(baseDir, "services"), "Individual service directories"},
	}

	// Create each directory
	for _, dir := range baseDirs {
		d.ui.Infof("Creating %s - %s", dir.path, dir.description)

		if err := d.fs.EnsureDirectory(dir.path, owner, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir.path, err)
		}

		d.ui.Successf("  ✓ Created %s", dir.path)
	}

	return nil
}

// CreateServiceDirs creates directories for individual services
func (d *DirectorySetup) CreateServiceDirs(baseDir, owner string, services []string) error {
	if len(services) == 0 {
		d.ui.Info("No services specified, skipping service directory creation")
		return nil
	}

	d.ui.Print("")
	d.ui.Infof("Creating service directories...")

	servicesDir := filepath.Join(baseDir, "services")

	for _, service := range services {
		serviceDir := filepath.Join(servicesDir, service)
		d.ui.Infof("Creating %s", serviceDir)

		if err := d.fs.EnsureDirectory(serviceDir, owner, 0755); err != nil {
			return fmt.Errorf("failed to create service directory %s: %w", serviceDir, err)
		}

		// Create common subdirectories for each service
		subdirs := []string{"config", "data"}
		for _, subdir := range subdirs {
			subdirPath := filepath.Join(serviceDir, subdir)
			if err := d.fs.EnsureDirectory(subdirPath, owner, 0755); err != nil {
				return fmt.Errorf("failed to create subdirectory %s: %w", subdirPath, err)
			}
		}

		d.ui.Successf("  ✓ Created %s with subdirectories", serviceDir)
	}

	return nil
}

// CreateNFSMountPoints creates mount points for NFS shares
func (d *DirectorySetup) CreateNFSMountPoints() error {
	// Check if NFS is configured
	nfsServer := d.config.GetOrDefault("NFS_SERVER", "")
	if nfsServer == "" {
		d.ui.Info("NFS not configured, skipping mount point creation")
		return nil
	}

	d.ui.Print("")
	d.ui.Infof("Creating NFS mount points...")

	// Common NFS mount points
	mountPoints := []struct {
		path        string
		description string
	}{
		{"/mnt/nas-media", "NFS media share"},
		{"/mnt/nas-photos", "NFS photos share"},
		{"/mnt/nas-backups", "NFS backups share"},
	}

	for _, mp := range mountPoints {
		d.ui.Infof("Creating %s - %s", mp.path, mp.description)

		if err := d.fs.EnsureDirectory(mp.path, "root:root", 0755); err != nil {
			return fmt.Errorf("failed to create mount point %s: %w", mp.path, err)
		}

		d.ui.Successf("  ✓ Created %s", mp.path)
	}

	return nil
}

// VerifyStructure verifies the directory structure was created correctly
func (d *DirectorySetup) VerifyStructure(baseDir string) error {
	d.ui.Print("")
	d.ui.Info("Verifying directory structure...")

	// Check base directories
	requiredDirs := []string{
		baseDir,
		filepath.Join(baseDir, "config"),
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "compose"),
		filepath.Join(baseDir, "services"),
	}

	for _, dir := range requiredDirs {
		exists, err := d.fs.DirectoryExists(dir)
		if err != nil {
			return fmt.Errorf("failed to check directory %s: %w", dir, err)
		}

		if !exists {
			return fmt.Errorf("directory %s was not created", dir)
		}

		d.ui.Successf("  ✓ %s exists", dir)
	}

	return nil
}

// DisplayStructure displays the created directory structure
func (d *DirectorySetup) DisplayStructure(baseDir string) error {
	d.ui.Print("")
	d.ui.Info("Directory structure created:")
	d.ui.Print("")
	d.ui.Printf("%s/", baseDir)
	d.ui.Print("  ├── config/     (service configurations)")
	d.ui.Print("  ├── data/       (service data)")
	d.ui.Print("  ├── compose/    (docker compose files)")
	d.ui.Print("  └── services/   (individual service directories)")

	// Check if services were created
	servicesDir := filepath.Join(baseDir, "services")
	entries, err := os.ReadDir(servicesDir)
	if err == nil && len(entries) > 0 {
		d.ui.Print("")
		d.ui.Info("Service directories:")
		for _, entry := range entries {
			if entry.IsDir() {
				d.ui.Printf("  - %s/", entry.Name())
			}
		}
	}

	return nil
}

// Run executes the directory setup step
func (d *DirectorySetup) Run() error {
	// Check if already completed
	exists, err := d.markers.Exists("directories-created")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if exists {
		d.ui.Info("Directory structure already created (marker found)")
		d.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/directories-created")
		return nil
	}

	d.ui.Header("Directory Structure Setup")
	d.ui.Info("Creating directory structure for homelab services...")
	d.ui.Print("")

	// Get homelab user from config
	homelabUser := d.config.GetOrDefault("HOMELAB_USER", "")
	if homelabUser == "" {
		return fmt.Errorf("homelab user not configured (run user configuration first)")
	}

	d.ui.Infof("Using homelab user: %s", homelabUser)

	// Prompt for base directory
	d.ui.Step("Select Base Directory")
	baseDir, err := d.PromptForBaseDir()
	if err != nil {
		return fmt.Errorf("failed to get base directory: %w", err)
	}

	// Create base structure
	d.ui.Step("Creating Base Directory Structure")
	if err := d.CreateBaseStructure(baseDir, homelabUser); err != nil {
		return fmt.Errorf("failed to create base structure: %w", err)
	}

	// Ask if they want to create service directories
	d.ui.Step("Service Directories")
	createServices, err := d.ui.PromptYesNo("Create directories for specific services?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt for service creation: %w", err)
	}

	if createServices {
		// Define common services
		commonServices := []string{
			"plex",
			"jellyfin",
			"sonarr",
			"radarr",
			"lidarr",
			"prowlarr",
			"qbittorrent",
			"overseerr",
			"nextcloud",
			"immich",
			"caddy",
			"adguard",
		}

		d.ui.Info("Select services to create directories for:")
		serviceIndices, err := d.ui.PromptMultiSelect("Services", commonServices)
		if err != nil {
			return fmt.Errorf("failed to prompt for services: %w", err)
		}

		selectedServices := []string{}
		for _, idx := range serviceIndices {
			if idx >= 0 && idx < len(commonServices) {
				selectedServices = append(selectedServices, commonServices[idx])
			}
		}

		if len(selectedServices) > 0 {
			if err := d.CreateServiceDirs(baseDir, homelabUser, selectedServices); err != nil {
				return fmt.Errorf("failed to create service directories: %w", err)
			}
		}
	}

	// Create NFS mount points if needed
	d.ui.Step("NFS Mount Points")
	if err := d.CreateNFSMountPoints(); err != nil {
		d.ui.Warning(fmt.Sprintf("Failed to create NFS mount points: %v", err))
		// Non-critical error, continue
	}

	// Verify structure
	d.ui.Step("Verification")
	if err := d.VerifyStructure(baseDir); err != nil {
		return fmt.Errorf("directory structure verification failed: %w", err)
	}

	// Display structure
	if err := d.DisplayStructure(baseDir); err != nil {
		d.ui.Warning(fmt.Sprintf("Failed to display structure: %v", err))
	}

	// Save configuration
	d.ui.Step("Saving Configuration")
	if err := d.config.Set("HOMELAB_BASE_DIR", baseDir); err != nil {
		return fmt.Errorf("failed to save base directory: %w", err)
	}

	d.ui.Print("")
	d.ui.Separator()
	d.ui.Success("✓ Directory structure created successfully")
	d.ui.Infof("Base directory: %s", baseDir)

	// Create completion marker
	if err := d.markers.Create("directories-created"); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
