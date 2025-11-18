package steps

import (
	"fmt"
	"os"
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

	return &ServiceInfo{
		Name:        serviceName,
		DisplayName: caser.String(serviceName),
		Directory:   filepath.Join(getServiceBaseDir(cfg), serviceName),
		UnitName:    fmt.Sprintf("podman-compose-%s.service", serviceName),
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
	runtimeStr := cfg.GetOrDefault("CONTAINER_RUNTIME", "podman")
	switch runtimeStr {
	case "podman":
		return system.RuntimePodman, nil
	case "docker":
		return system.RuntimeDocker, nil
	default:
		return system.RuntimeNone, fmt.Errorf("unsupported container runtime: %s", runtimeStr)
	}
}

// createComposeService creates a systemd service for docker-compose/podman-compose
func createComposeService(cfg *config.Config, ui *ui.UI, serviceInfo *ServiceInfo) error {
	ui.Infof("Creating systemd service: %s", serviceInfo.UnitName)

	serviceUser, err := getServiceUser(cfg)
	if err != nil {
		return err
	}

	// Get container runtime using helper
	runtime, err := getRuntimeFromConfig(cfg)
	if err != nil {
		return err
	}

	composeCmd, err := system.GetComposeCommand(runtime)
	if err != nil {
		return fmt.Errorf("failed to get compose command: %w", err)
	}

	ui.Infof("Using compose command: %s", composeCmd)

	// Create service unit content
	unitContent := fmt.Sprintf(`[Unit]
Description=Homelab %s Stack
Wants=network-online.target
After=network-online.target
RequiresMountsFor=%s

[Service]
User=%s
Group=%s
Type=oneshot
RemainAfterExit=true
WorkingDirectory=%s
ExecStartPre=%s pull
ExecStart=%s up -d
ExecStop=%s down
TimeoutStartSec=600

[Install]
WantedBy=multi-user.target
`, serviceInfo.DisplayName, serviceInfo.Directory,
		serviceUser, serviceUser,
		serviceInfo.Directory,
		composeCmd, composeCmd, composeCmd)

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

	composeCmd, err := system.GetComposeCommand(runtime)
	if err != nil {
		return fmt.Errorf("failed to get compose command: %w", err)
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

	runtimeStr := cfg.GetOrDefault("CONTAINER_RUNTIME", "podman")

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

	ui.Info("Note: Services may take a few minutes to fully start")
	ui.Info("Check container logs with: podman logs <container-name>")
	ui.Info("Or use: podman ps to see running containers")
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
