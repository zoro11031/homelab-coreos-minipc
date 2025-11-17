package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsPackageInstalled checks if a package is installed
func IsPackageInstalled(packageName string) (bool, error) {
	// Use rpm -q to check if package is installed
	cmd := exec.Command("rpm", "-q", packageName)
	err := cmd.Run()

	if err == nil {
		// Package is installed
		return true, nil
	}

	// Check if it's an exit error
	if exitErr, ok := err.(*exec.ExitError); ok {
		// rpm -q returns exit code 1 if package is not installed
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	// Other error occurred
	return false, fmt.Errorf("failed to check package %s: %w", packageName, err)
}

// CheckMultiplePackages checks if multiple packages are installed
// Returns a map of package name -> installed status
func CheckMultiplePackages(packages []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, pkg := range packages {
		installed, err := IsPackageInstalled(pkg)
		if err != nil {
			return nil, fmt.Errorf("error checking package %s: %w", pkg, err)
		}
		result[pkg] = installed
	}

	return result, nil
}

// CommandExists checks if a command is available in PATH
func CommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetPackageVersion returns the version of an installed package
func GetPackageVersion(packageName string) (string, error) {
	cmd := exec.Command("rpm", "-q", "--queryformat", "%{VERSION}-%{RELEASE}", packageName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %w", packageName, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// IsRpmOstreeSystem checks if the system is using rpm-ostree
func IsRpmOstreeSystem() bool {
	return CommandExists("rpm-ostree")
}

// GetRpmOstreeStatus returns the current rpm-ostree status
func GetRpmOstreeStatus() (string, error) {
	if !IsRpmOstreeSystem() {
		return "", fmt.Errorf("not an rpm-ostree system")
	}

	cmd := exec.Command("rpm-ostree", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get rpm-ostree status: %w", err)
	}

	return string(output), nil
}

// ListLayeredPackages returns a list of layered packages on rpm-ostree system
func ListLayeredPackages() ([]string, error) {
	if !IsRpmOstreeSystem() {
		return nil, fmt.Errorf("not an rpm-ostree system")
	}

	cmd := exec.Command("rpm-ostree", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list layered packages: %w", err)
	}

	// Note: Parsing JSON would require encoding/json
	// For now, return raw output as string slice
	// This can be enhanced later with proper JSON parsing
	return strings.Split(string(output), "\n"), nil
}
