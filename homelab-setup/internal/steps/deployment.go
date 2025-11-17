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

// Deployment handles service deployment
type Deployment struct {
	config  *config.Config
	ui      *ui.UI
	markers *config.Markers
}

// getServiceBaseDir resolves the base directory for service deployments.
// Uses CONTAINERS_BASE which should point to /srv/containers
func (d *Deployment) getServiceBaseDir() string {
	return d.config.GetOrDefault("CONTAINERS_BASE", "/srv/containers")
}

// ServiceInfo holds information about a service
type ServiceInfo struct {
	Name        string
	DisplayName string
	Directory   string
	UnitName    string
}

// NewDeployment creates a new Deployment instance
func NewDeployment(cfg *config.Config, ui *ui.UI, markers *config.Markers) *Deployment {
	return &Deployment{
		config:  cfg,
		ui:      ui,
		markers: markers,
	}
}

// GetSelectedServices returns the list of selected services from config
func (d *Deployment) GetSelectedServices() ([]string, error) {
	selectedStr := d.config.GetOrDefault("SELECTED_SERVICES", "")
	if selectedStr == "" {
		return nil, fmt.Errorf("no services selected (run container setup first)")
	}

	services := strings.Fields(selectedStr)
	return services, nil
}

// GetServiceInfo returns information about a service
func (d *Deployment) GetServiceInfo(serviceName string) *ServiceInfo {
	// Use cases.Title instead of deprecated strings.Title
	caser := cases.Title(language.English)

	return &ServiceInfo{
		Name:        serviceName,
		DisplayName: caser.String(serviceName),
		Directory:   filepath.Join(d.getServiceBaseDir(), serviceName),
		UnitName:    fmt.Sprintf("podman-compose-%s.service", serviceName),
	}
}

// CheckExistingService checks if a systemd service exists
func (d *Deployment) CheckExistingService(serviceInfo *ServiceInfo) (bool, error) {
	d.ui.Infof("Checking for service: %s", serviceInfo.UnitName)

	exists, err := system.ServiceExists(serviceInfo.UnitName)
	if err != nil {
		return false, fmt.Errorf("failed to check service: %w", err)
	}

	if exists {
		d.ui.Successf("Found pre-configured service: %s", serviceInfo.UnitName)
		return true, nil
	}

	d.ui.Info("Service not found (will be created)")
	return false, nil
}

// getRuntimeFromConfig is a helper to get container runtime from config
func (d *Deployment) getRuntimeFromConfig() (system.ContainerRuntime, error) {
	runtimeStr := d.config.GetOrDefault("CONTAINER_RUNTIME", "podman")
	switch runtimeStr {
	case "podman":
		return system.RuntimePodman, nil
	case "docker":
		return system.RuntimeDocker, nil
	default:
		return system.RuntimeNone, fmt.Errorf("unsupported container runtime: %s", runtimeStr)
	}
}

// CreateComposeService creates a systemd service for docker-compose/podman-compose
func (d *Deployment) CreateComposeService(serviceInfo *ServiceInfo) error {
	d.ui.Infof("Creating systemd service: %s", serviceInfo.UnitName)

	// Get container runtime using helper
	runtime, err := d.getRuntimeFromConfig()
	if err != nil {
		return err
	}

	composeCmd, err := system.GetComposeCommand(runtime)
	if err != nil {
		return fmt.Errorf("failed to get compose command: %w", err)
	}

	d.ui.Infof("Using compose command: %s", composeCmd)

	// Create service unit content
	unitContent := fmt.Sprintf(`[Unit]
Description=Homelab %s Stack
Wants=network-online.target
After=network-online.target
RequiresMountsFor=%s

[Service]
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
		serviceInfo.Directory,
		composeCmd, composeCmd, composeCmd)

	// Write service file
	unitPath := filepath.Join("/etc/systemd/system", serviceInfo.UnitName)
	if err := system.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	d.ui.Successf("Created service unit: %s", unitPath)

	// Reload systemd daemon
	d.ui.Info("Reloading systemd daemon...")
	if err := system.SystemdDaemonReload(); err != nil {
		d.ui.Warning(fmt.Sprintf("Failed to reload daemon: %v", err))
		// Non-critical, continue
	}

	return nil
}

// PullImages pulls container images for a service
func (d *Deployment) PullImages(serviceInfo *ServiceInfo) error {
	d.ui.Step(fmt.Sprintf("Pulling Container Images for %s", serviceInfo.DisplayName))

	// Check if compose file exists
	composeFile := filepath.Join(serviceInfo.Directory, "compose.yml")
	dockerComposeFile := filepath.Join(serviceInfo.Directory, "docker-compose.yml")

	composeExists, _ := system.FileExists(composeFile)
	dockerComposeExists, _ := system.FileExists(dockerComposeFile)

	if !composeExists && !dockerComposeExists {
		return fmt.Errorf("no compose file found in %s", serviceInfo.Directory)
	}

	d.ui.Info("This may take several minutes depending on your internet connection...")

	// Get container runtime using helper
	runtime, err := d.getRuntimeFromConfig()
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
			d.ui.Warning(fmt.Sprintf("Failed to restore working directory: %v", err))
		}
	}()

	// Execute compose pull
	d.ui.Infof("Running: %s pull", composeCmd)

	// For compatibility, we need to handle both "podman-compose" and "podman compose" formats
	cmdParts := strings.Fields(composeCmd)
	if len(cmdParts) == 0 {
		return fmt.Errorf("compose command is empty")
	}
	cmdParts = append(cmdParts, "pull")

	if err := system.RunSystemCommand(cmdParts[0], cmdParts[1:]...); err != nil {
		d.ui.Error(fmt.Sprintf("Failed to pull images: %v", err))
		d.ui.Info("You may need to pull images manually later")
		return nil // Non-critical error, continue
	}

	d.ui.Success("Images pulled successfully")
	return nil
}

// EnableAndStartService enables and starts a systemd service
func (d *Deployment) EnableAndStartService(serviceInfo *ServiceInfo) error {
	d.ui.Step(fmt.Sprintf("Enabling and Starting %s Service", serviceInfo.DisplayName))

	// Enable service
	d.ui.Infof("Enabling service: %s", serviceInfo.UnitName)
	if err := system.EnableService(serviceInfo.UnitName); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	d.ui.Success("Service enabled")

	// Start service
	d.ui.Infof("Starting service: %s", serviceInfo.UnitName)
	if err := system.StartService(serviceInfo.UnitName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	d.ui.Success("Service started")

	return nil
}

// VerifyContainers verifies that containers are running
func (d *Deployment) VerifyContainers(serviceInfo *ServiceInfo) error {
	d.ui.Step(fmt.Sprintf("Verifying %s Containers", serviceInfo.DisplayName))

	// Get container runtime using helper
	runtime, err := d.getRuntimeFromConfig()
	if err != nil {
		return err
	}

	runtimeStr := d.config.GetOrDefault("CONTAINER_RUNTIME", "podman")

	// List running containers
	containers, err := system.ListRunningContainers(runtime)
	if err != nil {
		d.ui.Warning(fmt.Sprintf("Could not list containers: %v", err))
		return nil // Non-critical
	}

	if len(containers) == 0 {
		d.ui.Warning("No containers are running")
		d.ui.Info("Check service status: systemctl status " + serviceInfo.UnitName)
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
		d.ui.Successf("Found %d running container(s):", len(serviceContainers))
		for _, container := range serviceContainers {
			d.ui.Printf("  - %s", container)
		}
	} else {
		d.ui.Warning("No containers found for this service")
		d.ui.Info("They may still be starting up. Check with: " + runtimeStr + " ps")
	}

	return nil
}

// DisplayAccessInfo displays service access information
func (d *Deployment) DisplayAccessInfo() {
	d.ui.Print("")
	d.ui.Info("Service Access Information:")
	d.ui.Separator()
	d.ui.Print("")

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

	selectedServices, _ := d.GetSelectedServices()

	// Use cases.Title instead of deprecated strings.Title
	caser := cases.Title(language.English)

	for _, service := range selectedServices {
		if ports, ok := servicePorts[service]; ok {
			d.ui.Infof("%s Stack:", caser.String(service))
			for name, port := range ports {
				d.ui.Printf("  - %s: http://localhost:%s", name, port)
			}
			d.ui.Print("")
		}
	}

	d.ui.Info("Note: Services may take a few minutes to fully start")
	d.ui.Info("Check container logs with: podman logs <container-name>")
	d.ui.Info("Or use: podman ps to see running containers")
	d.ui.Print("")
}

// DisplayManagementInfo displays service management instructions
func (d *Deployment) DisplayManagementInfo() {
	d.ui.Print("")
	d.ui.Info("Service Management:")
	d.ui.Separator()
	d.ui.Print("")

	selectedServices, _ := d.GetSelectedServices()

	d.ui.Info("Start services:")
	for _, service := range selectedServices {
		serviceInfo := d.GetServiceInfo(service)
		d.ui.Printf("  sudo systemctl start %s", serviceInfo.UnitName)
	}
	d.ui.Print("")

	d.ui.Info("Stop services:")
	for _, service := range selectedServices {
		serviceInfo := d.GetServiceInfo(service)
		d.ui.Printf("  sudo systemctl stop %s", serviceInfo.UnitName)
	}
	d.ui.Print("")

	d.ui.Info("Check service status:")
	for _, service := range selectedServices {
		serviceInfo := d.GetServiceInfo(service)
		d.ui.Printf("  sudo systemctl status %s", serviceInfo.UnitName)
	}
	d.ui.Print("")

	d.ui.Info("View service logs:")
	for _, service := range selectedServices {
		serviceInfo := d.GetServiceInfo(service)
		d.ui.Printf("  sudo journalctl -u %s -f", serviceInfo.UnitName)
	}
	d.ui.Print("")
}

// DeployService deploys a single service
func (d *Deployment) DeployService(serviceName string) error {
	serviceInfo := d.GetServiceInfo(serviceName)

	d.ui.Header(fmt.Sprintf("Deploying %s Stack", serviceInfo.DisplayName))

	// Check for existing service
	exists, err := d.CheckExistingService(serviceInfo)
	if err != nil {
		d.ui.Warning(fmt.Sprintf("Failed to check service: %v", err))
	}

	// Create service if it doesn't exist
	if !exists {
		if err := d.CreateComposeService(serviceInfo); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	// Pull images
	if err := d.PullImages(serviceInfo); err != nil {
		d.ui.Warning(fmt.Sprintf("Image pull had issues: %v", err))
		// Continue anyway
	}

	// Enable and start service
	if err := d.EnableAndStartService(serviceInfo); err != nil {
		return fmt.Errorf("failed to enable/start service: %w", err)
	}

	// Verify containers
	if err := d.VerifyContainers(serviceInfo); err != nil {
		d.ui.Warning(fmt.Sprintf("Container verification had issues: %v", err))
		// Continue anyway
	}

	d.ui.Print("")
	d.ui.Successf("✓ %s stack deployed successfully", serviceInfo.DisplayName)

	return nil
}

const deploymentCompletionMarker = "service-deployment-complete"

// Run executes the deployment step
func (d *Deployment) Run() error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(d.markers, deploymentCompletionMarker, "deployment-complete")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		d.ui.Info("Service deployment already completed (marker found)")
		d.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + deploymentCompletionMarker)
		return nil
	}

	d.ui.Header("Service Deployment")
	d.ui.Info("Deploying container services...")
	d.ui.Print("")

	// Get selected services
	selectedServices, err := d.GetSelectedServices()
	if err != nil {
		return fmt.Errorf("failed to get selected services: %w", err)
	}

	d.ui.Infof("Deploying %d service(s): %s", len(selectedServices), strings.Join(selectedServices, ", "))
	d.ui.Print("")

	// Deploy each service
	for _, serviceName := range selectedServices {
		if err := d.DeployService(serviceName); err != nil {
			d.ui.Error(fmt.Sprintf("Failed to deploy %s: %v", serviceName, err))
			d.ui.Info("Continuing with remaining services...")
			// Continue with other services
		}
	}

	// Display access information
	d.DisplayAccessInfo()

	// Display management information
	d.DisplayManagementInfo()

	d.ui.Print("")
	d.ui.Separator()
	d.ui.Success("✓ Service deployment completed")
	d.ui.Infof("Deployed %d stack(s)", len(selectedServices))

	// Create completion marker
	if err := d.markers.Create(deploymentCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
