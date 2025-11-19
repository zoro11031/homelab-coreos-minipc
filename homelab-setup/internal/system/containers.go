// Package system provides low-level system operations for the homelab setup tool,
// including filesystem operations, package management, service control, user/group
// management, network utilities, and container runtime abstraction. All operations
// that interact with the OS are encapsulated here.
package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// ContainerRuntime represents a container runtime type
type ContainerRuntime string

const (
	RuntimePodman ContainerRuntime = "podman"
	RuntimeDocker ContainerRuntime = "docker"
	RuntimeNone   ContainerRuntime = "none"
)

// DetectRuntime detects which container runtime is available
// Returns the first runtime found: podman, docker, or none
func DetectRuntime() (ContainerRuntime, error) {
	// Check for podman first (preferred for rootless)
	if CommandExists("podman") {
		return RuntimePodman, nil
	}

	// Check for docker
	if CommandExists("docker") {
		return RuntimeDocker, nil
	}

	return RuntimeNone, fmt.Errorf("no container runtime found (podman or docker)")
}

// GetComposeCommand returns the appropriate compose command for the runtime
func GetComposeCommand(runtime ContainerRuntime) (string, error) {
	switch runtime {
	case RuntimePodman:
		// Check if podman-compose is available
		if CommandExists("podman-compose") {
			return "podman-compose", nil
		}
		// Check if podman compose plugin is available
		cmd := exec.Command("podman", "compose", "version")
		if err := cmd.Run(); err == nil {
			return "podman compose", nil
		}
		return "", fmt.Errorf("neither podman-compose nor podman compose plugin found")

	case RuntimeDocker:
		// Check if docker-compose is available
		if CommandExists("docker-compose") {
			return "docker-compose", nil
		}
		// Check if docker compose plugin is available
		cmd := exec.Command("docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return "docker compose", nil
		}
		return "", fmt.Errorf("neither docker-compose nor docker compose plugin found")

	default:
		return "", fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

// GetRuntimeVersion returns the version of the container runtime
func GetRuntimeVersion(runtime ContainerRuntime) (string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "--version")
	case RuntimeDocker:
		cmd = exec.Command("docker", "--version")
	default:
		return "", fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get runtime version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ListContainers lists all containers (running and stopped)
func ListContainers(runtime ContainerRuntime) ([]string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "ps", "-a", "--format", "{{.Names}}")
	case RuntimeDocker:
		cmd = exec.Command("docker", "ps", "-a", "--format", "{{.Names}}")
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string
	for _, line := range lines {
		if line != "" {
			containers = append(containers, line)
		}
	}

	return containers, nil
}

// ListRunningContainers lists only running containers
func ListRunningContainers(runtime ContainerRuntime) ([]string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "ps", "--format", "{{.Names}}")
	case RuntimeDocker:
		cmd = exec.Command("docker", "ps", "--format", "{{.Names}}")
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list running containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string
	for _, line := range lines {
		if line != "" {
			containers = append(containers, line)
		}
	}

	return containers, nil
}

// IsContainerRunning checks if a specific container is running
func IsContainerRunning(runtime ContainerRuntime, containerName string) (bool, error) {
	running, err := ListRunningContainers(runtime)
	if err != nil {
		return false, err
	}

	for _, name := range running {
		if name == containerName {
			return true, nil
		}
	}

	return false, nil
}

// GetContainerLogs returns logs for a specific container
func GetContainerLogs(runtime ContainerRuntime, containerName string, lines int) (string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "logs", "--tail", fmt.Sprintf("%d", lines), containerName)
	case RuntimeDocker:
		cmd = exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", lines), containerName)
	default:
		return "", fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", containerName, err)
	}

	return string(output), nil
}

// InspectContainer returns detailed information about a container
func InspectContainer(runtime ContainerRuntime, containerName string) (string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "inspect", containerName)
	case RuntimeDocker:
		cmd = exec.Command("docker", "inspect", containerName)
	default:
		return "", fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}

	return string(output), nil
}

// ListNetworks lists container networks
func ListNetworks(runtime ContainerRuntime) ([]string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "network", "ls", "--format", "{{.Name}}")
	case RuntimeDocker:
		cmd = exec.Command("docker", "network", "ls", "--format", "{{.Name}}")
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var networks []string
	for _, line := range lines {
		if line != "" {
			networks = append(networks, line)
		}
	}

	return networks, nil
}

// PullImage pulls a container image
func PullImage(runtime ContainerRuntime, imageName string) error {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "pull", imageName)
	case RuntimeDocker:
		cmd = exec.Command("docker", "pull", imageName)
	default:
		return fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w\nOutput: %s", imageName, err, string(output))
	}

	return nil
}

// ListImages lists container images
func ListImages(runtime ContainerRuntime) ([]string, error) {
	var cmd *exec.Cmd

	switch runtime {
	case RuntimePodman:
		cmd = exec.Command("podman", "images", "--format", "{{.Repository}}:{{.Tag}}")
	case RuntimeDocker:
		cmd = exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var images []string
	for _, line := range lines {
		if line != "" && line != "<none>:<none>" {
			images = append(images, line)
		}
	}

	return images, nil
}

// CheckRootless returns true if the container runtime is running in rootless mode
func CheckRootless(runtime ContainerRuntime) (bool, error) {
	switch runtime {
	case RuntimePodman:
		cmd := exec.Command("podman", "info", "--format", "{{.Host.Security.Rootless}}")
		output, err := cmd.Output()
		if err != nil {
			return false, fmt.Errorf("failed to check rootless mode: %w", err)
		}
		return strings.TrimSpace(string(output)) == "true", nil

	case RuntimeDocker:
		// Check Docker's SecurityOptions for rootless indicators
		cmd := exec.Command("docker", "info", "--format", "{{.SecurityOptions}}")
		output, err := cmd.Output()
		if err != nil {
			return false, fmt.Errorf("failed to check docker security options: %w", err)
		}
		secOpts := strings.ToLower(string(output))
		// Look for rootless indicators in security options
		return strings.Contains(secOpts, "rootless") || strings.Contains(secOpts, "userns"), nil

	default:
		return false, fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

// CheckDockerService verifies that the Docker daemon service is running
func CheckDockerService() error {
	cmd := exec.Command("systemctl", "is-active", "docker.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker.service is not active: %w", err)
	}
	return nil
}

// CheckDockerComposeV2 checks for Docker Compose V2 plugin (docker compose)
func CheckDockerComposeV2() error {
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose plugin not available: %w", err)
	}
	return nil
}

// CheckDockerComposeV1 checks for Docker Compose V1 standalone (docker-compose)
func CheckDockerComposeV1() error {
	cmd := exec.Command("docker-compose", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose standalone not available: %w", err)
	}
	return nil
}
