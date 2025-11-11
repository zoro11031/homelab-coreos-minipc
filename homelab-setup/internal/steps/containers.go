package steps

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// ContainerSetup handles container runtime setup and validation
type ContainerSetup struct {
	containers *system.ContainerManager
	config     *config.Config
	ui         *ui.UI
	markers    *config.Markers
}

// NewContainerSetup creates a new ContainerSetup instance
func NewContainerSetup(containers *system.ContainerManager, cfg *config.Config, ui *ui.UI, markers *config.Markers) *ContainerSetup {
	return &ContainerSetup{
		containers: containers,
		config:     cfg,
		ui:         ui,
		markers:    markers,
	}
}

// DetectRuntime detects and displays the available container runtime
func (c *ContainerSetup) DetectRuntime() (system.ContainerRuntime, error) {
	c.ui.Info("Detecting container runtime...")

	runtime, err := c.containers.DetectRuntime()
	if err != nil {
		c.ui.Error("No container runtime found")
		c.ui.Info("Please install either Podman or Docker:")
		c.ui.Info("  For Podman: sudo rpm-ostree install podman podman-compose")
		c.ui.Info("  For Docker: sudo rpm-ostree install docker docker-compose")
		c.ui.Info("Then reboot: sudo systemctl reboot")
		return system.RuntimeNone, fmt.Errorf("no container runtime available: %w", err)
	}

	c.ui.Successf("Detected runtime: %s", runtime)

	// Get and display runtime version
	version, err := c.containers.GetRuntimeVersion(runtime)
	if err != nil {
		c.ui.Warning(fmt.Sprintf("Could not get runtime version: %v", err))
	} else {
		c.ui.Infof("Version: %s", version)
	}

	return runtime, nil
}

// CheckComposeCommand verifies compose command is available
func (c *ContainerSetup) CheckComposeCommand(runtime system.ContainerRuntime) error {
	c.ui.Info("Checking for compose command...")

	composeCmd, err := c.containers.GetComposeCommand(runtime)
	if err != nil {
		c.ui.Warning(fmt.Sprintf("Compose command not found: %v", err))
		c.ui.Info("You can install it later:")
		if runtime == system.RuntimePodman {
			c.ui.Info("  sudo rpm-ostree install podman-compose")
		} else {
			c.ui.Info("  sudo rpm-ostree install docker-compose")
		}
		c.ui.Info("Or use the compose plugin if available")
		return fmt.Errorf("compose command not available")
	}

	c.ui.Successf("Compose command available: %s", composeCmd)
	return nil
}

// ValidateRootless validates rootless container configuration
func (c *ContainerSetup) ValidateRootless(runtime system.ContainerRuntime, user string) error {
	c.ui.Infof("Checking rootless container configuration for user %s...", user)

	// Check if running in rootless mode
	isRootless, err := c.containers.CheckRootless(runtime)
	if err != nil {
		c.ui.Warning(fmt.Sprintf("Could not determine rootless status: %v", err))
		return nil // Non-critical error
	}

	if isRootless {
		c.ui.Success("Container runtime is running in rootless mode")
	} else {
		c.ui.Warning("Container runtime appears to be running with root privileges")
		c.ui.Info("For better security, consider configuring rootless mode")
		c.ui.Info("See: https://docs.podman.io/en/latest/markdown/podman.1.html#rootless-mode")
	}

	return nil
}

// TestContainerOperations performs basic container operations test
func (c *ContainerSetup) TestContainerOperations(runtime system.ContainerRuntime) error {
	c.ui.Info("Testing basic container operations...")

	// Try to list existing containers
	containers, err := c.containers.ListContainers(runtime)
	if err != nil {
		c.ui.Warning(fmt.Sprintf("Could not list containers: %v", err))
		return nil // Non-critical
	}

	if len(containers) > 0 {
		c.ui.Infof("Found %d existing container(s):", len(containers))
		for _, container := range containers {
			c.ui.Printf("  - %s", container)
		}
	} else {
		c.ui.Info("No existing containers found")
	}

	// Try to list networks
	networks, err := c.containers.ListNetworks(runtime)
	if err != nil {
		c.ui.Warning(fmt.Sprintf("Could not list networks: %v", err))
		return nil // Non-critical
	}

	if len(networks) > 0 {
		c.ui.Infof("Available networks: %d", len(networks))
	}

	// Try to list images
	images, err := c.containers.ListImages(runtime)
	if err != nil {
		c.ui.Warning(fmt.Sprintf("Could not list images: %v", err))
		return nil // Non-critical
	}

	if len(images) > 0 {
		c.ui.Infof("Found %d existing image(s)", len(images))
	} else {
		c.ui.Info("No container images found")
	}

	c.ui.Success("Container operations test completed")
	return nil
}

// TestImagePull optionally tests pulling a container image
func (c *ContainerSetup) TestImagePull(runtime system.ContainerRuntime) error {
	c.ui.Info("Container image pull test")

	// Ask if they want to test image pull
	testPull, err := c.ui.PromptYesNo("Test pulling a container image? (pulls busybox:latest)", false)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if !testPull {
		c.ui.Info("Skipping image pull test")
		return nil
	}

	c.ui.Info("Pulling busybox:latest...")
	if err := c.containers.PullImage(runtime, "busybox:latest"); err != nil {
		c.ui.Error(fmt.Sprintf("Failed to pull image: %v", err))
		c.ui.Info("This might indicate:")
		c.ui.Info("  1. No internet connectivity")
		c.ui.Info("  2. Registry access issues")
		c.ui.Info("  3. Container runtime configuration problems")
		return fmt.Errorf("image pull test failed: %w", err)
	}

	c.ui.Success("Successfully pulled busybox:latest")
	return nil
}

// DisplayRecommendations displays recommendations for container setup
func (c *ContainerSetup) DisplayRecommendations(runtime system.ContainerRuntime) {
	c.ui.Print("")
	c.ui.Info("Container Runtime Recommendations:")
	c.ui.Print("")

	if runtime == system.RuntimePodman {
		c.ui.Info("Podman Tips:")
		c.ui.Info("  - Rootless mode is recommended for security")
		c.ui.Info("  - Use systemd to manage containers as services")
		c.ui.Info("  - Enable linger for user: loginctl enable-linger <user>")
		c.ui.Info("  - Configure auto-update: podman auto-update")
	} else if runtime == system.RuntimeDocker {
		c.ui.Info("Docker Tips:")
		c.ui.Info("  - Add user to docker group for rootless operation")
		c.ui.Info("  - Enable Docker service: sudo systemctl enable docker")
		c.ui.Info("  - Configure docker-compose for systemd management")
	}

	c.ui.Print("")
}

// Run executes the container setup step
func (c *ContainerSetup) Run() error {
	// Check if already completed
	exists, err := c.markers.Exists("containers-configured")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if exists {
		c.ui.Info("Container runtime already configured (marker found)")
		c.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/containers-configured")
		return nil
	}

	c.ui.Header("Container Runtime Setup")
	c.ui.Info("Configuring container runtime for homelab services...")
	c.ui.Print("")

	// Get homelab user from config
	homelabUser := c.config.GetOrDefault("HOMELAB_USER", "")
	if homelabUser == "" {
		return fmt.Errorf("homelab user not configured (run user configuration first)")
	}

	// Detect runtime
	c.ui.Step("Detecting Container Runtime")
	runtime, err := c.DetectRuntime()
	if err != nil {
		return fmt.Errorf("runtime detection failed: %w", err)
	}

	// Check compose command
	c.ui.Step("Checking Compose Command")
	if err := c.CheckComposeCommand(runtime); err != nil {
		c.ui.Warning("Compose command not available")
		c.ui.Info("You can continue, but you'll need to install compose tools later")

		continueAnyway, err := c.ui.PromptYesNo("Continue without compose command?", true)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("setup cancelled - compose command required")
		}
	}

	// Validate rootless configuration
	c.ui.Step("Validating Rootless Configuration")
	if err := c.ValidateRootless(runtime, homelabUser); err != nil {
		c.ui.Warning(fmt.Sprintf("Rootless validation failed: %v", err))
		// Non-critical, continue
	}

	// Test container operations
	c.ui.Step("Testing Container Operations")
	if err := c.TestContainerOperations(runtime); err != nil {
		c.ui.Warning(fmt.Sprintf("Container operations test failed: %v", err))
		// Non-critical, continue
	}

	// Optional: Test image pull
	c.ui.Step("Image Pull Test")
	if err := c.TestImagePull(runtime); err != nil {
		c.ui.Warning(fmt.Sprintf("Image pull test failed: %v", err))
		// Non-critical, continue
	}

	// Display recommendations
	c.DisplayRecommendations(runtime)

	// Save configuration
	c.ui.Step("Saving Configuration")
	if err := c.config.Set("CONTAINER_RUNTIME", string(runtime)); err != nil {
		return fmt.Errorf("failed to save container runtime: %w", err)
	}

	c.ui.Print("")
	c.ui.Separator()
	c.ui.Success("✓ Container runtime setup completed")
	c.ui.Infof("Runtime: %s", runtime)

	// Create completion marker
	if err := c.markers.Create("containers-configured"); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
