package steps

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// PackageChecker defines the interface for checking package installation status
type PackageChecker interface {
	IsInstalled(packageName string) (bool, error)
}

// NFSConfigurator handles NFS mount configuration
type NFSConfigurator struct {
	fs       system.FileSystemManager
	network  *system.Network
	config   *config.Config
	ui       *ui.UI
	markers  *config.Markers
	runner   system.CommandRunner
	packages PackageChecker
}

// NewNFSConfigurator creates a new NFSConfigurator instance
func NewNFSConfigurator(fs system.FileSystemManager, network *system.Network, cfg *config.Config, ui *ui.UI, markers *config.Markers, packages PackageChecker) *NFSConfigurator {
	return &NFSConfigurator{
		fs:       fs,
		network:  network,
		config:   cfg,
		ui:       ui,
		markers:  markers,
		runner:   system.NewCommandRunner(),
		packages: packages,
	}
}

// CheckNFSUtils verifies that nfs-utils package is installed
func (n *NFSConfigurator) CheckNFSUtils() error {
	n.ui.Info("Checking for NFS client utilities...")

	installed, err := n.packages.IsInstalled("nfs-utils")
	if err != nil {
		n.ui.Warning(fmt.Sprintf("Could not verify nfs-utils package: %v", err))
		n.ui.Info("Proceeding anyway - mount may fail if package is not installed")
		return nil
	}

	if !installed {
		n.ui.Error("nfs-utils package is not installed")
		n.ui.Info("NFS client utilities are required for mounting NFS shares")
		n.ui.Print("")
		n.ui.Info("To install nfs-utils:")
		n.ui.Info("  1. Install the package:")
		n.ui.Info("     sudo rpm-ostree install nfs-utils")
		n.ui.Info("  2. Reboot the system:")
		n.ui.Info("     sudo systemctl reboot")
		n.ui.Info("  3. Re-run the setup after reboot")
		n.ui.Print("")
		return fmt.Errorf("nfs-utils package is not installed")
	}

	n.ui.Success("nfs-utils package is installed")
	return nil
}

// PromptForNFS asks if the user wants to configure NFS
func (n *NFSConfigurator) PromptForNFS() (bool, error) {
	n.ui.Info("NFS (Network File System) allows you to mount remote storage")
	n.ui.Info("This is useful for accessing media libraries from a NAS server")
	n.ui.Print("")

	useNFS, err := n.ui.PromptYesNo("Do you want to configure NFS mounts?", true)
	if err != nil {
		return false, fmt.Errorf("failed to prompt for NFS: %w", err)
	}

	return useNFS, nil
}

// PromptForNFSDetails prompts for NFS server and export details
func (n *NFSConfigurator) PromptForNFSDetails() (host, export, mountPoint string, err error) {
	// Check if already configured
	existingServer := n.config.GetOrDefault("NFS_SERVER", "")
	if existingServer != "" {
		n.ui.Infof("Previously configured NFS server: %s", existingServer)
		useExisting, err := n.ui.PromptYesNo("Use this NFS server?", true)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to prompt: %w", err)
		}
		if useExisting {
			host = existingServer
			export = n.config.GetOrDefault("NFS_EXPORT", "")
			mountPoint = n.config.GetOrDefault("NFS_MOUNT_POINT", "")
			if export != "" && mountPoint != "" {
				return host, export, mountPoint, nil
			}
		}
	}

	// Prompt for NFS server
	n.ui.Print("")
	n.ui.Info("Enter NFS server details:")
	host, err = n.ui.PromptInput("NFS server IP or hostname", "192.168.1.100")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for NFS server: %w", err)
	}

	// Validate IP or hostname
	// Try IP validation first
	if err := common.ValidateIP(host); err != nil {
		// Not an IP, try as hostname
		if err := common.ValidateDomain(host); err != nil {
			return "", "", "", fmt.Errorf("invalid NFS server (not a valid IP or hostname): %s", host)
		}
	}

	// Prompt for export path
	export, err = n.ui.PromptInput("NFS export path (e.g., /mnt/storage/media)", "/mnt/storage")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for NFS export: %w", err)
	}

	// Validate export path (use ValidateSafePath to prevent command injection)
	if err := common.ValidateSafePath(export); err != nil {
		return "", "", "", fmt.Errorf("invalid export path: %w", err)
	}

	// Prompt for mount point
	mountPoint, err = n.ui.PromptInput("Local mount point", "/mnt/nas-media")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for mount point: %w", err)
	}

	// Validate mount point (use ValidateSafePath to prevent command injection)
	if err := common.ValidateSafePath(mountPoint); err != nil {
		return "", "", "", fmt.Errorf("invalid mount point: %w", err)
	}

	return host, export, mountPoint, nil
}

// ValidateNFSConnection validates the NFS server is accessible and exports are available
func (n *NFSConfigurator) ValidateNFSConnection(host string) error {
	n.ui.Infof("Testing connection to NFS server %s...", host)

	// Get timeout from config (default 10 seconds)
	timeoutStr := n.config.GetOrDefault(config.KeyNetworkTestTimeout, "10")
	var timeout int
	if _, err := fmt.Sscanf(timeoutStr, "%d", &timeout); err != nil || timeout <= 0 {
		timeout = 10
	}

	// Test basic connectivity with configurable timeout
	reachable, err := n.network.TestConnectivity(host, timeout)
	if err != nil {
		return fmt.Errorf("failed to test connectivity: %w", err)
	}

	if !reachable {
		n.ui.Error(fmt.Sprintf("NFS server %s is not reachable", host))
		n.ui.Info("Please check:")
		n.ui.Info("  1. Server is powered on")
		n.ui.Info("  2. Network configuration is correct")
		n.ui.Info("  3. Firewall allows NFS traffic")
		return fmt.Errorf("NFS server is unreachable")
	}

	n.ui.Success("NFS server is reachable")

	// Check if NFS exports are available
	hasExports, err := n.network.CheckNFSServer(host)
	if err != nil {
		return fmt.Errorf("failed to check NFS exports: %w", err)
	}

	if !hasExports {
		n.ui.Warning("NFS server is reachable but showmount failed")
		n.ui.Info("This might indicate:")
		n.ui.Info("  1. NFS service is not running")
		n.ui.Info("  2. No exports are configured")
		n.ui.Info("  3. Firewall is blocking NFS RPC")

		// Ask if they want to continue anyway
		continueAnyway, err := n.ui.PromptYesNo("Continue with NFS setup anyway?", false)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("NFS setup cancelled")
		}
	} else {
		n.ui.Success("NFS server has accessible exports")

		// Try to display exports
		exports, err := n.network.GetNFSExports(host)
		if err == nil && exports != "" {
			n.ui.Print("")
			n.ui.Info("Available NFS exports:")
			for _, line := range strings.Split(exports, "\n") {
				if strings.TrimSpace(line) != "" {
					n.ui.Printf("  %s", line)
				}
			}
			n.ui.Print("")
		}
	}

	return nil
}

// ValidateNFSExport verifies that the specified export path exists on the NFS server
func (n *NFSConfigurator) ValidateNFSExport(host, export string) error {
	n.ui.Infof("Verifying export path '%s' on server...", export)

	// Get the list of exports from the server
	exports, err := n.network.GetNFSExports(host)
	if err != nil {
		n.ui.Warning(fmt.Sprintf("Could not verify export path: %v", err))
		n.ui.Info("Proceeding without verification - mount will fail if export doesn't exist")
		return nil // Non-critical, let mount attempt reveal the issue
	}

	// Parse exports and check if our export exists
	// showmount output format: "Export list for <host>:"
	// "/export/path client1,client2"
	exportFound := false
	lines := strings.Split(exports, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Export list") {
			continue
		}

		// Extract the export path (first field)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			serverExport := fields[0]
			if serverExport == export {
				exportFound = true
				n.ui.Successf("Export path '%s' exists on server", export)
				break
			}
		}
	}

	if !exportFound {
		n.ui.Warning(fmt.Sprintf("Export path '%s' not found in server's export list", export))
		n.ui.Info("Available exports are listed above")
		n.ui.Info("The mount will likely fail if this path doesn't exist")
		n.ui.Print("")

		// Ask if they want to continue
		continueAnyway, err := n.ui.PromptYesNo("Continue with this export path anyway?", false)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("NFS setup cancelled - export path not verified")
		}
	}

	return nil
}

// CreateMountPoint creates the local mount point directory
func (n *NFSConfigurator) CreateMountPoint(mountPoint string) error {
	n.ui.Infof("Creating mount point %s...", mountPoint)

	// Create directory with root ownership (mount points should be owned by root)
	if err := n.fs.EnsureDirectory(mountPoint, "root:root", 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	n.ui.Success("Mount point created")
	return nil
}

// mountPointToUnitBaseName converts a mount point path to a systemd unit base name.
// Example: "/mnt/nas-media" -> "mnt-nas-media"
func mountPointToUnitBaseName(mountPoint string) string {
	// Trim leading/trailing whitespace first
	cleanedPath := strings.TrimSpace(mountPoint)
	cleanedPath = filepath.Clean(cleanedPath)

	// Strip leading "/" if present
	name := strings.TrimPrefix(cleanedPath, "/")

	// Replace remaining "/" with "-"
	name = strings.ReplaceAll(name, "/", "-")

	// Replace any whitespace with "-" to keep systemd filenames valid
	name = strings.Join(strings.FieldsFunc(name, unicode.IsSpace), "-")

	return name
}

// getNFSMountOptions returns the NFS mount options from config or a default
func (n *NFSConfigurator) getNFSMountOptions() string {
	options := n.config.GetOrDefault(config.KeyNFSMountOptions, "")
	if options == "" {
		return "defaults,_netdev"
	}
	return options
}

// CreateSystemdUnits creates a systemd mount and automount unit for NFS.
func (n *NFSConfigurator) CreateSystemdUnits(host, export, mountPoint string) error {
	n.ui.Info("Creating systemd mount and automount units...")

	// Convert mount point to systemd unit name.
	// Example: /mnt/nas-media -> mnt-nas-media
	unitBaseName := mountPointToUnitBaseName(mountPoint)

	mountUnitName := unitBaseName + ".mount"
	automountUnitName := unitBaseName + ".automount"

	mountUnitPath := filepath.Join("/etc/systemd/system", mountUnitName)
	automountUnitPath := filepath.Join("/etc/systemd/system", automountUnitName)

	n.ui.Infof("Creating units: %s, %s", mountUnitName, automountUnitName)

	// Get NFS mount options from config or use default, add nofail for resilience.
	mountOptions := n.getNFSMountOptions()
	if !strings.Contains(mountOptions, "nofail") {
		mountOptions = "nofail," + mountOptions
	}
	n.ui.Infof("Using NFS mount options: %s", mountOptions)

	// Generate mount unit content.
	mountContent := fmt.Sprintf(`[Unit]
Description=NFS mount for %s
After=network-online.target
Requires=network-online.target

[Mount]
What=%s:%s
Where=%s
Type=nfs
Options=%s
TimeoutSec=30
`, mountPoint, host, export, mountPoint, mountOptions)

	// Generate automount unit content.
	automountContent := fmt.Sprintf(`[Unit]
Description=Automount for %s
After=network-online.target
Requires=network-online.target

[Automount]
Where=%s
TimeoutIdleSec=600

[Install]
WantedBy=multi-user.target
`, mountPoint, mountPoint)

	// Write the mount unit file.
	if err := n.fs.WriteFile(mountUnitPath, []byte(mountContent), 0644); err != nil {
		return fmt.Errorf("failed to write mount unit %s: %w", mountUnitPath, err)
	}
	n.ui.Successf("Created mount unit: %s", mountUnitPath)

	// Write the automount unit file.
	if err := n.fs.WriteFile(automountUnitPath, []byte(automountContent), 0644); err != nil {
		return fmt.Errorf("failed to write automount unit %s: %w", automountUnitPath, err)
	}
	n.ui.Successf("Created automount unit: %s", automountUnitPath)

	// Reload systemd to recognize the new units.
	if output, err := n.runner.Run("sudo", "-n", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w\nOutput: %s", err, output)
	}
	n.ui.Success("systemd reloaded")

	// Enable and start the automount unit.
	if output, err := n.runner.Run("sudo", "-n", "systemctl", "enable", "--now", automountUnitName); err != nil {
		return fmt.Errorf("failed to enable and start automount unit: %w\nOutput: %s", err, output)
	}

	n.ui.Successf("Enabled and started automount unit: %s", automountUnitName)

	return nil
}

const nfsCompletionMarker = "nfs-setup-complete"

// Run executes the NFS configuration step
func (n *NFSConfigurator) Run() error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(n.markers, nfsCompletionMarker, "nfs-configured", "nfs-skipped")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		n.ui.Info("NFS already configured (marker found)")
		n.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + nfsCompletionMarker)
		return nil
	}

	n.ui.Header("NFS Configuration")
	n.ui.Info("Configure Network File System (NFS) mounts...")
	n.ui.Print("")

	// Ask if they want to configure NFS
	n.ui.Step("NFS Setup")
	useNFS, err := n.PromptForNFS()
	if err != nil {
		return fmt.Errorf("failed to prompt for NFS: %w", err)
	}

	if !useNFS {
		n.ui.Info("Skipping NFS configuration")
		n.ui.Info("To configure NFS later, remove marker: ~/.local/homelab-setup/" + nfsCompletionMarker)
		if err := n.markers.Create(nfsCompletionMarker); err != nil {
			return fmt.Errorf("failed to create completion marker: %w", err)
		}
		return nil
	}

	// Check for nfs-utils package
	n.ui.Step("Checking NFS Prerequisites")
	if err := n.CheckNFSUtils(); err != nil {
		return fmt.Errorf("NFS prerequisites check failed: %w", err)
	}

	// Get NFS details
	n.ui.Step("NFS Server Details")
	host, export, mountPoint, err := n.PromptForNFSDetails()
	if err != nil {
		return fmt.Errorf("failed to get NFS details: %w", err)
	}

	// Validate NFS connection
	n.ui.Step("Validating NFS Connection")
	if err := n.ValidateNFSConnection(host); err != nil {
		n.ui.Error(fmt.Sprintf("NFS validation failed: %v", err))

		continueAnyway, err := n.ui.PromptYesNo("Continue with NFS setup despite validation errors?", false)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("NFS setup cancelled due to validation errors")
		}
	}

	// Validate export path exists on server
	n.ui.Step("Verifying Export Path")
	if err := n.ValidateNFSExport(host, export); err != nil {
		return fmt.Errorf("export path verification failed: %w", err)
	}

	// Create mount point
	n.ui.Step("Creating Mount Point")
	if err := n.CreateMountPoint(mountPoint); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Create systemd mount and automount units
	n.ui.Step("Creating Systemd Units")
	if err := n.CreateSystemdUnits(host, export, mountPoint); err != nil {
		return fmt.Errorf("failed to create systemd units: %w", err)
	}

	// The automount unit handles starting, so we don't need to manually start the mount unit.
	// We also don't need to check the status here as it will be mounted on first access.
	n.ui.Info("Systemd automount configured. The share will be mounted on first access.")

	// Save configuration
	n.ui.Step("Saving Configuration")
	if err := n.config.Set("NFS_SERVER", host); err != nil {
		return fmt.Errorf("failed to save NFS server: %w", err)
	}

	if err := n.config.Set("NFS_EXPORT", export); err != nil {
		return fmt.Errorf("failed to save NFS export: %w", err)
	}

	if err := n.config.Set("NFS_MOUNT_POINT", mountPoint); err != nil {
		return fmt.Errorf("failed to save NFS mount point: %w", err)
	}

	n.ui.Print("")
	n.ui.Separator()
	n.ui.Success("✓ NFS configuration completed")
	n.ui.Infof("Server: %s", host)
	n.ui.Infof("Export: %s", export)
	n.ui.Infof("Mount Point: %s", mountPoint)

	// Create completion marker
	if err := n.markers.Create(nfsCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
