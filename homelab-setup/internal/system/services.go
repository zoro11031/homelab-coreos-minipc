package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ServiceManager handles systemd service operations
type ServiceManager struct{}

// NewServiceManager creates a new ServiceManager instance
func NewServiceManager() *ServiceManager {
	return &ServiceManager{}
}

// ServiceExists checks if a systemd service unit file exists
func (sm *ServiceManager) ServiceExists(serviceName string) (bool, error) {
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
func (sm *ServiceManager) GetServiceLocation(serviceName string) (string, error) {
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

// IsActive checks if a service is currently active
func (sm *ServiceManager) IsActive(serviceName string) (bool, error) {
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

// IsEnabled checks if a service is enabled to start on boot
func (sm *ServiceManager) IsEnabled(serviceName string) (bool, error) {
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

// Enable enables a service to start on boot
func (sm *ServiceManager) Enable(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "enable", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// Disable disables a service from starting on boot
func (sm *ServiceManager) Disable(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "disable", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// Start starts a service
func (sm *ServiceManager) Start(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// Stop stops a service
func (sm *ServiceManager) Stop(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// Restart restarts a service
func (sm *ServiceManager) Restart(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "restart", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// Reload reloads a service configuration
func (sm *ServiceManager) Reload(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "reload", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload service %s: %w\nOutput: %s", serviceName, err, string(output))
	}
	return nil
}

// DaemonReload reloads systemd manager configuration
func (sm *ServiceManager) DaemonReload() error {
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// GetStatus returns the status output for a service
func (sm *ServiceManager) GetStatus(serviceName string) (string, error) {
	cmd := exec.Command("systemctl", "status", serviceName, "--no-pager", "-l")
	output, err := cmd.CombinedOutput()

	// Note: systemctl status returns non-zero for inactive services
	// We still want the output in that case
	return string(output), err
}

// GetJournalLogs returns recent journal logs for a service
func (sm *ServiceManager) GetJournalLogs(serviceName string, lines int) (string, error) {
	cmd := exec.Command("sudo", "journalctl", "-u", serviceName, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", serviceName, err)
	}
	return string(output), nil
}

// ListUnits lists all systemd units matching a pattern
func (sm *ServiceManager) ListUnits(pattern string) ([]string, error) {
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
