package steps

import (
	"fmt"
	"strings"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

const (
	userCompletionMarker = "user-setup-complete"
	defaultTimezone      = "America/Chicago"
)

// promptForUser prompts for a homelab username or allows using current user
func promptForUser(ui *ui.UI) (string, error) {
	// Get current user
	currentUser, err := system.GetCurrentUser()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	ui.Info(fmt.Sprintf("Current user: %s (UID: %s, GID: %s)", currentUser.Username, currentUser.Uid, currentUser.Gid))
	ui.Print("")

	// Prompt for username with current user as default
	username, err := ui.PromptInput("Enter homelab username (or press Enter to use current user)", currentUser.Username)
	if err != nil {
		return "", fmt.Errorf("failed to prompt for username: %w", err)
	}

	// Validate username
	if err := common.ValidateUsername(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}

	return username, nil
}

// getConfiguredUsername returns a validated username from existing configuration.
// It prefers HOMELAB_USER and falls back to SETUP_USER for backwards compatibility.
func getConfiguredUsername(cfg *config.Config, ui *ui.UI) (string, error) {
	configKeys := []string{"HOMELAB_USER", "SETUP_USER"}

	for _, key := range configKeys {
		value := strings.TrimSpace(cfg.GetOrDefault(key, ""))
		if value == "" {
			continue
		}

		if err := common.ValidateUsername(value); err != nil {
			ui.Warningf("Ignoring %s=%s: %v", key, value, err)
			continue
		}

		if key == "HOMELAB_USER" {
			ui.Infof("Using pre-configured homelab user: %s", value)
			return value, nil
		}

		ui.Infof("Using SETUP_USER (%s) for homelab user", value)
		if err := cfg.Set("HOMELAB_USER", value); err != nil {
			return "", fmt.Errorf("failed to persist HOMELAB_USER: %w", err)
		}
		return value, nil
	}

	return "", nil
}

// validateUser checks if a user exists and can be used for homelab
func validateUser(username string, ui *ui.UI) error {
	exists, err := system.UserExists(username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("user %s does not exist", username)
	}

	// Get user info to display
	userInfo, err := system.GetUserInfo(username)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	ui.Successf("User %s found (UID: %s, GID: %s)", username, userInfo.Uid, userInfo.Gid)

	// Check user groups
	groups, err := system.GetUserGroups(username)
	if err != nil {
		return fmt.Errorf("failed to get user groups: %w", err)
	}

	ui.Infof("User groups: %v", groups)

	// Check if user is in wheel group (recommended for sudo)
	inWheel, err := system.IsUserInGroup(username, "wheel")
	if err != nil {
		return fmt.Errorf("failed to check wheel group: %w", err)
	}

	if inWheel {
		ui.Success("User is in 'wheel' group (has sudo privileges)")
	} else {
		ui.Warning("User is NOT in 'wheel' group")
		ui.Info("This user may not have sudo privileges")
	}

	return nil
}

// createUserIfNeeded creates a user if they don't exist
// For Docker runtime, creates a system service account with /sbin/nologin
func createUserIfNeeded(cfg *config.Config, username string, ui *ui.UI) error {
	exists, err := system.UserExists(username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		ui.Infof("User %s already exists", username)
		return nil
	}

	ui.Infof("User %s does not exist", username)

	// Determine if we're using Docker (needs service account) or Podman (regular user)
	runtime := cfg.GetOrDefault("CONTAINER_RUNTIME", "docker")

	// Ask if they want to create the user
	createUser, err := ui.PromptYesNo(fmt.Sprintf("Create user %s?", username), true)
	if err != nil {
		return fmt.Errorf("failed to prompt for user creation: %w", err)
	}

	if !createUser {
		return fmt.Errorf("user %s does not exist and was not created", username)
	}

	if runtime == "docker" {
		// Create system service account for Docker
		ui.Info("Creating system service account for Docker (non-login shell)...")
		ui.Info("Note: Containers will run as this UID via PUID/PGID while Docker daemon runs as root")

		if err := system.CreateSystemUser(username, false, "/sbin/nologin"); err != nil {
			return fmt.Errorf("failed to create system user: %w", err)
		}

		ui.Successf("System service account %s created successfully", username)

		// Get and display the assigned UID/GID
		uid, err := system.GetUID(username)
		if err != nil {
			ui.Warning(fmt.Sprintf("Could not retrieve UID for %s: %v", username, err))
		} else {
			gid, err := system.GetGID(username)
			if err != nil {
				ui.Warning(fmt.Sprintf("Could not retrieve GID for %s: %v", username, err))
			} else {
				ui.Infof("Assigned UID=%d, GID=%d", uid, gid)
			}
		}
		ui.Info("This account uses /sbin/nologin (no interactive login)")

	} else {
		// Create regular user for Podman (rootless)
		ui.Infof("Creating user %s...", username)
		if err := system.CreateUser(username, true); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		ui.Successf("User %s created successfully", username)

		// Add user to wheel group for sudo access
		addToWheel, err := ui.PromptYesNo(fmt.Sprintf("Add %s to 'wheel' group (sudo privileges)?", username), true)
		if err != nil {
			return fmt.Errorf("failed to prompt for wheel group: %w", err)
		}

		if addToWheel {
			if err := system.AddUserToGroup(username, "wheel"); err != nil {
				ui.Warning(fmt.Sprintf("Failed to add user to wheel group: %v", err))
			} else {
				ui.Success("User added to 'wheel' group")
			}
		}
	}

	return nil
}

// configureSubuidSubgid configures subuid and subgid mappings for rootless containers
func configureSubuidSubgid(username string, ui *ui.UI) error {
	ui.Info("Checking subuid/subgid mappings for rootless containers...")

	// Check if subuid exists
	hasSubUID, err := system.CheckSubUIDExists(username)
	if err != nil {
		return fmt.Errorf("failed to check subuid: %w", err)
	}

	if hasSubUID {
		ui.Success("subuid mapping already configured")
	} else {
		ui.Warning("subuid mapping not found")
		ui.Info("Rootless containers require subuid/subgid mappings")
		ui.Info("These are typically created automatically when a user is created")
		ui.Info("If using an existing user, you may need to manually configure:")
		ui.Infof("  echo '%s:100000:65536' | sudo tee -a /etc/subuid", username)
	}

	// Check if subgid exists
	hasSubGID, err := system.CheckSubGIDExists(username)
	if err != nil {
		return fmt.Errorf("failed to check subgid: %w", err)
	}

	if hasSubGID {
		ui.Success("subgid mapping already configured")
	} else {
		ui.Warning("subgid mapping not found")
		ui.Infof("  echo '%s:100000:65536' | sudo tee -a /etc/subgid", username)
	}

	if !hasSubUID || !hasSubGID {
		ui.Warning("Rootless containers may not work properly without subuid/subgid mappings")
		ui.Info("Please configure them manually or use a freshly created user")
	}

	return nil
}

// setupShell optionally sets the user's shell
func setupShell(username string, ui *ui.UI) error {
	// Note: os/user.User struct doesn't include shell information
	// Would need to parse /etc/passwd or use getent to get current shell

	// Ask if they want to set the shell
	changeShell, err := ui.PromptYesNo("Would you like to change the shell?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt for shell change: %w", err)
	}

	if !changeShell {
		return nil
	}

	// Prompt for new shell
	shellOptions := []string{
		"/bin/bash",
		"/bin/zsh",
		"/bin/sh",
	}

	shellIndex, err := ui.PromptSelect("Select shell", shellOptions)
	if err != nil {
		return fmt.Errorf("failed to prompt for shell selection: %w", err)
	}

	selectedShell := shellOptions[shellIndex]

	// Set the shell
	ui.Infof("Setting shell to %s...", selectedShell)
	if err := system.SetUserShell(username, selectedShell); err != nil {
		return fmt.Errorf("failed to set shell: %w", err)
	}

	ui.Successf("Shell set to %s", selectedShell)
	return nil
}

// validatePUIDPGID validates that stored PUID/PGID match the current user's UID/GID
// If they don't match, prompts user to decide whether to update or abort
func validatePUIDPGID(cfg *config.Config, username string, ui *ui.UI) error {
	// Get current UID/GID
	currentUID, err := system.GetUID(username)
	if err != nil {
		return fmt.Errorf("failed to get UID: %w", err)
	}

	currentGID, err := system.GetGID(username)
	if err != nil {
		return fmt.Errorf("failed to get GID: %w", err)
	}

	// Check if PUID/PGID are already stored in config
	storedPUID := cfg.GetOrDefault("PUID", "")
	storedPGID := cfg.GetOrDefault("PGID", "")

	if storedPUID == "" && storedPGID == "" {
		// No stored values, this is fine
		ui.Infof("No previous PUID/PGID found, will use current values: UID=%d, GID=%d", currentUID, currentGID)
		return nil
	}

	// Parse stored values
	var expectedUID, expectedGID int
	if storedPUID != "" {
		if _, err := fmt.Sscanf(storedPUID, "%d", &expectedUID); err != nil {
			ui.Warning(fmt.Sprintf("Invalid stored PUID value: %s", storedPUID))
			return nil // Non-fatal, will overwrite
		}
	}
	if storedPGID != "" {
		if _, err := fmt.Sscanf(storedPGID, "%d", &expectedGID); err != nil {
			ui.Warning(fmt.Sprintf("Invalid stored PGID value: %s", storedPGID))
			return nil // Non-fatal, will overwrite
		}
	}

	// Validate consistency
	if expectedUID != 0 && expectedUID != currentUID {
		ui.Warning(fmt.Sprintf("UID mismatch: user %s has UID=%d but config has PUID=%d", username, currentUID, expectedUID))
		ui.Info("This can cause permission issues with existing container data")
		ui.Info("Recovery options:")
		ui.Info("  1. Update config to use current UID (recommended if user was recreated)")
		ui.Info("  2. Abort and use the original user")

		useCurrentUID, err := ui.PromptYesNo("Update PUID to match current user's UID?", true)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !useCurrentUID {
			return fmt.Errorf("UID mismatch - please use the original user or fix manually")
		}
		ui.Info("Will update PUID to match current UID")
	}

	if expectedGID != 0 && expectedGID != currentGID {
		ui.Warning(fmt.Sprintf("GID mismatch: user %s has GID=%d but config has PGID=%d", username, currentGID, expectedGID))

		useCurrentGID, err := ui.PromptYesNo("Update PGID to match current user's GID?", true)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !useCurrentGID {
			return fmt.Errorf("GID mismatch - please use the original user or fix manually")
		}
		ui.Info("Will update PGID to match current GID")
	}

	if (expectedUID == 0 || expectedUID == currentUID) && (expectedGID == 0 || expectedGID == currentGID) {
		ui.Success(fmt.Sprintf("UID/GID consistent: PUID=%d, PGID=%d", currentUID, currentGID))
	}

	return nil
}

// getTimezoneInfo gets and displays timezone information
func getTimezoneInfo(cfg *config.Config, ui *ui.UI) error {
	tz, err := system.GetTimezone()
	if err != nil {
		if loadErr := cfg.Load(); loadErr != nil {
			ui.Warning(fmt.Sprintf("Could not load existing timezone configuration (defaulting to %s): %v", defaultTimezone, loadErr))
		}

		fallback := cfg.GetOrDefault("TZ", "")
		if fallback == "" {
			fallback = defaultTimezone
		}

		ui.Warning(fmt.Sprintf("Could not determine timezone automatically (using %s): %v", fallback, err))
		tz = fallback
		ui.Infof("Using timezone: %s", tz)
	} else {
		ui.Infof("System timezone: %s", tz)
	}

	// Save timezone to config for later use
	if err := cfg.Set("TIMEZONE", tz); err != nil {
		return fmt.Errorf("failed to save timezone to config: %w", err)
	}
	if err := cfg.Set("TZ", tz); err != nil {
		return fmt.Errorf("failed to save TZ to config: %w", err)
	}

	return nil
}

// RunUserSetup executes the user configuration step
func RunUserSetup(cfg *config.Config, ui *ui.UI) error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(cfg, userCompletionMarker, "user-configured")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		ui.Info("User configuration already completed (marker found)")
		ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + userCompletionMarker)
		return nil
	}

	ui.Header("User Configuration")
	ui.Info("Configuring user account for homelab services...")
	ui.Print("")

	// Prompt for username
	ui.Step("Select Homelab User")
	username, err := getConfiguredUsername(cfg, ui)
	if err != nil {
		return fmt.Errorf("failed to read configured username: %w", err)
	}

	if username == "" {
		username, err = promptForUser(ui)
		if err != nil {
			return fmt.Errorf("failed to get username: %w", err)
		}
	}

	// Validate or create user
	ui.Step("Validating User Account")
	if err := validateUser(username, ui); err != nil {
		// User doesn't exist, try to create
		if err := createUserIfNeeded(cfg, username, ui); err != nil {
			return fmt.Errorf("user setup failed: %w", err)
		}
		// Validate again after creation
		if err := validateUser(username, ui); err != nil {
			return fmt.Errorf("user validation failed after creation: %w", err)
		}
	}

	// Check if stored PUID/PGID exist and validate against current user
	ui.Step("Validating UID/GID Consistency")
	if err := validatePUIDPGID(cfg, username, ui); err != nil {
		return fmt.Errorf("UID/GID validation failed: %w", err)
	}

	// Get UID and GID for config
	uid, err := system.GetUID(username)
	if err != nil {
		return fmt.Errorf("failed to get UID: %w", err)
	}

	gid, err := system.GetGID(username)
	if err != nil {
		return fmt.Errorf("failed to get GID: %w", err)
	}

	// Configure subuid/subgid
	ui.Step("Checking Rootless Container Configuration")
	if err := configureSubuidSubgid(username, ui); err != nil {
		return fmt.Errorf("failed to configure subuid/subgid: %w", err)
	}

	// Optional: Setup shell
	ui.Step("Shell Configuration")
	if err := setupShell(username, ui); err != nil {
		ui.Warning(fmt.Sprintf("Shell setup failed: %v", err))
		// Non-critical error, continue
	}

	// Get timezone information
	ui.Step("System Information")
	if err := getTimezoneInfo(cfg, ui); err != nil {
		ui.Warning(fmt.Sprintf("Failed to get timezone info: %v", err))
		// Non-critical error, continue
	}

	// Save configuration
	ui.Step("Saving Configuration")
	if err := cfg.Set("HOMELAB_USER", username); err != nil {
		return fmt.Errorf("failed to save homelab user: %w", err)
	}

	if err := cfg.Set("PUID", fmt.Sprintf("%d", uid)); err != nil {
		return fmt.Errorf("failed to save PUID: %w", err)
	}

	if err := cfg.Set("PGID", fmt.Sprintf("%d", gid)); err != nil {
		return fmt.Errorf("failed to save PGID: %w", err)
	}

	ui.Print("")
	ui.Separator()
	ui.Success("✓ User configuration completed successfully")
	ui.Infof("Homelab user: %s (UID: %d, GID: %d)", username, uid, gid)

	// Create completion marker
	if err := cfg.MarkComplete(userCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
