package steps

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// PreflightChecker performs system validation checks before setup begins
type PreflightChecker struct {
	packages *system.PackageManager
	network  *system.Network
	ui       *ui.UI
	markers  *config.Markers
	config   *config.Config
}

// NewPreflightChecker creates a new PreflightChecker instance
func NewPreflightChecker(packages *system.PackageManager, network *system.Network, ui *ui.UI, markers *config.Markers, cfg *config.Config) *PreflightChecker {
	return &PreflightChecker{
		packages: packages,
		network:  network,
		ui:       ui,
		markers:  markers,
		config:   cfg,
	}
}

// CheckRpmOstree verifies the system is running rpm-ostree
func (p *PreflightChecker) CheckRpmOstree() error {
	p.ui.Info("Checking for rpm-ostree system...")

	if !system.IsRpmOstreeSystem() {
		p.ui.Error("This system does not appear to be running rpm-ostree")
		p.ui.Info("These setup scripts are designed for UBlue uCore (rpm-ostree based)")
		p.ui.Info("Please use the appropriate setup scripts for your system")
		return fmt.Errorf("not an rpm-ostree system")
	}

	p.ui.Success("Confirmed: Running on rpm-ostree system")

	// Get and display rpm-ostree status
	status, err := system.GetRpmOstreeStatus()
	if err != nil {
		p.ui.Warning(fmt.Sprintf("Could not get detailed rpm-ostree status: %v", err))
		return nil
	}

	// Just log that we got the status (parsing JSON would require encoding/json)
	if len(status) > 0 {
		p.ui.Info("Successfully retrieved rpm-ostree deployment information")
	}

	return nil
}

// CheckRequiredPackages verifies all required packages are installed
func (p *PreflightChecker) CheckRequiredPackages() error {
	p.ui.Info("Checking packages...")

	// Core packages that are always needed
	corePackages := []string{}

	// Optional packages (for optional setup steps)
	optionalPackages := []string{
		"nfs-utils",       // Optional: for NFS setup
		"wireguard-tools", // Optional: for WireGuard VPN setup
	}

	// Check core packages (none currently required)
	if len(corePackages) > 0 {
		results, err := p.packages.CheckMultiple(corePackages)
		if err != nil {
			return fmt.Errorf("failed to check packages: %w", err)
		}

		missingPackages := []string{}
		for _, pkg := range corePackages {
			if results[pkg] {
				p.ui.Successf("  ✓ %s is installed", pkg)
			} else {
				p.ui.Errorf("  ✗ %s is NOT installed", pkg)
				missingPackages = append(missingPackages, pkg)
			}
		}

		if len(missingPackages) > 0 {
			p.ui.Error("Missing required packages")
			p.ui.Info("To install them, run:")
			for _, pkg := range missingPackages {
				p.ui.Infof("  sudo rpm-ostree install %s", pkg)
			}
			p.ui.Info("Then reboot the system:")
			p.ui.Info("  sudo systemctl reboot")
			return fmt.Errorf("missing required packages: %v", missingPackages)
		}
	}

	// Check optional packages (warnings only)
	if len(optionalPackages) > 0 {
		p.ui.Info("Checking optional packages...")
		results, err := p.packages.CheckMultiple(optionalPackages)
		if err != nil {
			p.ui.Warning(fmt.Sprintf("Failed to check optional packages: %v", err))
		} else {
			missingOptional := []string{}
			for _, pkg := range optionalPackages {
				if results[pkg] {
					p.ui.Successf("  ✓ %s is installed", pkg)
				} else {
					p.ui.Infof("  - %s is not installed (optional)", pkg)
					missingOptional = append(missingOptional, pkg)
				}
			}

			if len(missingOptional) > 0 {
				p.ui.Info("Optional packages can be installed later if needed:")
				for _, pkg := range missingOptional {
					p.ui.Infof("  sudo rpm-ostree install %s", pkg)
				}
			}
		}
	}

	p.ui.Success("Package check completed")
	return nil
}

// CheckContainerRuntime verifies a container runtime is available
func (p *PreflightChecker) CheckContainerRuntime() error {
	p.ui.Info("Checking container runtime...")

	// Check for podman first (preferred)
	if system.CommandExists("podman") {
		p.ui.Success("  ✓ Podman is available")

		// Check for podman-compose
		if system.CommandExists("podman-compose") {
			p.ui.Success("  ✓ podman-compose is available")
		} else {
			p.ui.Warning("  podman-compose not found (can be installed later)")
		}
		return nil
	}

	// Check for docker as fallback
	if system.CommandExists("docker") {
		p.ui.Success("  ✓ Docker is available")

		// Check for docker-compose
		if system.CommandExists("docker-compose") {
			p.ui.Success("  ✓ docker-compose is available")
		} else {
			p.ui.Warning("  docker-compose not found (can be installed later)")
		}
		return nil
	}

	p.ui.Error("No container runtime found (podman or docker required)")
	p.ui.Info("To install podman:")
	p.ui.Info("  sudo rpm-ostree install podman podman-compose")
	p.ui.Info("  sudo systemctl reboot")
	return fmt.Errorf("no container runtime available")
}

// CheckNetworkConnectivity tests basic network connectivity
func (p *PreflightChecker) CheckNetworkConnectivity() error {
	p.ui.Info("Checking network connectivity...")

	// Test connectivity to a reliable host
	reachable, err := p.network.TestConnectivity("8.8.8.8", 3)
	if err != nil {
		return fmt.Errorf("failed to test connectivity: %w", err)
	}

	if !reachable {
		p.ui.Error("No internet connectivity detected")
		p.ui.Info("Please check:")
		p.ui.Info("  1. Network cable is connected")
		p.ui.Info("  2. Network configuration is correct")
		p.ui.Info("  3. Default gateway is reachable")
		return fmt.Errorf("no internet connectivity")
	}

	p.ui.Success("Internet connectivity confirmed")

	// Get and display default gateway
	gateway, err := p.network.GetDefaultGateway()
	if err != nil {
		p.ui.Warning(fmt.Sprintf("Could not determine default gateway: %v", err))
	} else {
		p.ui.Infof("Default gateway: %s", gateway)

		// Test gateway connectivity
		gwReachable, _ := p.network.TestConnectivity(gateway, 2)
		if gwReachable {
			p.ui.Success("Default gateway is reachable")
		} else {
			p.ui.Warning("Default gateway is not responding to ping")
		}
	}

	return nil
}

// CheckNFSServer validates NFS server is accessible if configured
func (p *PreflightChecker) CheckNFSServer(host string) error {
	if host == "" {
		p.ui.Info("NFS server not configured yet, skipping NFS check")
		return nil
	}

	p.ui.Infof("Checking NFS server: %s", host)

	// First check basic connectivity
	reachable, err := p.network.TestConnectivity(host, 5)
	if err != nil {
		return fmt.Errorf("failed to test NFS server connectivity: %w", err)
	}

	if !reachable {
		p.ui.Error(fmt.Sprintf("NFS server %s is not reachable", host))
		p.ui.Info("Please check:")
		p.ui.Info("  1. NFS server is powered on")
		p.ui.Info("  2. Network connectivity to the server")
		p.ui.Info("  3. Firewall rules allow NFS traffic")
		return fmt.Errorf("NFS server %s is unreachable", host)
	}

	p.ui.Success(fmt.Sprintf("NFS server %s is reachable", host))

	// Check if NFS exports are available
	hasExports, err := p.network.CheckNFSServer(host)
	if err != nil {
		return fmt.Errorf("failed to check NFS exports: %w", err)
	}

	if !hasExports {
		p.ui.Warning("NFS server is reachable but showmount failed")
		p.ui.Info("This might indicate:")
		p.ui.Info("  1. NFS service is not running on the server")
		p.ui.Info("  2. No exports are configured")
		p.ui.Info("  3. Firewall is blocking NFS RPC calls")
		return fmt.Errorf("NFS server has no accessible exports")
	}

	p.ui.Success("NFS server has accessible exports")

	// Try to get and display exports
	exports, err := p.network.GetNFSExports(host)
	if err == nil && exports != "" {
		p.ui.Info("Available NFS exports:")
		p.ui.Print(exports)
	}

	return nil
}

// RunAll executes all preflight checks
func (p *PreflightChecker) RunAll() error {
	// Check if already completed
	exists, err := p.markers.Exists("preflight-complete")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if exists {
		p.ui.Info("Preflight checks already completed (marker found)")
		p.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/preflight-complete")
		return nil
	}

	p.ui.Header("Pre-flight System Validation")
	p.ui.Info("Verifying system requirements before setup...")
	p.ui.Print("")

	hasErrors := false
	errorMessages := []string{}

	// Run rpm-ostree check
	p.ui.Step("Checking Operating System")
	if err := p.CheckRpmOstree(); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run package checks
	p.ui.Step("Checking Required Packages")
	if err := p.CheckRequiredPackages(); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run container runtime check
	p.ui.Step("Checking Container Runtime")
	if err := p.CheckContainerRuntime(); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run network connectivity check
	p.ui.Step("Checking Network Connectivity")
	if err := p.CheckNetworkConnectivity(); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Check NFS server if configured
	nfsServer := p.config.GetOrDefault("NFS_SERVER", "")
	if nfsServer != "" {
		p.ui.Step("Checking NFS Server")
		if err := p.CheckNFSServer(nfsServer); err != nil {
			// NFS errors are warnings, not critical errors
			p.ui.Warning(err.Error())
		}
	}

	p.ui.Print("")
	p.ui.Separator()

	if hasErrors {
		p.ui.Error("Pre-flight checks FAILED")
		p.ui.Info("Please resolve the issues above before continuing")
		p.ui.Print("")
		for i, msg := range errorMessages {
			p.ui.Errorf("%d. %s", i+1, msg)
		}
		return fmt.Errorf("preflight checks failed with %d error(s)", len(errorMessages))
	}

	p.ui.Success("✓ All pre-flight checks PASSED")
	p.ui.Info("System is ready for homelab setup")

	// Create completion marker
	if err := p.markers.Create("preflight-complete"); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
