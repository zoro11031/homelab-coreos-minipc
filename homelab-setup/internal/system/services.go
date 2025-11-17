package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ServiceExists checks if a systemd service unit file exists
func ServiceExists(serviceName string) (bool, error) {
	// Check in standard systemd locations
	locations := []string{
		filepath.Join("/etc/systemd/system", serviceName),
		filepath.Join("/usr/lib/systemd/system", serviceName),
		filepath.Join("/lib/systemd/system", serviceName),
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			// Some other error (permission denied, etc.)
			return false, fmt.Errorf("error checking service at %s: %w", location, err)
		}
	}

	return false, nil
}

// GetServiceLocation returns the path to a service unit file
func GetServiceLocation(serviceName string) (string, error) {
	locations := []string{
		filepath.Join("/etc/systemd/system", serviceName),
		filepath.Join("/usr/lib/systemd/system", serviceName),
		filepath.Join("/lib/systemd/system", serviceName),
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return location, nil
		}
	}

	return "", fmt.Errorf("service %s not found", serviceName)
}

// IsServiceActive checks if a service is currently active
func IsServiceActive(serviceName string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", serviceName)
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		// systemctl is-active returns non-zero if inactive
		if exitErr.ExitCode() != 0 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to check service status: %w", err)
}

// IsServiceEnabled checks if a service is enabled to start on boot
func IsServiceEnabled(serviceName string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", "--quiet", serviceName)
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		// systemctl is-enabled returns non-zero if disabled
		if exitErr.ExitCode() != 0 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to check if service is enabled: %w", err)
}

// EnableService enables a service to start on boot
func EnableService(serviceName string) error {
	cmd := exec.Command("sudo", "-n", "systemctl", "enable", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// DisableService disables a service from starting on boot
func DisableService(serviceName string) error {
	cmd := exec.Command("sudo", "-n", "systemctl", "disable", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// StartService starts a service
func StartService(serviceName string) error {
	cmd := exec.Command("sudo", "-n", "systemctl", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// StopService stops a service
func StopService(serviceName string) error {
	cmd := exec.Command("sudo", "-n", "systemctl", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// RestartService restarts a service
func RestartService(serviceName string) error {
	cmd := exec.Command("sudo", "-n", "systemctl", "restart", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// ReloadService reloads a service configuration
func ReloadService(serviceName string) error {
	cmd := exec.Command("sudo", "-n", "systemctl", "reload", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// SystemdDaemonReload reloads systemd manager configuration
func SystemdDaemonReload() error {
	cmd := exec.Command("sudo", "-n", "systemctl", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// GetServiceStatus returns the status output for a service
func GetServiceStatus(serviceName string) (string, error) {
	cmd := exec.Command("systemctl", "status", serviceName, "--no-pager", "-l")
	output, err := cmd.CombinedOutput()

	// Note: systemctl status returns non-zero for inactive services
	// We still want the output in that case
	return string(output), err
}

// GetServiceJournalLogs returns recent journal logs for a service
func GetServiceJournalLogs(serviceName string, lines int) (string, error) {
	cmd := exec.Command("sudo", "-n", "journalctl", "-u", serviceName, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", serviceName, err)
	}
	return string(output), nil
}

// ListSystemdUnits lists all systemd units matching a pattern
func ListSystemdUnits(pattern string) ([]string, error) {
	cmd := exec.Command("systemctl", "list-units", pattern, "--no-pager", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var units []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Extract unit name (first field)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			units = append(units, fields[0])
		}
	}

	return units, nil
}

// RunSystemCommand runs a shell command with the given arguments
func RunSystemCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command %s: %w", command, err)
	}
	return nil
}
