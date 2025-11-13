package steps

import (
	"fmt"
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
}

// NewNFSConfigurator creates a new NFSConfigurator instance
func NewNFSConfigurator(fs *system.FileSystem, network *system.Network, cfg *config.Config, ui *ui.UI, markers *config.Markers) *NFSConfigurator {
	return &NFSConfigurator{
		fs:      fs,
		network: network,
		config:  cfg,
		ui:      ui,
		markers: markers,
	}
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

	// Validate export path
	if err := common.ValidatePath(export); err != nil {
		return "", "", "", fmt.Errorf("invalid export path: %w", err)
	}

	// Prompt for mount point
	mountPoint, err = n.ui.PromptInput("Local mount point", "/mnt/nas-media")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to prompt for mount point: %w", err)
	}

	// Validate mount point
	if err := common.ValidatePath(mountPoint); err != nil {
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

// AddToFstab adds NFS mount to /etc/fstab
func (n *NFSConfigurator) AddToFstab(host, export, mountPoint string) error {
	n.ui.Info("Adding NFS mount to /etc/fstab...")

	// Construct fstab entry
	// Format: server:/export /mountpoint nfs defaults,nfsvers=4.2,_netdev 0 0
	fstabEntry := fmt.Sprintf("%s:%s %s nfs defaults,nfsvers=4.2,_netdev 0 0\n", host, export, mountPoint)

	n.ui.Info("Fstab entry:")
	n.ui.Printf("  %s", strings.TrimSpace(fstabEntry))
	n.ui.Print("")

	// TODO: Implement WriteFile method in FileSystem to append to /etc/fstab
	// For now, provide manual instructions
	n.ui.Warning("Automatic fstab modification not yet implemented")
	n.ui.Info("Please manually add the following line to /etc/fstab:")
	n.ui.Printf("  %s", strings.TrimSpace(fstabEntry))
	n.ui.Print("")
	n.ui.Info("To add it:")
	n.ui.Infof("  echo '%s' | sudo tee -a /etc/fstab", strings.TrimSpace(fstabEntry))
	n.ui.Print("")

	addNow, err := n.ui.PromptYesNo("Have you added this line to /etc/fstab?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if !addNow {
		n.ui.Warning("You will need to manually add this entry to /etc/fstab later")
		return nil
	}

	return nil
}

// MountNFS attempts to mount the NFS share
func (n *NFSConfigurator) MountNFS(mountPoint string) error {
	n.ui.Infof("Mounting NFS share at %s...", mountPoint)

	// TODO: Implement mount command execution
	// For now, provide manual instructions
	n.ui.Info("To mount the share:")
	n.ui.Infof("  sudo mount %s", mountPoint)
	n.ui.Print("")

	mounted, err := n.ui.PromptYesNo("Have you mounted the share?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if !mounted {
		n.ui.Warning("NFS share not mounted")
		n.ui.Infof("You can mount it later with: sudo mount %s", mountPoint)
		return nil
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

	// Add to fstab
	n.ui.Step("Configuring /etc/fstab")
	if err := n.AddToFstab(host, export, mountPoint); err != nil {
		return fmt.Errorf("failed to add to fstab: %w", err)
	}

	// Mount NFS share
	n.ui.Step("Mounting NFS Share")
	if err := n.MountNFS(mountPoint); err != nil {
		n.ui.Warning(fmt.Sprintf("Failed to mount NFS: %v", err))
		// Non-critical error, save config anyway
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
