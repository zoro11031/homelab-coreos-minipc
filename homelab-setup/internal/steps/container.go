package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

const containerSetupCompletionMarker = "container-setup-complete"

// getContainersBase returns the base directory for container service files
func getContainersBase(cfg *config.Config) string {
	// Use CONTAINERS_BASE which should be set to /srv/containers
	return cfg.GetOrDefault(config.KeyContainersBase, "/srv/containers")
}

// serviceDirectory returns the directory path for a given service.
func serviceDirectory(cfg *config.Config, serviceName string) string {
	return filepath.Join(getContainersBase(cfg), serviceName)
}

// findTemplateDirectory locates compose templates
func findTemplateDirectory(cfg *config.Config, ui *ui.UI) (string, error) {
	ui.Step("Locating Compose Templates")

	// Check home setup directory first
	homeDir := os.Getenv("HOME")
	templateDirHome := filepath.Join(homeDir, "setup", "compose-setup")

	if exists, _ := system.DirectoryExists(templateDirHome); exists {
		// Count YAML files
		count, _ := countYAMLFiles(templateDirHome)
		if count > 0 {
			ui.Successf("Found templates in: %s (%d YAML file(s))", templateDirHome, count)
			return templateDirHome, nil
		}
		ui.Warningf("Directory exists but contains no YAML files: %s", templateDirHome)
	}

	// Check /usr/share as fallback
	templateDirUsr := "/usr/share/compose-setup"
	if exists, _ := system.DirectoryExists(templateDirUsr); exists {
		count, _ := countYAMLFiles(templateDirUsr)
		if count > 0 {
			ui.Successf("Found templates in: %s (%d YAML file(s))", templateDirUsr, count)
			return templateDirUsr, nil
		}
		ui.Warningf("Directory exists but contains no YAML files: %s", templateDirUsr)
	}

	ui.Error("No compose templates found in any location")
	ui.Info("Searched locations:")
	ui.Infof("  - %s", templateDirHome)
	ui.Infof("  - %s", templateDirUsr)
	ui.Print("")
	ui.Info("Expected to find .yml or .yaml files in one of these directories")

	return "", fmt.Errorf("no compose templates found")
}

// countYAMLFiles counts YAML files in a directory
func countYAMLFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yml" || ext == ".yaml" {
			count++
		}
	}

	return count, nil
}

// discoverStacks discovers available container stacks
func discoverStacks(cfg *config.Config, ui *ui.UI, templateDir string) (map[string]string, error) {
	ui.Step("Discovering Available Container Stacks")
	ui.Infof("Scanning directory: %s", templateDir)

	// Exclude patterns
	excludePatterns := []string{
		".*",        // Hidden files
		"*.example", // Example files
		"README*",   // Documentation files
		"*.md",      // Markdown files
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory: %w", err)
	}

	stacks := make(map[string]string)
	totalYAML := 0
	excludedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		ext := filepath.Ext(filename)

		// Only process YAML files
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		totalYAML++

		// Check exclude patterns
		shouldExclude := false
		for _, pattern := range excludePatterns {
			matched, _ := filepath.Match(pattern, filename)
			if matched {
				ui.Infof("Excluding: %s (matches pattern: %s)", filename, pattern)
				excludedCount++
				shouldExclude = true
				break
			}
		}

		if shouldExclude {
			continue
		}

		// Get service name (filename without extension)
		serviceName := strings.TrimSuffix(filename, ext)
		stacks[serviceName] = filename
		ui.Successf("Found stack: %s (%s)", serviceName, filename)
	}

	if len(stacks) == 0 {
		ui.Error("No valid compose stack files discovered")
		ui.Infof("Directory checked: %s", templateDir)
		ui.Infof("Total YAML files found: %d", totalYAML)
		ui.Infof("Files excluded by patterns: %d", excludedCount)
		ui.Print("")
		ui.Info("Exclude patterns:")
		for _, pattern := range excludePatterns {
			ui.Infof("  - %s", pattern)
		}
		ui.Print("")
		ui.Info("Stack files should be named like: media.yml, web.yml, cloud.yml")
		ui.Info("Excluded files: .env.example, README.md, .hidden files")
		return nil, fmt.Errorf("no valid stacks found")
	}

	ui.Successf("Discovered %d valid container stack(s) (excluded %d file(s))", len(stacks), excludedCount)
	return stacks, nil
}

// selectStacks allows user to select which stacks to setup
func selectStacks(cfg *config.Config, ui *ui.UI, stacks map[string]string) ([]string, error) {
	ui.Step("Container Stack Selection")
	ui.Print("")
	ui.Info("Available container stacks:")
	ui.Print("")

	// Sort stack names for consistent ordering
	var stackNames []string
	for name := range stacks {
		stackNames = append(stackNames, name)
	}
	sort.Strings(stackNames)

	// Display available stacks
	for i, name := range stackNames {
		ui.Printf("  %d) %s (%s)", i+1, name, stacks[name])
	}
	ui.Printf("  %d) All stacks", len(stackNames)+1)
	ui.Print("")

	// Prompt for selection
	ui.Info("Select which container stacks to setup:")
	ui.Info("  - You can select multiple stacks using Space, then press Enter")
	ui.Info("  - Or select 'All stacks' to setup everything")
	ui.Print("")

	// Build options for multi-select
	var options []string
	for _, name := range stackNames {
		options = append(options, fmt.Sprintf("%s (%s)", name, stacks[name]))
	}
	options = append(options, "All stacks")

	// Use multi-select prompt
	selectedIndices, err := ui.PromptMultiSelect("Select stacks to setup", options)
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for stack selection: %w", err)
	}

	if len(selectedIndices) == 0 {
		return nil, fmt.Errorf("no stacks selected")
	}

	// Check if "All stacks" was selected
	allStacksIndex := len(stackNames)
	for _, idx := range selectedIndices {
		if idx == allStacksIndex {
			ui.Success("Selected: All stacks")
			// Save selected services to config before returning
			if err := cfg.Set("SELECTED_SERVICES", strings.Join(stackNames, " ")); err != nil {
				ui.Warning(fmt.Sprintf("Failed to save selected services: %v", err))
			}
			return stackNames, nil
		}
	}

	// Get selected stack names
	var selected []string
	for _, idx := range selectedIndices {
		if idx < len(stackNames) {
			selected = append(selected, stackNames[idx])
		}
	}

	ui.Success("Selected stacks:")
	for _, name := range selected {
		ui.Infof("  - %s", name)
	}
	ui.Print("")

	// Save selected services to config
	if err := cfg.Set("SELECTED_SERVICES", strings.Join(selected, " ")); err != nil {
		ui.Warning(fmt.Sprintf("Failed to save selected services: %v", err))
	}

	return selected, nil
}

// copyTemplates copies selected compose templates to destination
func copyTemplates(cfg *config.Config, ui *ui.UI, templateDir string, stacks map[string]string, selectedStacks []string) error {
	ui.Step("Copying Compose Templates")

	setupUser := cfg.GetOrDefault("HOMELAB_USER", "")
	if setupUser == "" {
		return fmt.Errorf("homelab user not configured")
	}

	for _, serviceName := range selectedStacks {
		templateFile := stacks[serviceName]
		srcPath := filepath.Join(templateDir, templateFile)
		dstDir := serviceDirectory(cfg, serviceName)
		dstPath := filepath.Join(dstDir, "compose.yml")

		// Ensure destination directory exists
		if err := system.EnsureDirectory(dstDir, setupUser, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dstDir, err)
		}

		// Copy template
		ui.Infof("Copying: %s → %s", templateFile, dstPath)
		if err := system.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", templateFile, err)
		}

		// Set ownership and permissions
		if err := system.Chown(dstPath, fmt.Sprintf("%s:%s", setupUser, setupUser)); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", dstPath, err)
		}

		if err := system.Chmod(dstPath, 0644); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", dstPath, err)
		}

		ui.Successf("✓ %s/compose.yml", serviceName)

		// Also create docker-compose.yml symlink for compatibility
		altDstPath := filepath.Join(dstDir, "docker-compose.yml")
		if exists, _ := system.FileExists(altDstPath); !exists {
			if err := system.CreateSymlink("compose.yml", altDstPath); err != nil {
				ui.Warning(fmt.Sprintf("Failed to create symlink: %v", err))
			}
		}
	}

	ui.Successf("Copied %d compose file(s)", len(selectedStacks))
	return nil
}

// createBaseEnvConfig creates base environment configuration
func createBaseEnvConfig(cfg *config.Config, ui *ui.UI) error {
	ui.Step("Creating Base Environment Configuration")

	// Load or prompt for configuration values
	puid := cfg.GetOrDefault("PUID", "1000")
	pgid := cfg.GetOrDefault("PGID", "1000")
	tz := cfg.GetOrDefault("TZ", "America/Chicago")
	// Try APPDATA_BASE first (new standard), fall back to APPDATA_PATH (legacy)
	appdataPath := cfg.GetOrDefault("APPDATA_BASE", "")
	if appdataPath == "" {
		appdataPath = cfg.GetOrDefault("APPDATA_PATH", "/var/lib/containers/appdata")
	}

	// Save base config
	if err := cfg.Set("ENV_PUID", puid); err != nil {
		return fmt.Errorf("failed to save ENV_PUID: %w", err)
	}
	if err := cfg.Set("ENV_PGID", pgid); err != nil {
		return fmt.Errorf("failed to save ENV_PGID: %w", err)
	}
	if err := cfg.Set("ENV_TZ", tz); err != nil {
		return fmt.Errorf("failed to save ENV_TZ: %w", err)
	}
	if err := cfg.Set("ENV_APPDATA_PATH", appdataPath); err != nil {
		return fmt.Errorf("failed to save ENV_APPDATA_PATH: %w", err)
	}

	ui.Success("Base configuration:")
	ui.Infof("  PUID=%s", puid)
	ui.Infof("  PGID=%s", pgid)
	ui.Infof("  TZ=%s", tz)
	ui.Infof("  APPDATA_PATH=%s", appdataPath)

	return nil
}

// configureStackEnv configures environment for a specific stack
func configureStackEnv(cfg *config.Config, ui *ui.UI, serviceName string) error {
	switch serviceName {
	case "media":
		return configureMediaEnv(cfg, ui)
	case "web":
		return configureWebEnv(cfg, ui)
	case "cloud":
		return configureCloudEnv(cfg, ui)
	default:
		ui.Infof("No specific configuration for %s stack", serviceName)
		return nil
	}
}

// configureMediaEnv configures media stack environment
func configureMediaEnv(cfg *config.Config, ui *ui.UI) error {
	ui.Step("Configuring Media Stack Environment")

	// Get Plex claim token
	ui.Info("Plex Setup:")
	ui.Info("  Get your claim token from: https://plex.tv/claim")
	plexClaim, err := ui.PromptInput("Plex claim token (optional)", "")
	if err != nil {
		return err
	}
	if plexClaim != "" {
		if err := cfg.Set("PLEX_CLAIM_TOKEN", plexClaim); err != nil {
			return fmt.Errorf("failed to save PLEX_CLAIM_TOKEN: %w", err)
		}
	}

	// Jellyfin public URL
	jellyfinURL, err := ui.PromptInput("Jellyfin public URL (optional)", "")
	if err != nil {
		return err
	}
	if jellyfinURL != "" {
		if err := cfg.Set("JELLYFIN_PUBLIC_URL", jellyfinURL); err != nil {
			return fmt.Errorf("failed to save JELLYFIN_PUBLIC_URL: %w", err)
		}
	}

	return nil
}

// configureWebEnv configures web stack environment
func configureWebEnv(cfg *config.Config, ui *ui.UI) error {
	ui.Step("Configuring Web Stack Environment")

	// Overseerr API key (optional)
	overseerrAPI, err := ui.PromptInput("Overseerr API key (optional, can configure later)", "")
	if err != nil {
		return err
	}
	if overseerrAPI != "" {
		if err := cfg.Set("OVERSEERR_API_KEY", overseerrAPI); err != nil {
			return fmt.Errorf("failed to save OVERSEERR_API_KEY: %w", err)
		}
	}

	return nil
}

// configureCloudEnv configures cloud stack environment
func configureCloudEnv(cfg *config.Config, ui *ui.UI) error {
	ui.Step("Configuring Cloud Stack Environment")

	// Nextcloud configuration
	ui.Info("Nextcloud Setup:")
	ui.Print("")

	// Admin credentials for initial setup
	nextcloudAdminUser, err := ui.PromptInput("Nextcloud admin username", "admin")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_ADMIN_USER", nextcloudAdminUser); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_ADMIN_USER: %w", err)
	}

	nextcloudAdminPass, err := ui.PromptPasswordConfirm("Nextcloud admin password")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_ADMIN_PASSWORD", nextcloudAdminPass); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_ADMIN_PASSWORD: %w", err)
	}

	// Database credentials
	nextcloudDBUser, err := ui.PromptInput("Nextcloud database username", "nc_user")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_DB_USERNAME", nextcloudDBUser); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_DB_USERNAME: %w", err)
	}

	nextcloudDBPass, err := ui.PromptPasswordConfirm("Nextcloud database password")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_DB_PASSWORD", nextcloudDBPass); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_DB_PASSWORD: %w", err)
	}

	nextcloudDBName, err := ui.PromptInput("Nextcloud database name", "nextcloud")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_DB_DATABASE", nextcloudDBName); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_DB_DATABASE: %w", err)
	}

	nextcloudDomain, err := ui.PromptInput("Nextcloud trusted domain (e.g., cloud.example.com)", "localhost")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_TRUSTED_DOMAINS", nextcloudDomain); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_TRUSTED_DOMAINS: %w", err)
	}
	if err := cfg.Set("NEXTCLOUD_OVERWRITE_HOST", nextcloudDomain); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_OVERWRITE_HOST: %w", err)
	}

	// PHP limits
	phpMemory, err := ui.PromptInput("Nextcloud PHP memory limit", "1024M")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_PHP_MEMORY_LIMIT", phpMemory); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_PHP_MEMORY_LIMIT: %w", err)
	}

	phpUpload, err := ui.PromptInput("Nextcloud PHP upload limit", "1024M")
	if err != nil {
		return err
	}
	if err := cfg.Set("NEXTCLOUD_PHP_UPLOAD_LIMIT", phpUpload); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_PHP_UPLOAD_LIMIT: %w", err)
	}

	// Collabora configuration (truly optional)
	ui.Print("")
	setupCollabora, err := ui.PromptYesNo("Configure Collabora Online (office document editing)?", false)
	if err != nil {
		return err
	}

	if setupCollabora {
		ui.Info("Collabora Setup:")
		ui.Print("")

		collaboraUser, err := ui.PromptInput("Collabora username", "admin")
		if err != nil {
			return err
		}
		if err := cfg.Set("COLLABORA_USERNAME", collaboraUser); err != nil {
			return fmt.Errorf("failed to save COLLABORA_USERNAME: %w", err)
		}

		collaboraPass, err := ui.PromptPassword("Collabora admin password")
		if err != nil {
			return err
		}
		if err := cfg.Set("COLLABORA_PASSWORD", collaboraPass); err != nil {
			return fmt.Errorf("failed to save COLLABORA_PASSWORD: %w", err)
		}
	} else {
		// Set empty values for optional Collabora fields
		if err := cfg.Set("COLLABORA_USERNAME", "admin"); err != nil {
			return fmt.Errorf("failed to save COLLABORA_USERNAME: %w", err)
		}
		if err := cfg.Set("COLLABORA_PASSWORD", ""); err != nil {
			return fmt.Errorf("failed to save COLLABORA_PASSWORD: %w", err)
		}
	}

	// Escape domain for Collabora (dots need to be escaped)
	collaboraDomain := strings.ReplaceAll(nextcloudDomain, ".", "\\.")
	if err := cfg.Set("COLLABORA_DOMAIN", collaboraDomain); err != nil {
		return fmt.Errorf("failed to save COLLABORA_DOMAIN: %w", err)
	}

	// Immich configuration
	ui.Print("")
	ui.Info("Immich Setup:")
	ui.Print("")

	immichDBUser, err := ui.PromptInput("Immich database username", "postgres")
	if err != nil {
		return err
	}
	if err := cfg.Set("IMMICH_DB_USERNAME", immichDBUser); err != nil {
		return fmt.Errorf("failed to save IMMICH_DB_USERNAME: %w", err)
	}

	immichDBPass, err := ui.PromptPasswordConfirm("Immich database password")
	if err != nil {
		return err
	}
	if err := cfg.Set("IMMICH_DB_PASSWORD", immichDBPass); err != nil {
		return fmt.Errorf("failed to save IMMICH_DB_PASSWORD: %w", err)
	}

	immichDBName, err := ui.PromptInput("Immich database name", "immich")
	if err != nil {
		return err
	}
	if err := cfg.Set("IMMICH_DB_DATABASE", immichDBName); err != nil {
		return fmt.Errorf("failed to save IMMICH_DB_DATABASE: %w", err)
	}

	return nil
}

// createEnvFiles creates .env files for selected stacks
func createEnvFiles(cfg *config.Config, ui *ui.UI, selectedStacks []string) error {
	ui.Step("Creating Environment Files")

	serviceUser, err := getServiceUser(cfg)
	if err != nil {
		return err
	}

	for _, serviceName := range selectedStacks {
		envPath := filepath.Join(serviceDirectory(cfg, serviceName), ".env")
		ui.Infof("Creating environment file: %s", envPath)

		content := generateEnvContent(cfg, serviceName)

		// Write file
		if err := system.WriteFile(envPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write .env file for %s: %w", serviceName, err)
		}

		// Set ownership
		if err := system.Chown(envPath, fmt.Sprintf("%s:%s", serviceUser, serviceUser)); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", envPath, err)
		}

		ui.Successf("Created: %s", envPath)
	}

	return nil
}

// generateEnvContent generates .env file content for a service
func generateEnvContent(cfg *config.Config, serviceName string) string {
	puid := cfg.GetOrDefault("ENV_PUID", "1000")
	pgid := cfg.GetOrDefault("ENV_PGID", "1000")
	tz := cfg.GetOrDefault("ENV_TZ", "America/Chicago")
	appdataPath := cfg.GetOrDefault("ENV_APPDATA_PATH", "/var/lib/containers/appdata")

	// Use cases.Title instead of deprecated strings.Title
	caser := cases.Title(language.English)

	content := fmt.Sprintf(`# UBlue uCore Homelab - %s Stack Environment
# Generated by homelab-setup

# User/Group Configuration
PUID=%s
PGID=%s
TZ=%s

# Paths
APPDATA_PATH=%s

`, caser.String(serviceName), puid, pgid, tz, appdataPath)

	// Add service-specific variables
	switch serviceName {
	case "media":
		content += fmt.Sprintf(`# Plex Configuration
PLEX_CLAIM_TOKEN=%s
# Get your claim token from: https://www.plex.tv/claim/

# Jellyfin Configuration
JELLYFIN_PUBLIC_URL=%s

# Hardware Transcoding
# Intel QuickSync device for hardware transcoding
TRANSCODE_DEVICE=/dev/dri

# Note: Media paths are configured in the compose file
# Ensure NFS mounts are set up at /mnt/nas-media before starting services

`, cfg.GetOrDefault("PLEX_CLAIM_TOKEN", ""),
			cfg.GetOrDefault("JELLYFIN_PUBLIC_URL", ""))

	case "web":
		content += fmt.Sprintf(`# Overseerr Configuration (optional - configure in UI)
OVERSEERR_API_KEY=%s

# Web Service Ports (default values from compose file)
OVERSEERR_PORT=5055
WIZARR_PORT=5690
ORGANIZR_PORT=9983
HOMEPAGE_PORT=3000

# Note: These services are typically accessed via reverse proxy
# Configure your reverse proxy to route to these ports via WireGuard tunnel

`, cfg.GetOrDefault("OVERSEERR_API_KEY", ""))

	case "cloud":
		content += fmt.Sprintf(`# Nextcloud Admin Credentials (for initial setup)
NEXTCLOUD_ADMIN_USER=%s
NEXTCLOUD_ADMIN_PASSWORD=%s

# Nextcloud Database Configuration
NEXTCLOUD_DB_USERNAME=%s
NEXTCLOUD_DB_PASSWORD=%s
NEXTCLOUD_DB_DATABASE=%s

# Nextcloud Domain Configuration
NEXTCLOUD_TRUSTED_DOMAINS=%s
NEXTCLOUD_OVERWRITE_HOST=%s

# Nextcloud PHP Limits
NEXTCLOUD_PHP_MEMORY_LIMIT=%s
NEXTCLOUD_PHP_UPLOAD_LIMIT=%s

# Collabora Online Configuration (optional)
COLLABORA_USERNAME=%s
COLLABORA_PASSWORD=%s
COLLABORA_DOMAIN=%s

# Immich Database Configuration
IMMICH_DB_USERNAME=%s
IMMICH_DB_PASSWORD=%s
IMMICH_DB_DATABASE=%s

`, cfg.GetOrDefault("NEXTCLOUD_ADMIN_USER", "admin"),
			cfg.GetOrDefault("NEXTCLOUD_ADMIN_PASSWORD", ""),
			cfg.GetOrDefault("NEXTCLOUD_DB_USERNAME", "nc_user"),
			cfg.GetOrDefault("NEXTCLOUD_DB_PASSWORD", ""),
			cfg.GetOrDefault("NEXTCLOUD_DB_DATABASE", "nextcloud"),
			cfg.GetOrDefault("NEXTCLOUD_TRUSTED_DOMAINS", "localhost"),
			cfg.GetOrDefault("NEXTCLOUD_OVERWRITE_HOST", "localhost"),
			cfg.GetOrDefault("NEXTCLOUD_PHP_MEMORY_LIMIT", "1024M"),
			cfg.GetOrDefault("NEXTCLOUD_PHP_UPLOAD_LIMIT", "1024M"),
			cfg.GetOrDefault("COLLABORA_USERNAME", "admin"),
			cfg.GetOrDefault("COLLABORA_PASSWORD", ""),
			cfg.GetOrDefault("COLLABORA_DOMAIN", "localhost"),
			cfg.GetOrDefault("IMMICH_DB_USERNAME", "postgres"),
			cfg.GetOrDefault("IMMICH_DB_PASSWORD", ""),
			cfg.GetOrDefault("IMMICH_DB_DATABASE", "immich"))
	}

	return content
}

// RunContainerSetup executes the container setup step
func RunContainerSetup(cfg *config.Config, ui *ui.UI) error {
	// Check if already completed
	if cfg.IsComplete(containerSetupCompletionMarker) {
		ui.Info("Container setup already completed (marker found)")
		ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + containerSetupCompletionMarker)
		return nil
	}

	ui.Header("Container Stack Setup")
	ui.Info("Configuring container services for homelab...")
	ui.Print("")

	// Check homelab user
	homelabUser := cfg.GetOrDefault("HOMELAB_USER", "")
	if homelabUser == "" {
		return fmt.Errorf("homelab user not configured (run user setup first)")
	}

	// Find template directory
	templateDir, err := findTemplateDirectory(cfg, ui)
	if err != nil {
		return fmt.Errorf("failed to find templates: %w", err)
	}

	// Discover available stacks
	stacks, err := discoverStacks(cfg, ui, templateDir)
	if err != nil {
		return fmt.Errorf("failed to discover stacks: %w", err)
	}

	// Select stacks to setup
	selectedStacks, err := selectStacks(cfg, ui, stacks)
	if err != nil {
		return fmt.Errorf("failed to select stacks: %w", err)
	}

	// Copy templates
	if err := copyTemplates(cfg, ui, templateDir, stacks, selectedStacks); err != nil {
		return fmt.Errorf("failed to copy templates: %w", err)
	}

	// Create base environment configuration
	if err := createBaseEnvConfig(cfg, ui); err != nil {
		return fmt.Errorf("failed to create base config: %w", err)
	}

	// Configure each selected stack
	for _, serviceName := range selectedStacks {
		if err := configureStackEnv(cfg, ui, serviceName); err != nil {
			ui.Warningf("Failed to configure %s: %v", serviceName, err)
			// Continue with other stacks
		}
	}

	// Create .env files
	if err := createEnvFiles(cfg, ui, selectedStacks); err != nil {
		return fmt.Errorf("failed to create .env files: %w", err)
	}

	ui.Print("")
	ui.Separator()
	ui.Success("✓ Container stack setup completed")
	ui.Infof("Configured %d stack(s): %s", len(selectedStacks), strings.Join(selectedStacks, ", "))

	// Create completion marker
	if err := cfg.MarkComplete(containerSetupCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
