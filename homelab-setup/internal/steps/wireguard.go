package steps

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/common"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// WireGuardConfig holds WireGuard configuration
type WireGuardConfig struct {
	InterfaceName string
	InterfaceIP   string
	ListenPort    string
	PrivateKey    string
	PublicKey     string
}

// WireGuardPeer holds peer configuration
type WireGuardPeer struct {
	Name       string // Human-readable name for reference
	PublicKey  string
	AllowedIPs string
	Endpoint   string // Optional
}

// WireGuardSetup handles WireGuard VPN setup
type WireGuardSetup struct {
	packages *system.PackageManager
	services *system.ServiceManager
	fs       *system.FileSystem
	network  *system.Network
	config   *config.Config
	ui       *ui.UI
	markers  *config.Markers
}

// NewWireGuardSetup creates a new WireGuardSetup instance
func NewWireGuardSetup(packages *system.PackageManager, services *system.ServiceManager, fs *system.FileSystem, network *system.Network, cfg *config.Config, ui *ui.UI, markers *config.Markers) *WireGuardSetup {
	return &WireGuardSetup{
		packages: packages,
		services: services,
		fs:       fs,
		network:  network,
		config:   cfg,
		ui:       ui,
		markers:  markers,
	}
}

func (w *WireGuardSetup) configDir() string {
	return w.config.GetOrDefault("WIREGUARD_CONFIG_DIR", "/etc/wireguard")
}

// PromptForWireGuard asks if the user wants to configure WireGuard
func (w *WireGuardSetup) PromptForWireGuard() (bool, error) {
	w.ui.Info("WireGuard is a modern, fast VPN protocol")
	w.ui.Info("It can be used to:")
	w.ui.Info("  - Securely connect to your homelab from anywhere")
	w.ui.Info("  - Create encrypted tunnels to a VPS for external access")
	w.ui.Info("  - Build a mesh network between devices")
	w.ui.Print("")

	useWireGuard, err := w.ui.PromptYesNo("Do you want to configure WireGuard?", false)
	if err != nil {
		return false, fmt.Errorf("failed to prompt for WireGuard: %w", err)
	}

	return useWireGuard, nil
}

// CheckWireGuardInstalled checks if WireGuard tools are installed
func (w *WireGuardSetup) CheckWireGuardInstalled() error {
	w.ui.Info("Checking for WireGuard tools...")

	installed, err := w.packages.IsInstalled("wireguard-tools")
	if err != nil {
		return fmt.Errorf("failed to check wireguard-tools: %w", err)
	}

	if !installed {
		w.ui.Warning("wireguard-tools not installed")
		w.ui.Info("To install:")
		w.ui.Info("  sudo rpm-ostree install wireguard-tools")
		w.ui.Info("  sudo systemctl reboot")
		return fmt.Errorf("wireguard-tools not installed")
	}

	w.ui.Success("wireguard-tools is installed")

	// Check for wg command
	if !system.CommandExists("wg") {
		w.ui.Warning("wg command not found")
		return fmt.Errorf("wg command not available")
	}

	w.ui.Success("wg command is available")
	return nil
}

// GenerateKeys generates WireGuard private and public keys
func (w *WireGuardSetup) GenerateKeys() (privateKey, publicKey string, err error) {
	w.ui.Info("Generating WireGuard keys...")

	// Generate private key
	privCmd := exec.Command("wg", "genkey")
	privOutput, err := privCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}
	privateKey = strings.TrimSpace(string(privOutput))

	// Generate public key from private key
	pubCmd := exec.Command("wg", "pubkey")
	pubCmd.Stdin = strings.NewReader(privateKey)
	pubOutput, err := pubCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKey = strings.TrimSpace(string(pubOutput))

	w.ui.Success("Keys generated successfully")
	w.ui.Print("")
	w.ui.Info("Public key (share with peers):")
	w.ui.Printf("  %s", publicKey)
	w.ui.Print("")
	w.ui.Warning("Private key (keep secret!):")
	w.ui.Printf("  %s", privateKey)
	w.ui.Print("")

	return privateKey, publicKey, nil
}

// PromptForConfig prompts for WireGuard configuration
func (w *WireGuardSetup) PromptForConfig(publicKey string) (*WireGuardConfig, error) {
	w.ui.Print("")
	w.ui.Info("WireGuard Interface Configuration:")
	w.ui.Print("")

	cfg := &WireGuardConfig{
		PublicKey: publicKey,
	}

	// Prompt for interface name
	interfaceName, err := w.ui.PromptInput("Interface name", "wg0")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for interface name: %w", err)
	}
	cfg.InterfaceName = interfaceName

	// Prompt for interface IP
	interfaceIP, err := w.ui.PromptInput("Interface IP address (CIDR notation)", "10.253.0.1/24")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for interface IP: %w", err)
	}
	if err := common.ValidateCIDR(interfaceIP); err != nil {
		return nil, fmt.Errorf("invalid interface IP: %w", err)
	}
	cfg.InterfaceIP = interfaceIP

	// Prompt for listen port
	listenPort, err := w.ui.PromptInput("Listen port", "51820")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for listen port: %w", err)
	}

	// Validate port
	if err := common.ValidatePort(listenPort); err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	cfg.ListenPort = listenPort

	return cfg, nil
}

// WriteConfig writes the WireGuard configuration file
func (w *WireGuardSetup) WriteConfig(cfg *WireGuardConfig, privateKey string) error {
	w.ui.Infof("Writing WireGuard configuration for %s...", cfg.InterfaceName)

	configContent := fmt.Sprintf(`[Interface]
# WireGuard interface configuration
# Generated by homelab-setup

Address = %s
ListenPort = %s
PrivateKey = %s

# To add peers, add sections like:
# [Peer]
# PublicKey = <peer-public-key>
# AllowedIPs = 10.253.0.2/32
# Endpoint = <peer-ip>:51820
`, cfg.InterfaceIP, cfg.ListenPort, privateKey)

	w.ui.Print("")
	w.ui.Info("Configuration file content:")
	w.ui.Print(configContent)
	w.ui.Print("")

	configDir := w.configDir()
	configPath := filepath.Join(configDir, fmt.Sprintf("%s.conf", cfg.InterfaceName))

	if err := w.fs.EnsureDirectory(configDir, "root:root", 0750); err != nil {
		return fmt.Errorf("failed to ensure WireGuard directory %s: %w", configDir, err)
	}

	if err := w.fs.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write WireGuard config: %w", err)
	}

	exists, err := w.fs.FileExists(configPath)
	if err != nil {
		return fmt.Errorf("failed to verify config file: %w", err)
	}
	if !exists {
		return fmt.Errorf("WireGuard config %s was not created", configPath)
	}

	perms, err := w.fs.GetPermissions(configPath)
	if err != nil {
		return fmt.Errorf("failed to check permissions on %s: %w", configPath, err)
	}
	if perms.Perm() != 0600 {
		return fmt.Errorf("WireGuard config %s must have 0600 permissions, found %o", configPath, perms.Perm())
	}

	w.ui.Successf("Configuration file created at %s", configPath)
	w.ui.Info("Review the file to add peers as needed")

	return nil
}

// EnableService enables and starts the WireGuard service
func (w *WireGuardSetup) EnableService(interfaceName string) error {
	serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)

	w.ui.Print("")
	w.ui.Info("The WireGuard service needs to be enabled and started.")
	w.ui.Print("")

	autoEnable, err := w.ui.PromptYesNo("Do you want to enable and start the service now?", true)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if !autoEnable {
		w.ui.Print("")
		w.ui.Info("To enable and start the service manually:")
		w.ui.Infof("  sudo systemctl enable %s", serviceName)
		w.ui.Infof("  sudo systemctl start %s", serviceName)
		w.ui.Print("")
		w.ui.Warning("WireGuard service not started")
		return nil
	}

	w.ui.Print("")
	w.ui.Infof("Enabling %s...", serviceName)

	// Enable service
	if err := w.services.Enable(serviceName); err != nil {
		w.ui.Warning(fmt.Sprintf("Failed to enable service: %v", err))
		w.ui.Info("You may need to run manually:")
		w.ui.Infof("  sudo systemctl enable %s", serviceName)
		return fmt.Errorf("failed to enable service: %w", err)
	}
	w.ui.Success("Service enabled")

	// Start service
	w.ui.Infof("Starting %s...", serviceName)
	if err := w.services.Start(serviceName); err != nil {
		w.ui.Warning(fmt.Sprintf("Failed to start service: %v", err))
		w.ui.Info("You may need to run manually:")
		w.ui.Infof("  sudo systemctl start %s", serviceName)
		return fmt.Errorf("failed to start service: %w", err)
	}
	w.ui.Success("Service started")

	// Check if service is actually running
	active, err := w.services.IsActive(serviceName)
	if err != nil {
		w.ui.Warning(fmt.Sprintf("Could not verify service status: %v", err))
	} else if active {
		w.ui.Success("WireGuard service is running")
	} else {
		w.ui.Warning("Service may not be running correctly")
	}

	// Display status instructions
	w.ui.Print("")
	w.ui.Info("To check WireGuard status:")
	w.ui.Infof("  sudo systemctl status %s", serviceName)
	w.ui.Infof("  sudo wg show %s", interfaceName)

	return nil
}

// PromptForPeer prompts for peer configuration
func (w *WireGuardSetup) PromptForPeer(nextIP string) (*WireGuardPeer, error) {
	peer := &WireGuardPeer{}

	w.ui.Print("")

	// Prompt for peer name
	name, err := w.ui.PromptInput("Peer name (e.g., 'laptop', 'phone', 'vps')", "")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for peer name: %w", err)
	}
	if name == "" {
		return nil, fmt.Errorf("peer name is required")
	}
	peer.Name = name

	// Prompt for public key
	publicKey, err := w.ui.PromptInput("Peer public key", "")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for public key: %w", err)
	}
	if publicKey == "" {
		return nil, fmt.Errorf("public key is required")
	}
	peer.PublicKey = publicKey

	// Prompt for allowed IPs
	allowedIPs, err := w.ui.PromptInput("Allowed IPs for this peer", nextIP)
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for allowed IPs: %w", err)
	}
	if allowedIPs == "" {
		return nil, fmt.Errorf("allowed IPs are required")
	}
	// Validate each CIDR in AllowedIPs (comma-separated)
	for _, cidr := range strings.Split(allowedIPs, ",") {
		cidr = strings.TrimSpace(cidr)
		if err := common.ValidateCIDR(cidr); err != nil {
			return nil, fmt.Errorf("invalid allowed IPs CIDR '%s': %v", cidr, err)
		}
	}
	peer.AllowedIPs = allowedIPs

	// Prompt for endpoint (optional)
	w.ui.Info("Endpoint is optional - leave empty for road warrior clients")
	endpoint, err := w.ui.PromptInput("Endpoint (e.g., 'server.example.com:51820')", "")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for endpoint: %w", err)
	}
	peer.Endpoint = endpoint

	return peer, nil
}

// AddPeerToConfig appends a peer configuration to the WireGuard config file
func (w *WireGuardSetup) AddPeerToConfig(interfaceName string, peer *WireGuardPeer) error {
	configPath := filepath.Join(w.configDir(), fmt.Sprintf("%s.conf", interfaceName))

	// Read current config (using sudo cat to handle permissions)
	cmd := exec.Command("sudo", "-n", "cat", configPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Build peer section
	peerSection := fmt.Sprintf("\n# Peer: %s\n[Peer]\nPublicKey = %s\nAllowedIPs = %s\n",
		peer.Name, peer.PublicKey, peer.AllowedIPs)

	if peer.Endpoint != "" {
		peerSection += fmt.Sprintf("Endpoint = %s\n", peer.Endpoint)
	}

	// Append peer to config
	newContent := string(output) + peerSection

	// Write updated config
	if err := w.fs.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	w.ui.Successf("Peer '%s' added to configuration", peer.Name)
	return nil
}

// AddPeers interactively adds WireGuard peers
func (w *WireGuardSetup) AddPeers(interfaceName, publicKey, interfaceIP string) error {
	w.ui.Print("")
	w.ui.Info("WireGuard Peer Configuration:")
	w.ui.Separator()
	w.ui.Print("")

	w.ui.Info("Your server public key:")
	w.ui.Printf("  %s", publicKey)
	w.ui.Print("")

	addPeers, err := w.ui.PromptYesNo("Do you want to add peers now?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt for adding peers: %w", err)
	}

	if !addPeers {
		w.ui.Print("")
		w.ui.Info("You can add peers later by editing:")
		w.ui.Infof("  %s", filepath.Join(w.configDir(), fmt.Sprintf("%s.conf", interfaceName)))
		w.ui.Print("")
		w.ui.Info("After editing, restart the service:")
		w.ui.Infof("  sudo systemctl restart wg-quick@%s", interfaceName)
		return nil
	}

	// Parse the interface IP to suggest next IP for peers
	// For example, if server is 10.253.0.1/24, suggest 10.253.0.2/32 for first peer
	nextIP := "10.253.0.2/32"
	if strings.Contains(interfaceIP, "/") {
		parts := strings.Split(interfaceIP, "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			ipParts := strings.Split(parts[0], ".")
			if len(ipParts) == 4 {
				nextIP = fmt.Sprintf("%s.%s.%s.2/32", ipParts[0], ipParts[1], ipParts[2])
			}
		} else {
			// Malformed interfaceIP, fallback to default suggestion
			nextIP = "10.253.0.2/32"
		}
	}

	peerCount := 0
	for {
		w.ui.Print("")
		w.ui.Infof("Adding peer #%d", peerCount+1)

		peer, err := w.PromptForPeer(nextIP)
		if err != nil {
			// Check if the error is non-recoverable (e.g., EOF, input stream closed)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe) {
				w.ui.Error(fmt.Sprintf("Input stream closed: %v", err))
				break
			}
			// For recoverable errors (e.g., validation errors), show warning and retry
			w.ui.Warning(fmt.Sprintf("Failed to get peer configuration: %v", err))
			continue
		}

		if err := w.AddPeerToConfig(interfaceName, peer); err != nil {
			w.ui.Warning(fmt.Sprintf("Failed to add peer: %v", err))
			retry, err := w.ui.PromptYesNo("Failed to add peer. Would you like to retry?", true)
			if err != nil || !retry {
				break
			}
			continue
		}

		peerCount++

		// Increment suggested IP for next peer
		if strings.Contains(nextIP, "/32") {
			parts := strings.Split(nextIP, ".")
			if len(parts) == 4 {
				lastOctet := strings.Split(parts[3], "/")[0]
				var octet int
				if _, err := fmt.Sscanf(lastOctet, "%d", &octet); err != nil {
					w.ui.Warning(fmt.Sprintf("Failed to parse last octet '%s': %v", lastOctet, err))
					// Skip incrementing on parse error
				} else {
					octet++
					if octet > 254 {
						w.ui.Warning("Maximum number of peers reached (254). Cannot assign more unique IP addresses in this subnet.")
					} else {
						nextIP = fmt.Sprintf("%s.%s.%s.%d/32", parts[0], parts[1], parts[2], octet)
					}
				}
			}
		}

		w.ui.Print("")
		addMore, err := w.ui.PromptYesNo("Add another peer?", false)
		if err != nil || !addMore {
			break
		}
	}

	if peerCount > 0 {
		w.ui.Print("")
		w.ui.Successf("Added %d peer(s)", peerCount)

		// Check if service is running and offer to restart
		serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)
		active, err := w.services.IsActive(serviceName)
		if err != nil {
			w.ui.Warning(fmt.Sprintf("Could not determine service status for %s: %v", serviceName, err))
			w.ui.Infof("You may need to manually restart the service: sudo systemctl restart %s", serviceName)
		}

		if active {
			w.ui.Print("")
			w.ui.Info("The WireGuard service needs to be restarted to apply peer changes.")
			restart, err := w.ui.PromptYesNo("Restart the service now?", true)
			if err == nil && restart {
				w.ui.Info("Restarting service...")
				if err := w.services.Restart(serviceName); err != nil {
					w.ui.Warning(fmt.Sprintf("Failed to restart service: %v", err))
					w.ui.Infof("Restart manually: sudo systemctl restart %s", serviceName)
				} else {
					w.ui.Success("Service restarted successfully")
				}
			}
		} else {
			w.ui.Print("")
			w.ui.Info("The WireGuard service is not currently running.")
			w.ui.Info("To apply the peer changes, start the service manually:")
			w.ui.Infof("  sudo systemctl start wg-quick@%s", interfaceName)
		}
	}

	w.ui.Print("")
	w.ui.Info("For client configuration, provide them with:")
	w.ui.Infof("  - Server public key: %s", publicKey)
	w.ui.Info("  - Server endpoint: <your-public-ip>:51820")
	w.ui.Info("  - Client's AllowedIPs: 0.0.0.0/0 (to route all traffic) or specific subnets")

	return nil
}

const wireGuardCompletionMarker = "wireguard-setup-complete"

// Run executes the WireGuard setup step
func (w *WireGuardSetup) Run() error {
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(w.markers, wireGuardCompletionMarker, "wireguard-configured", "wireguard-skipped")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		w.ui.Info("WireGuard already configured (marker found)")
		w.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + wireGuardCompletionMarker)
		return nil
	}

	w.ui.Header("WireGuard VPN Setup")
	w.ui.Info("Configure WireGuard VPN for secure remote access...")
	w.ui.Print("")

	// Ask if they want to configure WireGuard
	w.ui.Step("WireGuard Setup")
	useWireGuard, err := w.PromptForWireGuard()
	if err != nil {
		return fmt.Errorf("failed to prompt for WireGuard: %w", err)
	}

	if !useWireGuard {
		w.ui.Info("Skipping WireGuard configuration")
		w.ui.Info("To configure WireGuard later, remove marker: ~/.local/homelab-setup/" + wireGuardCompletionMarker)
		if err := w.config.Set("WIREGUARD_ENABLED", "false"); err != nil {
			return fmt.Errorf("failed to update WireGuard configuration: %w", err)
		}
		if err := w.markers.Create(wireGuardCompletionMarker); err != nil {
			return fmt.Errorf("failed to create completion marker: %w", err)
		}
		return nil
	}

	// Check if WireGuard is installed
	w.ui.Step("Checking WireGuard Installation")
	if err := w.CheckWireGuardInstalled(); err != nil {
		return fmt.Errorf("WireGuard check failed: %w", err)
	}

	// Generate keys
	w.ui.Step("Generating Encryption Keys")
	privateKey, publicKey, err := w.GenerateKeys()
	if err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	// Prompt for configuration
	w.ui.Step("Interface Configuration")
	cfg, err := w.PromptForConfig(publicKey)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}
	cfg.PrivateKey = privateKey

	// Write configuration
	w.ui.Step("Creating Configuration File")
	if err := w.WriteConfig(cfg, privateKey); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Enable service
	w.ui.Step("Enabling WireGuard Service")
	if err := w.EnableService(cfg.InterfaceName); err != nil {
		w.ui.Warning(fmt.Sprintf("Failed to enable service: %v", err))
		// Non-critical, continue
	}

	// Add peers interactively
	w.ui.Step("Peer Configuration")
	if err := w.AddPeers(cfg.InterfaceName, publicKey, cfg.InterfaceIP); err != nil {
		w.ui.Warning(fmt.Sprintf("Failed to add peers: %v", err))
		// Non-critical, continue
	}

	// Save configuration
	w.ui.Step("Saving Configuration")
	if err := w.config.Set("WIREGUARD_ENABLED", "true"); err != nil {
		return fmt.Errorf("failed to save WireGuard enabled: %w", err)
	}

	if err := w.config.Set("WIREGUARD_INTERFACE", cfg.InterfaceName); err != nil {
		return fmt.Errorf("failed to save WireGuard interface: %w", err)
	}

	if err := w.config.Set("WIREGUARD_PUBLIC_KEY", publicKey); err != nil {
		return fmt.Errorf("failed to save WireGuard public key: %w", err)
	}

	w.ui.Print("")
	w.ui.Separator()
	w.ui.Success("✓ WireGuard configuration completed")
	w.ui.Infof("Interface: %s", cfg.InterfaceName)
	w.ui.Infof("Address: %s", cfg.InterfaceIP)
	w.ui.Infof("Port: %s", cfg.ListenPort)

	// Create completion marker
	if err := w.markers.Create(wireGuardCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
