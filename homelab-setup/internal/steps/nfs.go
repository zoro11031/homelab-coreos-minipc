package steps

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

const nfsCompletionMarker = "nfs-setup-complete"

// checkNFSUtils verifies that nfs-utils package is installed
func checkNFSUtils(cfg *config.Config, ui *ui.UI) error {
	ui.Info("Checking for NFS client utilities...")

	installed, err := system.IsPackageInstalled("nfs-utils")
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not verify nfs-utils package: %v", err))
		ui.Info("Proceeding anyway - mount may fail if package is not installed")
		return nil
	}

	if !installed {
		ui.Error("nfs-utils package is not installed")
		ui.Info("NFS client utilities are required for mounting NFS shares")
		ui.Print("")
		ui.Info("To install nfs-utils:")
		ui.Info("  1. Install the package:")
		ui.Info("     sudo rpm-ostree install nfs-utils")
		ui.Info("  2. Reboot the system:")
		ui.Info("     sudo systemctl reboot")
		ui.Info("  3. Re-run the setup after reboot")
		ui.Print("")
		return fmt.Errorf("nfs-utils package is not installed")
	}

	ui.Success("nfs-utils package is installed")
	return nil
}

// promptForNFS asks if the user wants to configure NFS
func promptForNFS(cfg *config.Config, ui *ui.UI) (bool, error) {
	ui.Info("NFS (Network File System) allows you to mount remote storage")
	ui.Info("This is useful for accessing media libraries from a NAS server")
	ui.Print("")

	useNFS, err := ui.PromptYesNo("Do you want to configure NFS mounts?", true)
	if err != nil {
		return false, fmt.Errorf("failed to prompt for NFS: %w", err)
	}

	return useNFS, nil
}

// promptForNFSDetails prompts for NFS server and export details
func promptForNFSDetails(cfg *config.Config, ui *ui.UI) (host, export, mountPoint string, err error) {
	// Check if already configured
	existingServer := cfg.GetOrDefault("NFS_SERVER", "")
	if existingServer != "" {
		ui.Infof("Previously configured NFS server: %s", existingServer)
		useExisting, err := ui.PromptYesNo("Use this NFS server?", true)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to prompt: %w", err)
		}
		if useExisting {
			host = existingServer
			export = cfg.GetOrDefault("NFS_EXPORT", "")
			mountPoint = cfg.GetOrDefault("NFS_MOUNT_POINT", "")
			if export != "" && mountPoint != "" {
				return host, export, mountPoint, nil
			}
		}
	}

	// Prompt for NFS server
	ui.Print("")
	ui.Info("Enter NFS server details:")
	host, err = ui.PromptInput("NFS server IP or hostname", "192.168.1.100")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for NFS server: %w", err)
	}

	// Validate IP or hostname
	// Note: IP/hostname validation is intentionally inlined here rather than using a
	// shared validator function. This trades code reuse for simplicity and keeps
	// NFS-specific validation logic self-contained.
	// Validate NFS server - must be valid IPv4 or hostname
	isValidIP := false
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		isValidIP = true
	}
	if !isValidIP {
		// Not an IP, validate as hostname/domain
		// Allow single-label hostnames (e.g., 'nas', 'truenas') for mDNS/NetBIOS names
		if host == "" || len(host) > 253 {
			return "", "", "", fmt.Errorf("invalid NFS server (not a valid IP or hostname): %s", host)
		}
		// Basic hostname validation - check each label
		parts := strings.Split(host, ".")
		for _, part := range parts {
			if part == "" || len(part) > 63 {
				return "", "", "", fmt.Errorf("invalid NFS server hostname: %s", host)
			}
		}
	}

	// Prompt for export path
	export, err = ui.PromptInput("NFS export path (e.g., /mnt/storage/media)", "/mnt/storage")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for NFS export: %w", err)
	}

	// Validate export path (use ValidateSafePath to prevent command injection)
	if err := common.ValidateSafePath(export); err != nil {
		return "", "", "", fmt.Errorf("invalid export path: %w", err)
	}

	// Prompt for mount point
	mountPoint, err = ui.PromptInput("Local mount point", "/mnt/nas-media")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for mount point: %w", err)
	}

	// Validate mount point (use ValidateSafePath to prevent command injection)
	if err := common.ValidateSafePath(mountPoint); err != nil {
		return "", "", "", fmt.Errorf("invalid mount point: %w", err)
	}

	return host, export, mountPoint, nil
}

// validateNFSConnection validates the NFS server is accessible and exports are available
func validateNFSConnection(cfg *config.Config, ui *ui.UI, host string) error {
	ui.Infof("Testing connection to NFS server %s...", host)

	// Get timeout from config (default 10 seconds)
	timeoutStr := cfg.GetOrDefault(config.KeyNetworkTestTimeout, "10")
	var timeout int
	if _, err := fmt.Sscanf(timeoutStr, "%d", &timeout); err != nil || timeout <= 0 {
		timeout = 10
	}

	// Test basic connectivity with configurable timeout
	reachable, err := system.TestConnectivity(host, timeout)
	if err != nil {
		return fmt.Errorf("failed to test connectivity: %w", err)
	}

	if !reachable {
		ui.Error(fmt.Sprintf("NFS server %s is not reachable", host))
		ui.Info("Please check:")
		ui.Info("  1. Server is powered on")
		ui.Info("  2. Network configuration is correct")
		ui.Info("  3. Firewall allows NFS traffic")
		return fmt.Errorf("NFS server is unreachable")
	}

	ui.Success("NFS server is reachable")

	// Check if NFS exports are available
	hasExports, err := system.CheckNFSServer(host)
	if err != nil {
		return fmt.Errorf("failed to check NFS exports: %w", err)
	}

	if !hasExports {
		ui.Warning("NFS server is reachable but showmount failed")
		ui.Info("This might indicate:")
		ui.Info("  1. NFS service is not running")
		ui.Info("  2. No exports are configured")
		ui.Info("  3. Firewall is blocking NFS RPC")

		// Ask if they want to continue anyway
		continueAnyway, err := ui.PromptYesNo("Continue with NFS setup anyway?", false)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("NFS setup cancelled")
		}
	} else {
		ui.Success("NFS server has accessible exports")

		// Try to display exports
		exports, err := system.GetNFSExports(host)
		if err == nil && exports != "" {
			ui.Print("")
			ui.Info("Available NFS exports:")
			for _, line := range strings.Split(exports, "\n") {
				if strings.TrimSpace(line) != "" {
					ui.Printf("  %s", line)
				}
			}
			ui.Print("")
		}
	}

	return nil
}

// validateNFSExport verifies that the specified export path exists on the NFS server
func validateNFSExport(cfg *config.Config, ui *ui.UI, host, export string) error {
	ui.Infof("Verifying export path '%s' on server...", export)

	// Get the list of exports from the server
	exports, err := system.GetNFSExports(host)
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not verify export path: %v", err))
		ui.Info("Proceeding without verification - mount will fail if export doesn't exist")
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
				ui.Successf("Export path '%s' exists on server", export)
				break
			}
		}
	}

	if !exportFound {
		ui.Warning(fmt.Sprintf("Export path '%s' not found in server's export list", export))
		ui.Info("Available exports are listed above")
		ui.Info("The mount will likely fail if this path doesn't exist")
		ui.Print("")

		// Ask if they want to continue
		continueAnyway, err := ui.PromptYesNo("Continue with this export path anyway?", false)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("NFS setup cancelled - export path not verified")
		}
	}

	return nil
}

// createMountPoint creates the local mount point directory
func createMountPoint(cfg *config.Config, ui *ui.UI, mountPoint string) error {
	ui.Infof("Creating mount point %s...", mountPoint)

	// Create directory with root ownership (mount points should be owned by root)
	if err := system.EnsureDirectory(mountPoint, "root:root", 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	ui.Success("Mount point created")
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
func getNFSMountOptions(cfg *config.Config) string {
	options := cfg.GetOrDefault(config.KeyNFSMountOptions, "")
	if options == "" {
		return "defaults,_netdev"
	}
	return options
}

// createSystemdUnits creates a systemd mount and automount unit for NFS.
func createSystemdUnits(cfg *config.Config, ui *ui.UI, host, export, mountPoint string) error {
	ui.Info("Creating systemd mount and automount units...")

	// Convert mount point to systemd unit name.
	// Example: /mnt/nas-media -> mnt-nas-media
	unitBaseName := mountPointToUnitBaseName(mountPoint)

	mountUnitName := unitBaseName + ".mount"
	automountUnitName := unitBaseName + ".automount"

	mountUnitPath := filepath.Join("/etc/systemd/system", mountUnitName)
	automountUnitPath := filepath.Join("/etc/systemd/system", automountUnitName)

	ui.Infof("Creating units: %s, %s", mountUnitName, automountUnitName)

	// Get NFS mount options from config or use default, add nofail for resilience.
	mountOptions := getNFSMountOptions(cfg)
	if !strings.Contains(mountOptions, "nofail") {
		mountOptions = "nofail," + mountOptions
	}
	ui.Infof("Using NFS mount options: %s", mountOptions)

	// Generate mount unit content.
	mountContent := fmt.Sprintf(`[Unit]
Description=NFS mount for %s
After=network-online.target

[Mount]
What=%s:%s
Where=%s
Type=nfs
Options=%s
TimeoutSec=30

[Install]
WantedBy=multi-user.target
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
	if err := system.WriteFile(mountUnitPath, []byte(mountContent), 0644); err != nil {
		return fmt.Errorf("failed to write mount unit %s: %w", mountUnitPath, err)
	}
	ui.Successf("Created mount unit: %s", mountUnitPath)

	// Write the automount unit file.
	if err := system.WriteFile(automountUnitPath, []byte(automountContent), 0644); err != nil {
		return fmt.Errorf("failed to write automount unit %s: %w", automountUnitPath, err)
	}
	ui.Successf("Created automount unit: %s", automountUnitPath)

	// Reload systemd to recognize the new units.
	cmd := exec.Command("sudo", "-n", "systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w\nOutput: %s", err, string(output))
	}
	ui.Success("systemd reloaded")

	// Enable and start the automount unit.
	cmd = exec.Command("sudo", "-n", "systemctl", "enable", "--now", automountUnitName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable and start automount unit: %w\nOutput: %s", err, string(output))
	}

	ui.Successf("Enabled and started automount unit: %s", automountUnitName)

	return nil
}

// RunNFSSetup executes the NFS configuration step
func RunNFSSetup(cfg *config.Config, ui *ui.UI) error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(cfg, nfsCompletionMarker, "nfs-configured", "nfs-skipped")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		ui.Info("NFS already configured (marker found)")
		ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + nfsCompletionMarker)
		return nil
	}

	ui.Header("NFS Configuration")
	ui.Info("Configure Network File System (NFS) mounts...")
	ui.Print("")

	// Ask if they want to configure NFS
	ui.Step("NFS Setup")
	useNFS, err := promptForNFS(cfg, ui)
	if err != nil {
		return fmt.Errorf("failed to prompt for NFS: %w", err)
	}

	if !useNFS {
		ui.Info("Skipping NFS configuration")
		ui.Info("To configure NFS later, remove marker: ~/.local/homelab-setup/" + nfsCompletionMarker)
		if err := cfg.MarkComplete(nfsCompletionMarker); err != nil {
			return fmt.Errorf("failed to create completion marker: %w", err)
		}
		return nil
	}

	// Check for nfs-utils package
	ui.Step("Checking NFS Prerequisites")
	if err := checkNFSUtils(cfg, ui); err != nil {
		return fmt.Errorf("NFS prerequisites check failed: %w", err)
	}

	// Get NFS details
	ui.Step("NFS Server Details")
	host, export, mountPoint, err := promptForNFSDetails(cfg, ui)
	if err != nil {
		return fmt.Errorf("failed to get NFS details: %w", err)
	}

	// Validate NFS connection
	ui.Step("Validating NFS Connection")
	if err := validateNFSConnection(cfg, ui, host); err != nil {
		ui.Error(fmt.Sprintf("NFS validation failed: %v", err))

		continueAnyway, err := ui.PromptYesNo("Continue with NFS setup despite validation errors?", false)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !continueAnyway {
			return fmt.Errorf("NFS setup cancelled due to validation errors")
		}
	}

	// Validate export path exists on server
	ui.Step("Verifying Export Path")
	if err := validateNFSExport(cfg, ui, host, export); err != nil {
		return fmt.Errorf("export path verification failed: %w", err)
	}

	// Create mount point
	ui.Step("Creating Mount Point")
	if err := createMountPoint(cfg, ui, mountPoint); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Create systemd mount and automount units
	ui.Step("Creating Systemd Units")
	if err := createSystemdUnits(cfg, ui, host, export, mountPoint); err != nil {
		return fmt.Errorf("failed to create systemd units: %w", err)
	}

	// The automount unit handles starting, so we don't need to manually start the mount unit.
	// We also don't need to check the status here as it will be mounted on first access.
	ui.Info("Systemd automount configured. The share will be mounted on first access.")

	// Save configuration
	ui.Step("Saving Configuration")
	if err := cfg.Set("NFS_SERVER", host); err != nil {
		return fmt.Errorf("failed to save NFS server: %w", err)
	}

	if err := cfg.Set("NFS_EXPORT", export); err != nil {
		return fmt.Errorf("failed to save NFS export: %w", err)
	}

	if err := cfg.Set("NFS_MOUNT_POINT", mountPoint); err != nil {
		return fmt.Errorf("failed to save NFS mount point: %w", err)
	}

	ui.Print("")
	ui.Separator()
	ui.Success("✓ NFS configuration completed")
	ui.Infof("Server: %s", host)
	ui.Infof("Export: %s", export)
	ui.Infof("Mount Point: %s", mountPoint)

	// Create completion marker
	if err := cfg.MarkComplete(nfsCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
