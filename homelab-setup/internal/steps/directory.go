package steps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// DirectorySetup handles directory structure creation
type DirectorySetup struct {
	config  *config.Config
	ui      *ui.UI
	markers *config.Markers
}

// NewDirectorySetup creates a new DirectorySetup instance
func NewDirectorySetup(cfg *config.Config, ui *ui.UI, markers *config.Markers) *DirectorySetup {
	return &DirectorySetup{
		config:  cfg,
		ui:      ui,
		markers: markers,
	}
}

// CreateBaseStructure creates the base directory structure
func (d *DirectorySetup) CreateBaseStructure(baseDir, owner string) error {
	d.ui.Infof("Creating container service directories in %s...", baseDir)
	d.ui.Print("")

	// Define container service directories (media, web, cloud)
	serviceDirs := []struct {
		name        string
		description string
	}{
		{"media", "Plex, Jellyfin, Tautulli"},
		{"web", "Overseerr, Wizarr, Organizr, Homepage"},
		{"cloud", "Nextcloud, Immich, Collabora"},
	}

	// Create base containers directory
	if err := system.EnsureDirectory(baseDir, owner, 0755); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", baseDir, err)
	}
	d.ui.Successf("  ✓ Created %s", baseDir)

	// Create each service directory
	for _, svc := range serviceDirs {
		svcPath := filepath.Join(baseDir, svc.name)
		d.ui.Infof("Creating %s - %s", svcPath, svc.description)

		if err := system.EnsureDirectory(svcPath, owner, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", svcPath, err)
		}

		d.ui.Successf("  ✓ Created %s/", svc.name)
	}

	return nil
}

// CreateAppdataDirs creates application data directories
func (d *DirectorySetup) CreateAppdataDirs(appdataBase, owner string) error {
	d.ui.Print("")
	d.ui.Infof("Creating application data directories in %s...", appdataBase)

	// Define appdata directories for each service
	appdataDirs := []string{
		"plex",
		"jellyfin",
		"tautulli",
		"overseerr",
		"wizarr",
		"organizr",
		"homepage",
		"nextcloud",
		"nextcloud-db",
		"nextcloud-redis",
		"collabora",
		"immich",
		"immich-db",
		"immich-redis",
		"immich-ml",
	}

	// Create base appdata directory
	if err := system.EnsureDirectory(appdataBase, owner, 0755); err != nil {
		return fmt.Errorf("failed to create appdata base directory %s: %w", appdataBase, err)
	}
	d.ui.Successf("  ✓ Created %s", appdataBase)

	// Create each appdata directory
	for _, service := range appdataDirs {
		serviceDir := filepath.Join(appdataBase, service)

		if err := system.EnsureDirectory(serviceDir, owner, 0755); err != nil {
			return fmt.Errorf("failed to create appdata directory %s: %w", serviceDir, err)
		}
	}

	d.ui.Successf("  ✓ Created %d appdata directories", len(appdataDirs))
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

		if err := system.EnsureDirectory(mp.path, "root:root", 0755); err != nil {
			return fmt.Errorf("failed to create mount point %s: %w", mp.path, err)
		}

		d.ui.Successf("  ✓ Created %s", mp.path)
	}

	return nil
}

// VerifyStructure verifies the directory structure was created correctly
func (d *DirectorySetup) VerifyStructure(containersBase, appdataBase string) error {
	d.ui.Print("")
	d.ui.Info("Verifying directory structure...")

	// Check container service directories
	serviceDirs := []string{"media", "web", "cloud"}
	for _, service := range serviceDirs {
		serviceDir := filepath.Join(containersBase, service)
		exists, err := system.DirectoryExists(serviceDir)
		if err != nil {
			return fmt.Errorf("failed to check directory %s: %w", serviceDir, err)
		}

		if !exists {
			return fmt.Errorf("directory %s was not created", serviceDir)
		}

		d.ui.Successf("  ✓ %s exists", serviceDir)
	}

	// Check appdata base directory
	exists, err := system.DirectoryExists(appdataBase)
	if err != nil {
		return fmt.Errorf("failed to check appdata directory: %w", err)
	}
	if !exists {
		return fmt.Errorf("appdata directory %s was not created", appdataBase)
	}
	d.ui.Successf("  ✓ %s exists", appdataBase)

	// Count appdata subdirectories
	entries, err := os.ReadDir(appdataBase)
	if err == nil {
		count := 0
		for _, entry := range entries {
			if entry.IsDir() {
				count++
			}
		}
		d.ui.Successf("  ✓ Found %d appdata subdirectories", count)
	}

	return nil
}

// DisplayStructure displays the created directory structure
func (d *DirectorySetup) DisplayStructure(containersBase, appdataBase string) error {
	d.ui.Print("")
	d.ui.Info("Directory structure created:")
	d.ui.Print("")

	d.ui.Info("Container Services:")
	d.ui.Printf("%s/", containersBase)
	d.ui.Print("  ├── media/       (compose.yml, .env)")
	d.ui.Print("  ├── web/         (compose.yml, .env)")
	d.ui.Print("  └── cloud/       (compose.yml, .env)")

	d.ui.Print("")
	d.ui.Info("Application Data:")
	d.ui.Printf("%s/", appdataBase)

	// Show sample appdata directories
	entries, err := os.ReadDir(appdataBase)
	if err == nil && len(entries) > 0 {
		count := 0
		for _, entry := range entries {
			if entry.IsDir() {
				if count < 5 {
					d.ui.Printf("  ├── %s/", entry.Name())
				}
				count++
			}
		}
		if count > 5 {
			d.ui.Printf("  └── ... and %d more", count-5)
		}
	}

	return nil
}

// VerifyAppdataPermissions verifies the homelab user can write to appdata directories
func (d *DirectorySetup) VerifyAppdataPermissions(appdataBase, owner string) error {
	d.ui.Print("")
	d.ui.Info("Verifying write permissions for appdata directories...")

	// Create a test file to verify write access
	testFilePath := filepath.Join(appdataBase, ".write-test")
	testContent := []byte("permission test")

	// Try to write test file
	if err := system.WriteFile(testFilePath, testContent, 0644); err != nil {
		return fmt.Errorf("cannot write to appdata directory %s: %w (check owner is %s)", appdataBase, err, owner)
	}

	// Verify we can read it back
	readContent, err := os.ReadFile(testFilePath)
	if err != nil {
		// Clean up test file even if read fails
		os.Remove(testFilePath)
		return fmt.Errorf("cannot read from appdata directory %s: %w", appdataBase, err)
	}

	// Verify content matches
	if string(readContent) != string(testContent) {
		os.Remove(testFilePath)
		return fmt.Errorf("appdata directory write verification failed: content mismatch")
	}

	// Clean up test file
	if err := os.Remove(testFilePath); err != nil {
		d.ui.Warning(fmt.Sprintf("Could not remove test file %s: %v", testFilePath, err))
	}

	d.ui.Success("Write permissions verified - user can write to appdata directories")
	return nil
}

const directoryCompletionMarker = "directory-setup-complete"

// Run executes the directory setup step
func (d *DirectorySetup) Run() error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(d.markers, directoryCompletionMarker, "directories-created")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		d.ui.Info("Directory structure already created (marker found)")
		d.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + directoryCompletionMarker)
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
	d.ui.Print("")

	// Prompt for containers base directory
	d.ui.Step("Container Services Directory")
	d.ui.Info("This directory will contain compose files organized by service type")
	d.ui.Info("Structure: /srv/containers/{media,web,cloud}/")
	d.ui.Print("")

	existingContainersBase := d.config.GetOrDefault("CONTAINERS_BASE", "")
	var containersBase string

	if existingContainersBase != "" {
		d.ui.Infof("Previously configured: %s", existingContainersBase)
		useExisting, err := d.ui.PromptYesNo("Use this directory?", true)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if useExisting {
			containersBase = existingContainersBase
		}
	}

	if containersBase == "" {
		input, err := d.ui.PromptInput("Enter containers base directory", "/srv/containers")
		if err != nil {
			return fmt.Errorf("failed to prompt for containers directory: %w", err)
		}
		containersBase = input
	}

	// Appdata directory (fixed location per documentation)
	appdataBase := "/var/lib/containers/appdata"
	d.ui.Print("")
	d.ui.Info("Application data will be stored in: " + appdataBase)

	// Create container service directories
	d.ui.Step("Creating Container Service Directories")
	if err := d.CreateBaseStructure(containersBase, homelabUser); err != nil {
		return fmt.Errorf("failed to create container structure: %w", err)
	}

	// Create appdata directories
	d.ui.Step("Creating Application Data Directories")
	if err := d.CreateAppdataDirs(appdataBase, homelabUser); err != nil {
		return fmt.Errorf("failed to create appdata directories: %w", err)
	}

	// Verify write permissions
	d.ui.Step("Verifying Permissions")
	if err := d.VerifyAppdataPermissions(appdataBase, homelabUser); err != nil {
		return fmt.Errorf("permission verification failed: %w", err)
	}

	// Create NFS mount points if needed
	d.ui.Step("NFS Mount Points")
	if err := d.CreateNFSMountPoints(); err != nil {
		d.ui.Warning(fmt.Sprintf("Failed to create NFS mount points: %v", err))
		// Non-critical error, continue
	}

	// Verify structure
	d.ui.Step("Verification")
	if err := d.VerifyStructure(containersBase, appdataBase); err != nil {
		return fmt.Errorf("directory structure verification failed: %w", err)
	}

	// Display structure
	if err := d.DisplayStructure(containersBase, appdataBase); err != nil {
		d.ui.Warning(fmt.Sprintf("Failed to display structure: %v", err))
	}

	// Save configuration
	d.ui.Step("Saving Configuration")
	if err := d.config.Set("CONTAINERS_BASE", containersBase); err != nil {
		return fmt.Errorf("failed to save containers base directory: %w", err)
	}
	// Use APPDATA_BASE as per architecture document
	if err := d.config.Set("APPDATA_BASE", appdataBase); err != nil {
		return fmt.Errorf("failed to save appdata base: %w", err)
	}
	// Also set APPDATA_PATH for backwards compatibility with legacy configs and .env files
	if err := d.config.Set("APPDATA_PATH", appdataBase); err != nil {
		return fmt.Errorf("failed to save appdata path: %w", err)
	}

	d.ui.Print("")
	d.ui.Separator()
	d.ui.Success("✓ Directory structure created successfully")
	d.ui.Infof("Container services: %s", containersBase)
	d.ui.Infof("Application data: %s", appdataBase)

	// Create completion marker
	if err := d.markers.Create(directoryCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
