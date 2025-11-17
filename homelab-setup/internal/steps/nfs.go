package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// NFSConfigurator handles NFS mount configuration
type NFSConfigurator struct {
	fs      *system.FileSystem
	network *system.Network
	config  *config.Config
	ui      *ui.UI
	markers *config.Markers
	runner  system.CommandRunner
}

// NewNFSConfigurator creates a new NFSConfigurator instance
func NewNFSConfigurator(fs *system.FileSystem, network *system.Network, cfg *config.Config, ui *ui.UI, markers *config.Markers) *NFSConfigurator {
	return &NFSConfigurator{
		fs:      fs,
		network: network,
		config:  cfg,
		ui:      ui,
		markers: markers,
		runner:  system.NewCommandRunner(),
	}
}

func (n *NFSConfigurator) getFstabPath() string {
	path := n.config.GetOrDefault("NFS_FSTAB_PATH", "/etc/fstab")
	if path == "" {
		return "/etc/fstab"
	}
	return path
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

// ValidateNFSConnection validates the NFS server is accessible
func (n *NFSConfigurator) ValidateNFSConnection(host string) error {
	n.ui.Infof("Testing connection to NFS server %s...", host)

	// Test basic connectivity
	reachable, err := n.network.TestConnectivity(host, 5)
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

// CreateSystemdMountUnit creates a systemd mount unit for NFS
func (n *NFSConfigurator) CreateSystemdMountUnit(host, export, mountPoint string) error {
	n.ui.Info("Creating systemd mount unit...")

	// Convert mount point to systemd unit name
	// Example: /mnt/nas-media -> mnt-nas\x2dmedia.mount
	unitName, err := pathToUnitName(n.runner, mountPoint)
	if err != nil {
		return fmt.Errorf("failed to escape mount point %s: %w", mountPoint, err)
	}
	unitPath := filepath.Join("/etc/systemd/system", unitName)

	n.ui.Infof("Unit name: %s", unitName)

	// Generate mount unit content
	content := fmt.Sprintf(`[Unit]
Description=NFS mount for %s
After=network-online.target
Requires=network-online.target
Wants=network-online.target

[Mount]
What=%s:%s
Where=%s
Type=nfs
Options=defaults,nfsvers=4.2,_netdev
TimeoutSec=30

[Install]
WantedBy=multi-user.target
`, mountPoint, host, export, mountPoint)

	// Check if unit already exists
	existingContent, err := os.ReadFile(unitPath)
	if err == nil {
		// Unit exists, check if content is the same
		if string(existingContent) == content {
			n.ui.Info("Mount unit already exists with correct configuration")
			return nil
		}
		n.ui.Info("Updating existing mount unit")
	}

	// Write the mount unit file
	if err := n.fs.WriteFile(unitPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write mount unit %s: %w", unitPath, err)
	}

	n.ui.Success(fmt.Sprintf("Created mount unit: %s", unitPath))

	// Reload systemd to recognize the new unit
	if output, err := n.runner.Run("sudo", "-n", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w\nOutput: %s", err, output)
	}

	n.ui.Success("systemd reloaded")

	// Enable the mount unit
	if output, err := n.runner.Run("sudo", "-n", "systemctl", "enable", unitName); err != nil {
		return fmt.Errorf("failed to enable mount unit: %w\nOutput: %s", err, output)
	}

	n.ui.Success(fmt.Sprintf("Enabled mount unit: %s", unitName))

	return nil
}

// pathToUnitName converts a mount point path to a systemd unit name
// Example: /mnt/nas-media -> mnt-nas\x2dmedia.mount
func pathToUnitName(runner system.CommandRunner, mountPoint string) (string, error) {
	output, err := runner.Run("systemd-escape", "--path", "--suffix=mount", mountPoint)
	if err != nil {
		return "", fmt.Errorf("systemd-escape failed: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// AddToFstab adds NFS mount to /etc/fstab (deprecated, kept for compatibility)
func (n *NFSConfigurator) AddToFstab(host, export, mountPoint string) error {
	n.ui.Info("Adding NFS mount to /etc/fstab...")

	entry := fmt.Sprintf("%s:%s %s nfs defaults,nfsvers=4.2,_netdev 0 0", host, export, mountPoint)
	n.ui.Info("Fstab entry:")
	n.ui.Printf("  %s", entry)
	n.ui.Print("")
	fstabPath := n.getFstabPath()

	existing, err := os.ReadFile(fstabPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", fstabPath, err)
		}
		// If the file doesn't exist, ensure the directory exists
		dir := filepath.Dir(fstabPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", fstabPath, err)
		}
	}

	entryExists := false
	if len(existing) > 0 {
		for _, line := range strings.Split(string(existing), "\n") {
			if strings.TrimSpace(line) == entry {
				entryExists = true
				break
			}
		}
	}

	if entryExists {
		n.ui.Info("Fstab entry already exists; skipping append")
	} else {
		var builder strings.Builder
		if len(existing) > 0 {
			builder.Write(existing)
			if !strings.HasSuffix(string(existing), "\n") {
				builder.WriteString("\n")
			}
		}
		builder.WriteString(entry)
		builder.WriteString("\n")

		if err := n.fs.WriteFile(fstabPath, []byte(builder.String()), 0644); err != nil {
			return fmt.Errorf("failed to update %s: %w", fstabPath, err)
		}
		successMessage := "fstab entry"
		if fstabPath != "/etc/fstab" {
			successMessage = fmt.Sprintf("fstab entry in %s", fstabPath)
		}
		n.ui.Success(fmt.Sprintf("Created %s", successMessage))
	}

	if output, err := n.runner.Run("sudo", "-n", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd after fstab update: %w\nOutput: %s", err, output)
	}

	n.ui.Success("systemd reloaded to pick up new mount units")
	return nil
}

// MountNFS attempts to mount the NFS share
func (n *NFSConfigurator) MountNFS(mountPoint string) error {
	n.ui.Infof("Mounting NFS share at %s...", mountPoint)

	if output, err := n.runner.Run("sudo", "-n", "mount", mountPoint); err != nil {
		return fmt.Errorf("failed to mount %s: %w\nOutput: %s", mountPoint, err, output)
	}

	n.ui.Success("NFS share mounted successfully")
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

	// Create mount point
	n.ui.Step("Creating Mount Point")
	if err := n.CreateMountPoint(mountPoint); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Create systemd mount unit
	n.ui.Step("Creating Systemd Mount Unit")
	if err := n.CreateSystemdMountUnit(host, export, mountPoint); err != nil {
		return fmt.Errorf("failed to create systemd mount unit: %w", err)
	}

	// Start the mount unit
	n.ui.Step("Starting Mount Unit")
	unitName, err := pathToUnitName(n.runner, mountPoint)
	if err != nil {
		return fmt.Errorf("failed to escape mount point %s: %w", mountPoint, err)
	}
	if output, err := n.runner.Run("sudo", "-n", "systemctl", "start", unitName); err != nil {
		n.ui.Warning(fmt.Sprintf("Failed to start mount unit: %v", err))
		n.ui.Info("Output: " + output)
		n.ui.Info("The mount will be activated automatically when needed")
	} else {
		n.ui.Success("NFS share mounted successfully")
	}

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
