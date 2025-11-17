package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var troubleshootCmd = &cobra.Command{
	Use:   "troubleshoot",
	Short: "Run troubleshooting diagnostics",
	Long:  `Run diagnostic checks to troubleshoot common issues.`,
	RunE:  runTroubleshoot,
}

var (
	troubleshootAll      bool
	troubleshootServices bool
	troubleshootNetwork  bool
	troubleshootStorage  bool
)

func init() {
	rootCmd.AddCommand(troubleshootCmd)
	troubleshootCmd.Flags().BoolVarP(&troubleshootAll, "all", "a", false, "Run all diagnostics")
	troubleshootCmd.Flags().BoolVarP(&troubleshootServices, "services", "s", false, "Check services and containers only")
	troubleshootCmd.Flags().BoolVarP(&troubleshootNetwork, "network", "n", false, "Check network connectivity only")
	troubleshootCmd.Flags().BoolVarP(&troubleshootStorage, "storage", "d", false, "Check storage and disk usage only")
}

func runTroubleshoot(cmd *cobra.Command, args []string) error {
	ctx, err := cli.NewSetupContext()
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	ctx.UI.Header("Homelab Troubleshooting Tool")

	// Determine what to run based on flags
	runAll := troubleshootAll || (!troubleshootServices && !troubleshootNetwork && !troubleshootStorage)

	if runAll || troubleshootServices {
		checkSystemInfo(ctx)
		checkConfiguration(ctx)
		checkServices(ctx)
		checkContainers(ctx)
	}

	if runAll || troubleshootNetwork {
		checkNetwork(ctx)
		checkWireGuard(ctx)
	}

	if runAll || troubleshootStorage {
		checkNFSMounts(ctx)
		checkDiskUsage(ctx)
	}

	ctx.UI.Print("")
	ctx.UI.Separator()
	ctx.UI.Info("For detailed service logs, use:")
	ctx.UI.Info("  sudo journalctl -u <service-name> -n 50")
	ctx.UI.Info("For container logs, use:")
	ctx.UI.Info("  podman logs <container-name>")

	return nil
}

// ============================================================================
// System Information
// ============================================================================

func checkSystemInfo(ctx *cli.SetupContext) {
	ctx.UI.Header("System Information")

	// OS Information
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "NAME=") || strings.HasPrefix(line, "VERSION=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					value := strings.Trim(parts[1], "\"")
					ctx.UI.Infof("  %s", value)
				}
			}
		}
	}

	// Kernel version
	if output, err := exec.Command("uname", "-r").Output(); err == nil {
		ctx.UI.Infof("  Kernel: %s", strings.TrimSpace(string(output)))
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		ctx.UI.Infof("  Hostname: %s", hostname)
	}

	// Uptime
	if output, err := exec.Command("uptime", "-p").Output(); err == nil {
		ctx.UI.Infof("  Uptime: %s", strings.TrimSpace(string(output)))
	}

	// RPM-OSTree status (if available)
	if _, err := exec.LookPath("rpm-ostree"); err == nil {
		ctx.UI.Print("")
		ctx.UI.Info("RPM-OSTree Status:")
		if output, err := exec.Command("rpm-ostree", "status", "--booted").Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if line != "" {
					ctx.UI.Infof("  %s", line)
				}
			}
		}
	}

	ctx.UI.Print("")
}

// ============================================================================
// Configuration Status
// ============================================================================

func checkConfiguration(ctx *cli.SetupContext) {
	ctx.UI.Header("Configuration Status")

	configFile := ctx.Config.FilePath()
	if _, err := os.Stat(configFile); err == nil {
		ctx.UI.Successf("Configuration file exists: %s", configFile)

		// Show key configurations
		ctx.UI.Print("")
		ctx.UI.Info("Key configurations:")
		keyConfigs := []string{"SETUP_USER", "ENV_PUID", "ENV_PGID", "ENV_TZ", "NFS_SERVER", "CONTAINERS_BASE", "APPDATA_BASE"}
		for _, key := range keyConfigs {
			if value := ctx.Config.GetOrDefault(key, ""); value != "" {
				ctx.UI.Infof("  %s=%s", key, value)
			}
		}
	} else {
		ctx.UI.Warning("Configuration file not found")
		ctx.UI.Info("  Run 'homelab-setup run' to create configuration")
	}

	ctx.UI.Print("")

	// Check setup markers
	markers, err := ctx.Markers.List()
	if err != nil {
		ctx.UI.Error(fmt.Sprintf("Failed to list markers: %v", err))
	} else {
		ctx.UI.Info("Completed setup steps:")
		if len(markers) == 0 {
			ctx.UI.Info("  (none - setup not started)")
		} else {
			for _, marker := range markers {
				ctx.UI.Successf("  ✓ %s", marker)
			}
		}
	}

	ctx.UI.Print("")
}

// ============================================================================
// Service Status
// ============================================================================

func checkServices(ctx *cli.SetupContext) {
	ctx.UI.Header("Systemd Service Status")

	services := []string{
		"podman-compose-media.service",
		"podman-compose-web.service",
		"podman-compose-cloud.service",
	}

	for _, service := range services {
		ctx.UI.Infof("Checking: %s", service)

		// Check if service exists and is active
		checkCmd := exec.Command("systemctl", "is-active", "--quiet", service)
		if err := checkCmd.Run(); err == nil {
			ctx.UI.Successf("  Status: Active")

			// Show brief status
			if output, err := exec.Command("systemctl", "status", service, "--no-pager", "-n", "3").Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				for i, line := range lines {
					if i < 5 && line != "" {
						ctx.UI.Infof("    %s", strings.TrimSpace(line))
					}
				}
			}
		} else {
			ctx.UI.Warning("  Status: Inactive or not found")

			// Check if it failed
			failedCmd := exec.Command("systemctl", "is-failed", "--quiet", service)
			if err := failedCmd.Run(); err == nil {
				ctx.UI.Error("  Service has failed!")
				ctx.UI.Infof("  View logs: sudo journalctl -u %s -n 50", service)
			}
		}
		ctx.UI.Print("")
	}
}

// ============================================================================
// Container Status
// ============================================================================

func checkContainers(ctx *cli.SetupContext) {
	ctx.UI.Header("Container Status")

	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		ctx.UI.Error("Podman not available")
		ctx.UI.Print("")
		return
	}

	// Get running containers
	output, err := exec.Command("podman", "ps", "--format", "{{.Names}}").Output()
	if err != nil {
		ctx.UI.Error(fmt.Sprintf("Failed to list containers: %v", err))
		ctx.UI.Print("")
		return
	}

	runningContainers := strings.Split(strings.TrimSpace(string(output)), "\n")
	runningCount := 0
	for _, name := range runningContainers {
		if name != "" {
			runningCount++
		}
	}

	ctx.UI.Infof("Running containers: %d", runningCount)
	ctx.UI.Print("")

	if runningCount > 0 {
		// Show container table
		if tableOutput, err := exec.Command("podman", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}").Output(); err == nil {
			ctx.UI.Info(string(tableOutput))
		}
	} else {
		ctx.UI.Warning("No containers are running")

		// Check for stopped containers
		if stoppedOutput, err := exec.Command("podman", "ps", "-a", "--filter", "status=exited", "--format", "{{.Names}}").Output(); err == nil {
			stoppedContainers := strings.Split(strings.TrimSpace(string(stoppedOutput)), "\n")
			stoppedCount := 0
			for _, name := range stoppedContainers {
				if name != "" {
					stoppedCount++
				}
			}

			if stoppedCount > 0 {
				ctx.UI.Warningf("Found %d stopped container(s)", stoppedCount)
				if tableOutput, err := exec.Command("podman", "ps", "-a", "--filter", "status=exited", "--format", "table {{.Names}}\t{{.Status}}").Output(); err == nil {
					ctx.UI.Info(string(tableOutput))
				}
			}
		}
	}

	// Check for containers in error state
	if errorOutput, err := exec.Command("podman", "ps", "-a", "--filter", "status=error", "--format", "{{.Names}}").Output(); err == nil {
		errorContainers := strings.Split(strings.TrimSpace(string(errorOutput)), "\n")
		errorCount := 0
		for _, name := range errorContainers {
			if name != "" {
				errorCount++
			}
		}

		if errorCount > 0 {
			ctx.UI.Errorf("Found %d container(s) in error state!", errorCount)
			if tableOutput, err := exec.Command("podman", "ps", "-a", "--filter", "status=error", "--format", "table {{.Names}}\t{{.Status}}").Output(); err == nil {
				ctx.UI.Info(string(tableOutput))
			}
		}
	}

	ctx.UI.Print("")
}

// ============================================================================
// Network Diagnostics
// ============================================================================

func checkNetwork(ctx *cli.SetupContext) {
	ctx.UI.Header("Network Diagnostics")

	// Default gateway
	if output, err := exec.Command("ip", "route", "show", "default").Output(); err == nil {
		line := strings.TrimSpace(string(output))
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				gateway := parts[2]
				ctx.UI.Successf("Default gateway: %s", gateway)

				// Test gateway reachability
				pingCmd := exec.Command("ping", "-c", "1", "-W", "2", gateway)
				if err := pingCmd.Run(); err == nil {
					ctx.UI.Success("  Gateway is reachable")
				} else {
					ctx.UI.Error("  Gateway is not reachable")
				}
			}
		}
	} else {
		ctx.UI.Error("No default gateway configured")
	}

	// Internet connectivity
	ctx.UI.Info("Testing internet connectivity...")
	pingCmd := exec.Command("ping", "-c", "1", "-W", "3", "8.8.8.8")
	if err := pingCmd.Run(); err == nil {
		ctx.UI.Success("  Internet is reachable")
	} else {
		ctx.UI.Error("  Internet is not reachable")
	}

	// DNS resolution
	ctx.UI.Info("Testing DNS resolution...")
	hostCmd := exec.Command("host", "google.com")
	if err := hostCmd.Run(); err == nil {
		ctx.UI.Success("  DNS is working")
	} else {
		ctx.UI.Error("  DNS resolution failed")
	}

	ctx.UI.Print("")
}

// ============================================================================
// WireGuard Status
// ============================================================================

func checkWireGuard(ctx *cli.SetupContext) {
	ctx.UI.Header("WireGuard VPN Status")

	// Check if WireGuard tools are installed
	if _, err := exec.LookPath("wg"); err != nil {
		ctx.UI.Warning("WireGuard tools not installed")
		ctx.UI.Print("")
		return
	}

	// Check if wg-quick@wg0 service is active
	checkCmd := exec.Command("systemctl", "is-active", "--quiet", "wg-quick@wg0.service")
	if err := checkCmd.Run(); err == nil {
		ctx.UI.Success("WireGuard service is active")

		// Check if interface exists
		if _, err := exec.Command("ip", "link", "show", "wg0").Output(); err == nil {
			ctx.UI.Success("WireGuard interface exists")

			// Get interface IP
			if ipOutput, err := exec.Command("ip", "addr", "show", "wg0").Output(); err == nil {
				lines := strings.Split(string(ipOutput), "\n")
				for _, line := range lines {
					if strings.Contains(line, "inet ") {
						parts := strings.Fields(line)
						if len(parts) >= 2 {
							ctx.UI.Infof("  Interface IP: %s", parts[1])
						}
					}
				}
			}

			// Show peer status (requires sudo)
			ctx.UI.Print("")
			ctx.UI.Info("Peer status (run with sudo for details):")
			if wgOutput, err := exec.Command("sudo", "wg", "show", "wg0").Output(); err == nil {
				ctx.UI.Info(string(wgOutput))
			}
		} else {
			ctx.UI.Error("WireGuard interface not found")
		}
	} else {
		ctx.UI.Warning("WireGuard service is not active")
		ctx.UI.Info("  Start with: sudo systemctl start wg-quick@wg0.service")
	}

	ctx.UI.Print("")
}

// ============================================================================
// NFS Mount Status
// ============================================================================

func checkNFSMounts(ctx *cli.SetupContext) {
	ctx.UI.Header("NFS Mount Status")

	mounts := []string{
		"/mnt/nas-media",
		"/mnt/nas-nextcloud",
		"/mnt/nas-immich",
		"/mnt/nas-photos",
	}

	for _, mount := range mounts {
		// Check if mount point is mounted
		checkCmd := exec.Command("mountpoint", "-q", mount)
		if err := checkCmd.Run(); err == nil {
			ctx.UI.Successf("%s is mounted", mount)

			// Test read access
			if entries, err := os.ReadDir(mount); err == nil {
				ctx.UI.Successf("  Readable: Yes (%d entries)", len(entries))
			} else {
				ctx.UI.Error("  Readable: No")
			}

			// Test write access
			testFile := filepath.Join(mount, ".write-test")
			if file, err := os.Create(testFile); err == nil {
				file.Close()
				os.Remove(testFile)
				ctx.UI.Success("  Writable: Yes")
			} else {
				ctx.UI.Info("  Writable: No (may be read-only)")
			}
		} else {
			ctx.UI.Errorf("%s is NOT mounted", mount)

			// Get mount unit name
			escapedPath := strings.ReplaceAll(mount, "/", "-")
			escapedPath = strings.TrimPrefix(escapedPath, "-")
			unitName := escapedPath + ".mount"

			ctx.UI.Infof("  Mount unit: %s", unitName)
			ctx.UI.Infof("  Start with: sudo systemctl start %s", unitName)

			// Check if failed
			failedCmd := exec.Command("systemctl", "is-failed", "--quiet", unitName)
			if err := failedCmd.Run(); err == nil {
				ctx.UI.Error("  Mount unit has failed!")
				ctx.UI.Infof("  View logs: sudo journalctl -u %s", unitName)
			}
		}
		ctx.UI.Print("")
	}
}

// ============================================================================
// Disk Usage
// ============================================================================

func checkDiskUsage(ctx *cli.SetupContext) {
	ctx.UI.Header("Disk Usage")

	filesystems := []string{"/", "/var", "/srv", "/mnt"}

	for _, fs := range filesystems {
		// Get disk usage with df
		output, err := exec.Command("df", "-h", fs).Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")
		if len(lines) < 2 {
			continue
		}

		// Parse df output (line 2 has the data)
		fields := strings.Fields(lines[1])
		if len(fields) < 5 {
			continue
		}

		usagePercent := strings.TrimSuffix(fields[4], "%")
		available := fields[3]

		// Parse usage as integer
		var usage int
		if _, err := fmt.Sscanf(usagePercent, "%d", &usage); err != nil {
			// If parsing fails, skip this filesystem
			continue
		}

		if usage >= 90 {
			ctx.UI.Errorf("%s: %s%% used (%s available) - CRITICAL", fs, usagePercent, available)
		} else if usage >= 80 {
			ctx.UI.Warningf("%s: %s%% used (%s available) - WARNING", fs, usagePercent, available)
		} else {
			ctx.UI.Successf("%s: %s%% used (%s available)", fs, usagePercent, available)
		}
	}

	ctx.UI.Print("")

	// Container storage (if podman available)
	if _, err := exec.LookPath("podman"); err == nil {
		ctx.UI.Info("Container storage:")
		if output, err := exec.Command("podman", "system", "df").Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if line != "" {
					ctx.UI.Infof("  %s", line)
				}
			}
		}
	}

	ctx.UI.Print("")
}
