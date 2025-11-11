package steps

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// UserConfigurator handles user and group configuration
type UserConfigurator struct {
	users   *system.UserManager
	config  *config.Config
	ui      *ui.UI
	markers *config.Markers
}

// NewUserConfigurator creates a new UserConfigurator instance
func NewUserConfigurator(users *system.UserManager, cfg *config.Config, ui *ui.UI, markers *config.Markers) *UserConfigurator {
	return &UserConfigurator{
		users:   users,
		config:  cfg,
		ui:      ui,
		markers: markers,
	}
}

// PromptForUser prompts for a homelab username or allows using current user
func (u *UserConfigurator) PromptForUser() (string, error) {
	// Get current user
	currentUser, err := u.users.GetCurrentUser()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	u.ui.Info(fmt.Sprintf("Current user: %s (UID: %s, GID: %s)", currentUser.Username, currentUser.Uid, currentUser.Gid))
	u.ui.Print("")

	// Prompt for username with current user as default
	username, err := u.ui.PromptInput("Enter homelab username (or press Enter to use current user)", currentUser.Username)
	if err != nil {
		return "", fmt.Errorf("failed to prompt for username: %w", err)
	}

	// Validate username
	if err := common.ValidateUsername(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}

	return username, nil
}

// ValidateUser checks if a user exists and can be used for homelab
func (u *UserConfigurator) ValidateUser(username string) error {
	exists, err := u.users.UserExists(username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("user %s does not exist", username)
	}

	// Get user info to display
	userInfo, err := u.users.GetUserInfo(username)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	u.ui.Successf("User %s found (UID: %s, GID: %s)", username, userInfo.Uid, userInfo.Gid)

	// Check user groups
	groups, err := u.users.GetUserGroups(username)
	if err != nil {
		return fmt.Errorf("failed to get user groups: %w", err)
	}

	u.ui.Infof("User groups: %v", groups)

	// Check if user is in wheel group (recommended for sudo)
	inWheel, err := u.users.IsUserInGroup(username, "wheel")
	if err != nil {
		return fmt.Errorf("failed to check wheel group: %w", err)
	}

	if inWheel {
		u.ui.Success("User is in 'wheel' group (has sudo privileges)")
	} else {
		u.ui.Warning("User is NOT in 'wheel' group")
		u.ui.Info("This user may not have sudo privileges")
	}

	return nil
}

// CreateUserIfNeeded creates a user if they don't exist
func (u *UserConfigurator) CreateUserIfNeeded(username string) error {
	exists, err := u.users.UserExists(username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		u.ui.Infof("User %s already exists", username)
		return nil
	}

	u.ui.Infof("User %s does not exist", username)

	// Ask if they want to create the user
	createUser, err := u.ui.PromptYesNo(fmt.Sprintf("Create user %s?", username), true)
	if err != nil {
		return fmt.Errorf("failed to prompt for user creation: %w", err)
	}

	if !createUser {
		return fmt.Errorf("user %s does not exist and was not created", username)
	}

	// Create user with home directory
	u.ui.Infof("Creating user %s...", username)
	if err := u.users.CreateUser(username, true); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	u.ui.Successf("User %s created successfully", username)

	// Add user to wheel group for sudo access
	addToWheel, err := u.ui.PromptYesNo(fmt.Sprintf("Add %s to 'wheel' group (sudo privileges)?", username), true)
	if err != nil {
		return fmt.Errorf("failed to prompt for wheel group: %w", err)
	}

	if addToWheel {
		if err := u.users.AddUserToGroup(username, "wheel"); err != nil {
			u.ui.Warning(fmt.Sprintf("Failed to add user to wheel group: %v", err))
		} else {
			u.ui.Success("User added to 'wheel' group")
		}
	}

	return nil
}

// ConfigureSubuidSubgid configures subuid and subgid mappings for rootless containers
func (u *UserConfigurator) ConfigureSubuidSubgid(username string) error {
	u.ui.Info("Checking subuid/subgid mappings for rootless containers...")

	// Check if subuid exists
	hasSubUID, err := u.users.CheckSubUIDExists(username)
	if err != nil {
		return fmt.Errorf("failed to check subuid: %w", err)
	}

	if hasSubUID {
		u.ui.Success("subuid mapping already configured")
	} else {
		u.ui.Warning("subuid mapping not found")
		u.ui.Info("Rootless containers require subuid/subgid mappings")
		u.ui.Info("These are typically created automatically when a user is created")
		u.ui.Info("If using an existing user, you may need to manually configure:")
		u.ui.Infof("  echo '%s:100000:65536' | sudo tee -a /etc/subuid", username)
	}

	// Check if subgid exists
	hasSubGID, err := u.users.CheckSubGIDExists(username)
	if err != nil {
		return fmt.Errorf("failed to check subgid: %w", err)
	}

	if hasSubGID {
		u.ui.Success("subgid mapping already configured")
	} else {
		u.ui.Warning("subgid mapping not found")
		u.ui.Infof("  echo '%s:100000:65536' | sudo tee -a /etc/subgid", username)
	}

	if !hasSubUID || !hasSubGID {
		u.ui.Warning("Rootless containers may not work properly without subuid/subgid mappings")
		u.ui.Info("Please configure them manually or use a freshly created user")
	}

	return nil
}

// SetupShell optionally sets the user's shell
func (u *UserConfigurator) SetupShell(username string) error {
	// Note: os/user.User struct doesn't include shell information
	// Would need to parse /etc/passwd or use getent to get current shell

	// Ask if they want to set the shell
	changeShell, err := u.ui.PromptYesNo("Would you like to change the shell?", false)
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

	shellIndex, err := u.ui.PromptSelect("Select shell", shellOptions)
	if err != nil {
		return fmt.Errorf("failed to prompt for shell selection: %w", err)
	}

	selectedShell := shellOptions[shellIndex]

	// Set the shell
	u.ui.Infof("Setting shell to %s...", selectedShell)
	if err := u.users.SetUserShell(username, selectedShell); err != nil {
		return fmt.Errorf("failed to set shell: %w", err)
	}

	u.ui.Successf("Shell set to %s", selectedShell)
	return nil
}

// GetTimezoneInfo gets and displays timezone information
func (u *UserConfigurator) GetTimezoneInfo() error {
	tz, err := system.GetTimezone()
	if err != nil {
		u.ui.Warning(fmt.Sprintf("Could not determine timezone: %v", err))
		return nil
	}

	u.ui.Infof("System timezone: %s", tz)

	// Save timezone to config for later use
	if err := u.config.Set("TIMEZONE", tz); err != nil {
		return fmt.Errorf("failed to save timezone to config: %w", err)
	}

	return nil
}

// Run executes the user configuration step
func (u *UserConfigurator) Run() error {
	// Check if already completed
	exists, err := u.markers.Exists("user-configured")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if exists {
		u.ui.Info("User configuration already completed (marker found)")
		u.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/user-configured")
		return nil
	}

	u.ui.Header("User Configuration")
	u.ui.Info("Configuring user account for homelab services...")
	u.ui.Print("")

	// Prompt for username
	u.ui.Step("Select Homelab User")
	username, err := u.PromptForUser()
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}

	// Validate or create user
	u.ui.Step("Validating User Account")
	if err := u.ValidateUser(username); err != nil {
		// User doesn't exist, try to create
		if err := u.CreateUserIfNeeded(username); err != nil {
			return fmt.Errorf("user setup failed: %w", err)
		}
		// Validate again after creation
		if err := u.ValidateUser(username); err != nil {
			return fmt.Errorf("user validation failed after creation: %w", err)
		}
	}

	// Get UID and GID for config
	uid, err := u.users.GetUID(username)
	if err != nil {
		return fmt.Errorf("failed to get UID: %w", err)
	}

	gid, err := u.users.GetGID(username)
	if err != nil {
		return fmt.Errorf("failed to get GID: %w", err)
	}

	// Configure subuid/subgid
	u.ui.Step("Checking Rootless Container Configuration")
	if err := u.ConfigureSubuidSubgid(username); err != nil {
		return fmt.Errorf("failed to configure subuid/subgid: %w", err)
	}

	// Optional: Setup shell
	u.ui.Step("Shell Configuration")
	if err := u.SetupShell(username); err != nil {
		u.ui.Warning(fmt.Sprintf("Shell setup failed: %v", err))
		// Non-critical error, continue
	}

	// Get timezone information
	u.ui.Step("System Information")
	if err := u.GetTimezoneInfo(); err != nil {
		u.ui.Warning(fmt.Sprintf("Failed to get timezone info: %v", err))
		// Non-critical error, continue
	}

	// Save configuration
	u.ui.Step("Saving Configuration")
	if err := u.config.Set("HOMELAB_USER", username); err != nil {
		return fmt.Errorf("failed to save homelab user: %w", err)
	}

	if err := u.config.Set("PUID", fmt.Sprintf("%d", uid)); err != nil {
		return fmt.Errorf("failed to save PUID: %w", err)
	}

	if err := u.config.Set("PGID", fmt.Sprintf("%d", gid)); err != nil {
		return fmt.Errorf("failed to save PGID: %w", err)
	}

	u.ui.Print("")
	u.ui.Separator()
	u.ui.Success("✓ User configuration completed successfully")
	u.ui.Infof("Homelab user: %s (UID: %d, GID: %d)", username, uid, gid)

	// Create completion marker
	if err := u.markers.Create("user-configured"); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
