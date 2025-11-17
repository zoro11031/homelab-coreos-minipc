package system

import (
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

// UserExists checks if a user exists
func UserExists(username string) (bool, error) {
	_, err := user.Lookup(username)
	if err == nil {
		return true, nil
	}

	// Check if it's a "user not found" error
	if _, ok := err.(user.UnknownUserError); ok {
		return false, nil
	}

	// Some other error
	return false, fmt.Errorf("failed to lookup user %s: %w", username, err)
}

// GroupExists checks if a group exists
func GroupExists(groupName string) (bool, error) {
	_, err := user.LookupGroup(groupName)
	if err == nil {
		return true, nil
	}

	// Check if it's a "group not found" error
	if _, ok := err.(user.UnknownGroupError); ok {
		return false, nil
	}

	// Some other error
	return false, fmt.Errorf("failed to lookup group %s: %w", groupName, err)
}

// GetUserInfo returns information about a user
func GetUserInfo(username string) (*user.User, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info for %s: %w", username, err)
	}
	return u, nil
}

// GetUID returns the UID for a username
func GetUID(username string) (int, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, fmt.Errorf("failed to get UID for %s: %w", username, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("invalid UID for %s: %w", username, err)
	}

	return uid, nil
}

// GetGID returns the primary GID for a username
func GetGID(username string) (int, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, fmt.Errorf("failed to get GID for %s: %w", username, err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return 0, fmt.Errorf("invalid GID for %s: %w", username, err)
	}

	return gid, nil
}

// GetUserGroups returns all groups a user belongs to
func GetUserGroups(username string) ([]string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user %s: %w", username, err)
	}

	gids, err := u.GroupIds()
	if err != nil {
		return nil, fmt.Errorf("failed to get group IDs for %s: %w", username, err)
	}

	var groups []string
	for _, gid := range gids {
		g, err := user.LookupGroupId(gid)
		if err != nil {
			// Skip groups we can't lookup
			continue
		}
		groups = append(groups, g.Name)
	}

	return groups, nil
}

// IsUserInGroup checks if a user is in a specific group
func IsUserInGroup(username, groupName string) (bool, error) {
	groups, err := GetUserGroups(username)
	if err != nil {
		return false, err
	}

	for _, g := range groups {
		if g == groupName {
			return true, nil
		}
	}

	return false, nil
}

// CreateUser creates a new user
func CreateUser(username string, createHome bool) error {
	args := []string{"useradd"}

	if createHome {
		args = append(args, "-m")
	}

	args = append(args, username)

	cmd := exec.Command("sudo", append([]string{"-n"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create user %s: %w\nOutput: %s", username, err, string(output))
	}

	return nil
}

// DeleteUser deletes a user
func DeleteUser(username string, removeHome bool) error {
	args := []string{"userdel"}

	if removeHome {
		args = append(args, "-r")
	}

	args = append(args, username)

	cmd := exec.Command("sudo", append([]string{"-n"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete user %s: %w\nOutput: %s", username, err, string(output))
	}

	return nil
}

// AddUserToGroup adds a user to a group
func AddUserToGroup(username, groupName string) error {
	cmd := exec.Command("sudo", "-n", "usermod", "-aG", groupName, username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add user %s to group %s: %w\nOutput: %s", username, groupName, err, string(output))
	}

	return nil
}

// SetUserShell sets the login shell for a user
func SetUserShell(username, shell string) error {
	cmd := exec.Command("sudo", "-n", "usermod", "-s", shell, username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set shell for user %s: %w\nOutput: %s", username, err, string(output))
	}

	return nil
}

// GetCurrentUser returns the current user information
func GetCurrentUser() (*user.User, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return u, nil
}

// CheckSubUIDExists checks if a user has subuid mappings
func CheckSubUIDExists(username string) (bool, error) {
	cmd := exec.Command("grep", "-q", fmt.Sprintf("^%s:", username), "/etc/subuid")
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to check subuid for %s: %w", username, err)
}

// CheckSubGIDExists checks if a user has subgid mappings
func CheckSubGIDExists(username string) (bool, error) {
	cmd := exec.Command("grep", "-q", fmt.Sprintf("^%s:", username), "/etc/subgid")
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to check subgid for %s: %w", username, err)
}

// GetTimezone returns the system timezone
func GetTimezone() (string, error) {
	cmd := exec.Command("timedatectl", "show", "--property=Timezone", "--value")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("timedatectl failed: %w", err)
	}

	timezone := strings.TrimSpace(string(output))
	if timezone == "" {
		return "", fmt.Errorf("timedatectl returned an empty timezone")
	}

	return timezone, nil
}
