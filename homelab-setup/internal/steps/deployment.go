package steps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

const deploymentCompletionMarker = "service-deployment-complete"

// getServiceBaseDir resolves the base directory for service deployments.
// Uses CONTAINERS_BASE which should point to /srv/containers
func getServiceBaseDir(cfg *config.Config) string {
	return cfg.GetOrDefault("CONTAINERS_BASE", "/srv/containers")
}

// ServiceInfo holds information about a service
type ServiceInfo struct {
	Name        string
	DisplayName string
	Directory   string
	UnitName    string
}

// getServiceInfo returns information about a service
func getServiceInfo(cfg *config.Config, serviceName string) *ServiceInfo {
	// Use cases.Title instead of deprecated strings.Title
	caser := cases.Title(language.English)

	// Determine unit name prefix based on runtime
	runtimeStr := cfg.GetOrDefault(config.KeyContainerRuntime, "docker")
	unitPrefix := "docker-compose"
	if runtimeStr == "podman" {
		unitPrefix = "podman-compose"
	}

	return &ServiceInfo{
		Name:        serviceName,
		DisplayName: caser.String(serviceName),
		Directory:   filepath.Join(getServiceBaseDir(cfg), serviceName),
		UnitName:    fmt.Sprintf("%s-%s.service", unitPrefix, serviceName),
	}
}

// getSelectedServices returns the list of selected services from config
func getSelectedServices(cfg *config.Config) ([]string, error) {
	selectedStr := cfg.GetOrDefault("SELECTED_SERVICES", "")
	if selectedStr == "" {
		return nil, fmt.Errorf("no services selected (run container setup first)")
	}

	services := strings.Fields(selectedStr)
	return services, nil
}

// checkExistingService checks if a systemd service exists
func checkExistingService(cfg *config.Config, ui *ui.UI, serviceInfo *ServiceInfo) (bool, error) {
	ui.Infof("Checking for service: %s", serviceInfo.UnitName)

	exists, err := system.ServiceExists(serviceInfo.UnitName)
	if err != nil {
		return false, fmt.Errorf("failed to check service: %w", err)
	}

	if exists {
		ui.Successf("Found pre-configured service: %s", serviceInfo.UnitName)
		return true, nil
	}

	ui.Info("Service not found (will be created)")
	return false, nil
}

// getRuntimeFromConfig is a helper to get container runtime from config
func getRuntimeFromConfig(cfg *config.Config) (system.ContainerRuntime, error) {
	runtimeStr := cfg.GetOrDefault("CONTAINER_RUNTIME", "docker")
	switch runtimeStr {
	case "podman":
		return system.RuntimePodman, nil
	case "docker":
		return system.RuntimeDocker, nil
	default:
		return system.RuntimeNone, fmt.Errorf("unsupported container runtime: %s", runtimeStr)
	}
}

// fstabMountToSystemdUnit converts a fstab mount point to a systemd unit name using systemd-escape
func fstabMountToSystemdUnit(mountPoint string) (string, error) {
	// Use systemd-escape to properly escape the path for systemd unit name
	cmd := exec.Command("systemd-escape", "-p", "--suffix=mount", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to escape mount point %s: %w", mountPoint, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// detectComposeCommand detects and stores the appropriate compose command for the runtime
func detectComposeCommand(cfg *config.Config, runtime system.ContainerRuntime) (string, error) {
	// Check if already detected and stored
	if stored := cfg.GetOrDefault(config.KeyComposeCommand, ""); stored != "" {
		return stored, nil
	}

	// Detect compose command based on runtime
	var composeCmd string
	var err error

	if runtime == system.RuntimeDocker {
		// For Docker, prefer "docker compose" (V2 plugin), fallback to "docker-compose" (V1)
		cmd := exec.Command("docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			composeCmd = "docker compose"
		} else if system.CommandExists("docker-compose") {
			composeCmd = "docker-compose"
		} else {
			return "", fmt.Errorf("neither 'docker compose' (V2) nor 'docker-compose' (V1) found")
		}
	} else {
		// For Podman, use existing detection
		composeCmd, err = system.GetComposeCommand(runtime)
		if err != nil {
			return "", err
		}
	}

	// Store detected command in config for consistency
	if err := cfg.Set(config.KeyComposeCommand, composeCmd); err != nil {
		// Non-fatal error, log warning but continue
		return composeCmd, nil
	}

	return composeCmd, nil
}

// formatComposeCommandForSystemd formats a compose command for use in systemd Exec directives
// Multi-word commands like "docker compose" are formatted with absolute path
// Single commands like "docker-compose" are returned as-is
func formatComposeCommandForSystemd(composeCmd string) string {
	cmdParts := strings.Fields(composeCmd)
	if len(cmdParts) == 2 {
		// Multi-word command like "docker compose" - use absolute path for first part
		return fmt.Sprintf("/usr/bin/%s %s", cmdParts[0], cmdParts[1])
	}
	// Single command like "docker-compose" - use as-is
	return composeCmd
}

// createComposeService creates a systemd service for docker-compose/podman-compose
// For Docker runtime, creates system-level units that depend on docker.service and NFS mounts
// For Podman runtime, maintains rootless behavior with User= directive
func createComposeService(cfg *config.Config, ui *ui.UI, serviceInfo *ServiceInfo) error {
	ui.Infof("Creating systemd service: %s", serviceInfo.UnitName)

	// Get container runtime using helper
	runtime, err := getRuntimeFromConfig(cfg)
	if err != nil {
		return err
	}

	// Detect and store compose command
	composeCmd, err := detectComposeCommand(cfg, runtime)
	if err != nil {
		return fmt.Errorf("failed to detect compose command: %w", err)
	}

	ui.Infof("Using compose command: %s", composeCmd)

	// Build unit dependencies
	var unitAfter, unitRequires, unitWants []string
	var mountDependencies string

	if runtime == system.RuntimeDocker {
		// Docker runtime: system-level service with docker.service dependency
		unitWants = append(unitWants, "network-online.target", "docker.service")
		unitAfter = append(unitAfter, "network-online.target", "docker.service")

		// Check if NFS is configured and add mount dependency
		nfsMountPoint := cfg.GetOrDefault(config.KeyNFSMountPoint, "")
		if nfsMountPoint != "" {
			// Get escaped mount unit name
			mountUnit, err := fstabMountToSystemdUnit(nfsMountPoint)
			if err != nil {
				ui.Warning(fmt.Sprintf("Failed to escape NFS mount point: %v", err))
				ui.Info("NFS mount dependency will not be added to service unit")
			} else {
				unitAfter = append(unitAfter, mountUnit)
				unitRequires = append(unitRequires, mountUnit)
				mountDependencies = fmt.Sprintf("RequiresMountsFor=%s\n", nfsMountPoint)
				ui.Infof("Service will depend on NFS mount: %s (%s)", nfsMountPoint, mountUnit)
			}
		}
	} else {
		// Podman runtime: rootless with User= directive (legacy behavior)
		unitWants = append(unitWants, "network-online.target")
		unitAfter = append(unitAfter, "network-online.target")
	}

	// Build [Unit] section with optional Requires= directive
	var requiresDirective string
	if len(unitRequires) > 0 {
		requiresDirective = fmt.Sprintf("Requires=%s\n", strings.Join(unitRequires, " "))
	}

	unitSection := fmt.Sprintf(`[Unit]
Description=Homelab %s Stack
Wants=%s
After=%s
%s%sRequiresMountsFor=%s

`, serviceInfo.DisplayName,
		strings.Join(unitWants, " "),
		strings.Join(unitAfter, " "),
		requiresDirective,
		mountDependencies,
		serviceInfo.Directory)

	// Build [Service] section
	var serviceSection string
	if runtime == system.RuntimeDocker {
		// Docker: system-level service (no User= directive)
		// Format compose command for systemd Exec directives
		execComposeCmd := formatComposeCommandForSystemd(composeCmd)

		// Add ExecStartPre to verify NFS mount if configured
		var preExecChecks string
		nfsMountPoint := cfg.GetOrDefault(config.KeyNFSMountPoint, "")
		if nfsMountPoint != "" {
			preExecChecks = fmt.Sprintf("ExecStartPre=/usr/bin/findmnt %s\n", nfsMountPoint)
		}
		preExecChecks += fmt.Sprintf("ExecStartPre=%s pull --quiet\n", execComposeCmd)

		serviceSection = fmt.Sprintf(`[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=%s
%sExecStart=%s up -d --remove-orphans
ExecStop=%s down --timeout 30
Restart=on-failure
RestartSec=10
TimeoutStartSec=600
TimeoutStopSec=120

`, serviceInfo.Directory, preExecChecks, execComposeCmd, execComposeCmd)
	} else {
		// Podman: rootless service with User= directive (legacy)
		serviceUser, err := getServiceUser(cfg)
		if err != nil {
			return err
		}

		lingerEnabled, err := system.IsLingerEnabled(serviceUser)
		if err != nil {
			return fmt.Errorf("failed to check lingering for %s: %w", serviceUser, err)
		}

		if !lingerEnabled {
			ui.Infof("Enabling lingering for %s so /run/user is available for rootless compose", serviceUser)
			if err := system.EnableLinger(serviceUser); err != nil {
				return fmt.Errorf("failed to enable lingering for %s: %w", serviceUser, err)
			}
			ui.Successf("Enabled lingering for %s", serviceUser)
		}

		runtimeDir, err := system.EnsureUserRuntimeDir(serviceUser)
		if err != nil {
			return fmt.Errorf("failed to prepare runtime directory for %s: %w", serviceUser, err)
		}

		// Format compose command for systemd Exec directives
		execComposeCmd := formatComposeCommandForSystemd(composeCmd)

		serviceSection = fmt.Sprintf(`[Service]
User=%s
Group=%s
Environment="XDG_RUNTIME_DIR=%s"
Type=oneshot
RemainAfterExit=true
WorkingDirectory=%s
ExecStartPre=%s pull
ExecStart=%s up -d
ExecStop=%s down
TimeoutStartSec=600

`, serviceUser, serviceUser, runtimeDir, serviceInfo.Directory, execComposeCmd, execComposeCmd, execComposeCmd)
	}

	// Build complete unit content
	unitContent := unitSection + serviceSection + `[Install]
WantedBy=multi-user.target
`

	// Write service file
	unitPath := filepath.Join("/etc/systemd/system", serviceInfo.UnitName)
	if err := system.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	ui.Successf("Created service unit: %s", unitPath)

	// Reload systemd daemon
	ui.Info("Reloading systemd daemon...")
	if err := system.SystemdDaemonReload(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to reload daemon: %v", err))
		// Non-critical, continue
	}

	return nil
}

// pullImages pulls container images for a service
func pullImages(cfg *config.Config, ui *ui.UI, serviceInfo *ServiceInfo) error {
	ui.Step(fmt.Sprintf("Pulling Container Images for %s", serviceInfo.DisplayName))

	// Check if compose file exists
	composeFile := filepath.Join(serviceInfo.Directory, "compose.yml")
	dockerComposeFile := filepath.Join(serviceInfo.Directory, "docker-compose.yml")

	composeExists, _ := system.FileExists(composeFile)
	dockerComposeExists, _ := system.FileExists(dockerComposeFile)

	if !composeExists && !dockerComposeExists {
		return fmt.Errorf("no compose file found in %s", serviceInfo.Directory)
	}

	ui.Info("This may take several minutes depending on your internet connection...")

	// Get container runtime using helper
	runtime, err := getRuntimeFromConfig(cfg)
	if err != nil {
		return err
	}

	// Use detected compose command from config
	composeCmd, err := detectComposeCommand(cfg, runtime)
	if err != nil {
		return fmt.Errorf("failed to detect compose command: %w", err)
	}

	// Change to service directory and pull
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := os.Chdir(serviceInfo.Directory); err != nil {
		return fmt.Errorf("failed to change to service directory: %w", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			ui.Warning(fmt.Sprintf("Failed to restore working directory: %v", err))
		}
	}()

	// Execute compose pull
	ui.Infof("Running: %s pull", composeCmd)

	// For compatibility, we need to handle both "podman-compose" and "podman compose" formats
	cmdParts := strings.Fields(composeCmd)
	if len(cmdParts) == 0 {
		return fmt.Errorf("compose command is empty")
	}
	cmdParts = append(cmdParts, "pull")

	if err := system.RunSystemCommand(cmdParts[0], cmdParts[1:]...); err != nil {
		ui.Error(fmt.Sprintf("Failed to pull images: %v", err))
		ui.Info("You may need to pull images manually later")
		return nil // Non-critical error, continue
	}

	ui.Success("Images pulled successfully")
	return nil
}

// enableAndStartService enables and starts a systemd service
func enableAndStartService(cfg *config.Config, ui *ui.UI, serviceInfo *ServiceInfo) error {
	ui.Step(fmt.Sprintf("Enabling and Starting %s Service", serviceInfo.DisplayName))

	// Enable service
	ui.Infof("Enabling service: %s", serviceInfo.UnitName)
	if err := system.EnableService(serviceInfo.UnitName); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	ui.Success("Service enabled")

	// Start service
	ui.Infof("Starting service: %s", serviceInfo.UnitName)
	if err := system.StartService(serviceInfo.UnitName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	ui.Success("Service started")

	return nil
}

// verifyContainers verifies that containers are running
func verifyContainers(cfg *config.Config, ui *ui.UI, serviceInfo *ServiceInfo) error {
	ui.Step(fmt.Sprintf("Verifying %s Containers", serviceInfo.DisplayName))

	// Get container runtime using helper
	runtime, err := getRuntimeFromConfig(cfg)
	if err != nil {
		return err
	}

	runtimeStr := cfg.GetOrDefault("CONTAINER_RUNTIME", "docker")

	// List running containers
	containers, err := system.ListRunningContainers(runtime)
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not list containers: %v", err))
		return nil // Non-critical
	}

	if len(containers) == 0 {
		ui.Warning("No containers are running")
		ui.Info("Check service status: systemctl status " + serviceInfo.UnitName)
		return nil
	}

	// Filter containers related to this service
	var serviceContainers []string
	serviceName := serviceInfo.Name
	for _, container := range containers {
		// Container names usually include the service/stack name
		if strings.Contains(strings.ToLower(container), strings.ToLower(serviceName)) {
			serviceContainers = append(serviceContainers, container)
		}
	}

	if len(serviceContainers) > 0 {
		ui.Successf("Found %d running container(s):", len(serviceContainers))
		for _, container := range serviceContainers {
			ui.Printf("  - %s", container)
		}
	} else {
		ui.Warning("No containers found for this service")
		ui.Info("They may still be starting up. Check with: " + runtimeStr + " ps")
	}

	return nil
}

// displayAccessInfo displays service access information
func displayAccessInfo(cfg *config.Config, ui *ui.UI) {
	ui.Print("")
	ui.Info("Service Access Information:")
	ui.Separator()
	ui.Print("")

	// Common service ports
	servicePorts := map[string]map[string]string{
		"media": {
			"Plex":     "32400",
			"Jellyfin": "8096",
			"Tautulli": "8181",
		},
		"web": {
			"Overseerr": "5055",
			"Wizarr":    "5690",
			"Organizr":  "9983",
			"Homepage":  "3000",
		},
		"cloud": {
			"Nextcloud": "8080",
			"Collabora": "9980",
			"Immich":    "2283",
		},
	}

	selectedServices, _ := getSelectedServices(cfg)

	// Use cases.Title instead of deprecated strings.Title
	caser := cases.Title(language.English)

	for _, service := range selectedServices {
		if ports, ok := servicePorts[service]; ok {
			ui.Infof("%s Stack:", caser.String(service))
			for name, port := range ports {
				ui.Printf("  - %s: http://localhost:%s", name, port)
			}
			ui.Print("")
		}
	}

	// Get runtime for displaying correct commands
	runtimeStr := cfg.GetOrDefault(config.KeyContainerRuntime, "docker")

	ui.Info("Note: Services may take a few minutes to fully start")
	ui.Infof("Check container logs with: %s logs <container-name>", runtimeStr)
	ui.Infof("Or use: %s ps to see running containers", runtimeStr)
	ui.Print("")
}

// displayManagementInfo displays service management instructions
func displayManagementInfo(cfg *config.Config, ui *ui.UI) {
	ui.Print("")
	ui.Info("Service Management:")
	ui.Separator()
	ui.Print("")

	selectedServices, _ := getSelectedServices(cfg)

	ui.Info("Start services:")
	for _, service := range selectedServices {
		serviceInfo := getServiceInfo(cfg, service)
		ui.Printf("  sudo systemctl start %s", serviceInfo.UnitName)
	}
	ui.Print("")

	ui.Info("Stop services:")
	for _, service := range selectedServices {
		serviceInfo := getServiceInfo(cfg, service)
		ui.Printf("  sudo systemctl stop %s", serviceInfo.UnitName)
	}
	ui.Print("")

	ui.Info("Check service status:")
	for _, service := range selectedServices {
		serviceInfo := getServiceInfo(cfg, service)
		ui.Printf("  sudo systemctl status %s", serviceInfo.UnitName)
	}
	ui.Print("")

	ui.Info("View service logs:")
	for _, service := range selectedServices {
		serviceInfo := getServiceInfo(cfg, service)
		ui.Printf("  sudo journalctl -u %s -f", serviceInfo.UnitName)
	}
	ui.Print("")
}

// deployService deploys a single service
func deployService(cfg *config.Config, ui *ui.UI, serviceName string) error {
	serviceInfo := getServiceInfo(cfg, serviceName)

	ui.Header(fmt.Sprintf("Deploying %s Stack", serviceInfo.DisplayName))

	// Check for existing service
	exists, err := checkExistingService(cfg, ui, serviceInfo)
	if err != nil {
		ui.Warning(fmt.Sprintf("Failed to check service: %v", err))
	}

	// Create service if it doesn't exist
	if !exists {
		if err := createComposeService(cfg, ui, serviceInfo); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	// Pull images
	if err := pullImages(cfg, ui, serviceInfo); err != nil {
		ui.Warning(fmt.Sprintf("Image pull had issues: %v", err))
		// Continue anyway
	}

	// Enable and start service
	if err := enableAndStartService(cfg, ui, serviceInfo); err != nil {
		return fmt.Errorf("failed to enable/start service: %w", err)
	}

	// Verify containers
	if err := verifyContainers(cfg, ui, serviceInfo); err != nil {
		ui.Warning(fmt.Sprintf("Container verification had issues: %v", err))
		// Continue anyway
	}

	ui.Print("")
	ui.Successf("✓ %s stack deployed successfully", serviceInfo.DisplayName)

	return nil
}

// runDeploymentPreflight performs preflight checks before deployment
func runDeploymentPreflight(cfg *config.Config, ui *ui.UI) error {
	runtime, err := getRuntimeFromConfig(cfg)
	if err != nil {
		return err
	}

	// For Docker runtime, perform strict preflight checks
	if runtime == system.RuntimeDocker {
		ui.Info("Checking Docker service availability...")

		// Check if docker.service is active
		cmd := exec.Command("systemctl", "is-active", "docker.service")
		if err := cmd.Run(); err != nil {
			ui.Error("docker.service is not active")
			ui.Info("Docker must be running for deployment. Start it with:")
			ui.Info("  sudo systemctl start docker.service")
			ui.Info("  sudo systemctl enable docker.service")
			return fmt.Errorf("docker.service is not active - start it before deploying services")
		}
		ui.Success("docker.service is active")

		// Check compose availability and detect command
		ui.Info("Detecting Docker Compose command...")
		composeCmd, err := detectComposeCommand(cfg, runtime)
		if err != nil {
			ui.Error("Docker Compose is not available")
			ui.Info("Install Docker Compose V2 (preferred):")
			ui.Info("  Follow: https://docs.docker.com/compose/install/")
			ui.Info("Or install V1 standalone:")
			ui.Info("  sudo rpm-ostree install docker-compose")
			return fmt.Errorf("docker compose not available: %w", err)
		}
		ui.Successf("Using compose command: %s", composeCmd)

		// Validate compose files for selected services
		ui.Info("Validating compose files...")
		selectedServices, err := getSelectedServices(cfg)
		if err != nil {
			return fmt.Errorf("failed to get selected services: %w", err)
		}

		for _, serviceName := range selectedServices {
			serviceInfo := getServiceInfo(cfg, serviceName)
			composeFile := filepath.Join(serviceInfo.Directory, "compose.yml")
			dockerComposeFile := filepath.Join(serviceInfo.Directory, "docker-compose.yml")

			// Check if compose file exists
			composeExists, _ := system.FileExists(composeFile)
			dockerComposeExists, _ := system.FileExists(dockerComposeFile)

			if !composeExists && !dockerComposeExists {
				return fmt.Errorf("no compose file found in %s", serviceInfo.Directory)
			}

			// Validate compose file syntax
			originalDir, _ := os.Getwd()
			if err := os.Chdir(serviceInfo.Directory); err != nil {
				return fmt.Errorf("failed to change to service directory %s: %w", serviceInfo.Directory, err)
			}

			// Run compose config --quiet to validate
			cmdParts := strings.Fields(composeCmd)
			cmdParts = append(cmdParts, "config", "--quiet")
			cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
			if output, err := cmd.CombinedOutput(); err != nil {
				os.Chdir(originalDir)
				ui.Error(fmt.Sprintf("Compose file validation failed for %s", serviceName))
				ui.Error(fmt.Sprintf("Output: %s", string(output)))
				return fmt.Errorf("invalid compose file in %s: %w", serviceInfo.Directory, err)
			}

			os.Chdir(originalDir)
			ui.Successf("Validated compose file for %s", serviceName)
		}

		// Check NFS mount if configured
		nfsMountPoint := cfg.GetOrDefault(config.KeyNFSMountPoint, "")
		if nfsMountPoint != "" {
			ui.Infof("Verifying NFS mount at %s...", nfsMountPoint)
			cmd := exec.Command("findmnt", nfsMountPoint)
			if err := cmd.Run(); err != nil {
				ui.Error(fmt.Sprintf("NFS mount not available at %s", nfsMountPoint))
				ui.Info("Ensure the NFS mount is configured and accessible:")
				ui.Info("  1. Check /etc/fstab entry")
				ui.Info("  2. Run: sudo mount -a")
				ui.Info("  3. Verify: findmnt " + nfsMountPoint)
				return fmt.Errorf("NFS mount not available at %s - services may fail without media storage", nfsMountPoint)
			}
			ui.Successf("NFS mount verified at %s", nfsMountPoint)
		}
	} else {
		// For Podman, use existing detection
		ui.Info("Detecting Podman compose command...")
		composeCmd, err := detectComposeCommand(cfg, runtime)
		if err != nil {
			return fmt.Errorf("failed to detect compose command: %w", err)
		}
		ui.Successf("Using compose command: %s", composeCmd)
	}

	ui.Success("Preflight checks passed")
	return nil
}

// RunDeployment executes the deployment step
func RunDeployment(cfg *config.Config, ui *ui.UI) error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(cfg, deploymentCompletionMarker, "deployment-complete")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		ui.Info("Service deployment already completed (marker found)")
		ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + deploymentCompletionMarker)
		return nil
	}

	ui.Header("Service Deployment")
	ui.Info("Deploying container services...")
	ui.Print("")

	// Preflight validation
	ui.Step("Preflight Checks")
	if err := runDeploymentPreflight(cfg, ui); err != nil {
		return fmt.Errorf("preflight checks failed: %w", err)
	}

	// Get selected services
	selectedServices, err := getSelectedServices(cfg)
	if err != nil {
		return fmt.Errorf("failed to get selected services: %w", err)
	}

	ui.Infof("Deploying %d service(s): %s", len(selectedServices), strings.Join(selectedServices, ", "))
	ui.Print("")

	// Deploy each service
	for _, serviceName := range selectedServices {
		if err := deployService(cfg, ui, serviceName); err != nil {
			ui.Error(fmt.Sprintf("Failed to deploy %s: %v", serviceName, err))
			ui.Info("Continuing with remaining services...")
			// Continue with other services
		}
	}

	// Display access information
	displayAccessInfo(cfg, ui)

	// Display management information
	displayManagementInfo(cfg, ui)

	ui.Print("")
	ui.Separator()
	ui.Success("✓ Service deployment completed")
	ui.Infof("Deployed %d stack(s)", len(selectedServices))

	// Create completion marker
	if err := cfg.MarkComplete(deploymentCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
