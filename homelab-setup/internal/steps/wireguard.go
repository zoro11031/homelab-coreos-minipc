package steps

import (
	"fmt"
	"os/exec"
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

	configPath := fmt.Sprintf("/etc/wireguard/%s.conf", cfg.InterfaceName)

	w.ui.Print("")
	w.ui.Info("Configuration file content:")
	w.ui.Print(configContent)
	w.ui.Print("")

	// TODO: Implement WriteFile in FileSystem
	w.ui.Warning("Automatic config file creation not yet implemented")
	w.ui.Infof("Please create %s with the content shown above", configPath)
	w.ui.Info("Commands:")
	w.ui.Infof("  echo '<content>' | sudo tee %s", configPath)
	w.ui.Infof("  sudo chmod 600 %s", configPath)
	w.ui.Print("")

	created, err := w.ui.PromptYesNo("Have you created the config file?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if !created {
		w.ui.Warning("Config file not created")
		w.ui.Info("You'll need to create it manually before starting the service")
		return nil
	}

	return nil
}

// EnableService enables and starts the WireGuard service
func (w *WireGuardSetup) EnableService(interfaceName string) error {
	serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)

	w.ui.Infof("Enabling %s...", serviceName)

	// Check if service exists
	exists, err := w.services.ServiceExists(serviceName)
	if err != nil {
		return fmt.Errorf("failed to check service: %w", err)
	}

	if !exists {
		w.ui.Warning(fmt.Sprintf("Service %s not found", serviceName))
		w.ui.Info("This is normal - wg-quick services are dynamically created")
	}

	// Enable service
	w.ui.Info("To enable and start the service:")
	w.ui.Infof("  sudo systemctl enable %s", serviceName)
	w.ui.Infof("  sudo systemctl start %s", serviceName)
	w.ui.Print("")

	enabled, err := w.ui.PromptYesNo("Have you enabled and started the service?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if enabled {
		w.ui.Success("WireGuard service enabled and started")

		// Display status instructions
		w.ui.Print("")
		w.ui.Info("To check WireGuard status:")
		w.ui.Infof("  sudo systemctl status %s", serviceName)
		w.ui.Infof("  sudo wg show %s", interfaceName)
	} else {
		w.ui.Warning("WireGuard service not started")
		w.ui.Info("Start it later with the commands shown above")
	}

	return nil
}

// DisplayPeerInstructions displays instructions for adding peers
func (w *WireGuardSetup) DisplayPeerInstructions(interfaceName, publicKey string) {
	w.ui.Print("")
	w.ui.Info("Adding WireGuard Peers:")
	w.ui.Separator()
	w.ui.Print("")

	w.ui.Info("Your server public key:")
	w.ui.Printf("  %s", publicKey)
	w.ui.Print("")

	w.ui.Info("To add a peer, edit the config file:")
	w.ui.Infof("  sudo nano /etc/wireguard/%s.conf", interfaceName)
	w.ui.Print("")

	w.ui.Info("Add a peer section:")
	w.ui.Print("  [Peer]")
	w.ui.Print("  PublicKey = <peer-public-key>")
	w.ui.Print("  AllowedIPs = 10.253.0.2/32")
	w.ui.Print("  # Endpoint = <peer-ip>:51820  # If peer is client")
	w.ui.Print("")

	w.ui.Info("After editing, restart the service:")
	w.ui.Infof("  sudo systemctl restart wg-quick@%s", interfaceName)
	w.ui.Print("")

	w.ui.Info("For client configuration, provide them with:")
	w.ui.Infof("  - Server public key: %s", publicKey)
	w.ui.Info("  - Server endpoint: <your-public-ip>:51820")
	w.ui.Info("  - Allowed IPs for the client")
	w.ui.Print("")
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

	// Display peer instructions
	w.ui.Step("Peer Configuration")
	w.DisplayPeerInstructions(cfg.InterfaceName, publicKey)

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
