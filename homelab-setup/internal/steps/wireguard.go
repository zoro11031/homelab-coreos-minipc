// Package steps implements the setup workflow steps for homelab configuration.
// Each step is a function that performs a specific setup task (user creation,
// directory setup, service deployment, etc.) and creates a completion marker
// to track progress. Steps can be run individually or sequentially as part of
// the complete setup workflow.
package steps

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

const wireGuardCompletionMarker = "wireguard-setup-complete"

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

// WireGuardKeyGenerator describes key generation/derivation helpers so the
// workflow can be unit-tested without shelling out to the wg binary.
type WireGuardKeyGenerator interface {
	GenerateKeyPair() (privateKey, publicKey string, err error)
	GeneratePresharedKey() (string, error)
	DerivePublicKey(privateKey string) (string, error)
}

// CommandKeyGenerator implements WireGuardKeyGenerator by calling wg commands.
type CommandKeyGenerator struct{}

// GenerateKeyPair produces a WireGuard key pair using "wg genkey".
func (kg CommandKeyGenerator) GenerateKeyPair() (string, string, error) {
	privCmd := exec.Command("wg", "genkey")
	privOutput, err := privCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}
	privateKey := strings.TrimSpace(string(privOutput))

	pub, err := kg.DerivePublicKey(privateKey)
	if err != nil {
		return "", "", err
	}
	return privateKey, pub, nil
}

// GeneratePresharedKey runs "wg genpsk".
func (CommandKeyGenerator) GeneratePresharedKey() (string, error) {
	cmd := exec.Command("wg", "genpsk")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate preshared key: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// DerivePublicKey runs "wg pubkey" using the provided private key.
func (CommandKeyGenerator) DerivePublicKey(privateKey string) (string, error) {
	pubCmd := exec.Command("wg", "pubkey")
	pubCmd.Stdin = strings.NewReader(privateKey)
	pubOutput, err := pubCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to derive public key: %w", err)
	}
	return strings.TrimSpace(string(pubOutput)), nil
}

// sanitizePeerName removes or replaces characters that could break the WireGuard config format
// or be used to inject additional configuration sections
func sanitizePeerName(name string) string {
	// Remove newlines, carriage returns, and other control characters
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\t", " ")

	// Remove brackets that could be used to inject sections
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")

	// Remove hash/pound sign to prevent comment injection
	name = strings.ReplaceAll(name, "#", "")

	// Trim whitespace
	name = strings.TrimSpace(name)

	return name
}

// sanitizeConfigValue removes characters that could break the WireGuard config format
// or be used to inject additional configuration directives. This is critical for values
// like PublicKey, AllowedIPs, and Endpoint that are written directly to the config file.
//
// Security: Uses deny-listing to prevent config injection attacks by removing:
// - Newlines/control chars that could split into multiple config lines
// - Section markers [] that could inject new config sections
// - Comment markers # that could hide malicious directives
// - Shell metacharacters =;|&`$\ that could be exploited in PostUp/PreDown scripts
func sanitizeConfigValue(value string) string {
	// Remove newlines and carriage returns to prevent config injection
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\t", " ")

	// Remove brackets that could be used to inject sections
	value = strings.ReplaceAll(value, "[", "")
	value = strings.ReplaceAll(value, "]", "")

	// Remove hash/pound sign to prevent comment injection
	value = strings.ReplaceAll(value, "#", "")

	// Remove equals sign to prevent key=value injection
	value = strings.ReplaceAll(value, "=", "")

	// Remove shell metacharacters that could be exploited in PostUp/PreDown
	value = strings.ReplaceAll(value, ";", "")  // Command separator
	value = strings.ReplaceAll(value, "|", "")  // Pipe operator
	value = strings.ReplaceAll(value, "&", "")  // Background/AND operator
	value = strings.ReplaceAll(value, "`", "")  // Command substitution
	value = strings.ReplaceAll(value, "$", "")  // Variable expansion
	value = strings.ReplaceAll(value, "\\", "") // Escape sequences

	// Trim whitespace
	value = strings.TrimSpace(value)

	return value
}

// configDir returns the WireGuard configuration directory
func configDir(cfg *config.Config) string {
	return cfg.GetOrDefault("WIREGUARD_CONFIG_DIR", "/etc/wireguard")
}

// incrementIP increments the last octet of an IP address in CIDR notation.
// For example, "10.253.0.2/32" becomes "10.253.0.3/32".
// Returns an error if the IP format is invalid or the last octet would exceed 254.
func incrementIP(ip string) (string, error) {
	if !strings.Contains(ip, "/") {
		return "", fmt.Errorf("IP address must be in CIDR notation (e.g., 10.0.0.1/32)")
	}

	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return "", fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Extract the last octet and CIDR suffix
	lastPart := parts[3]
	lastOctetStr, cidrSuffix, found := strings.Cut(lastPart, "/")
	if !found {
		return "", fmt.Errorf("invalid IP address format: missing CIDR suffix in '%s'", ip)
	}

	var octet int
	if _, err := fmt.Sscanf(lastOctetStr, "%d", &octet); err != nil {
		return "", fmt.Errorf("failed to parse last octet '%s': %w", lastOctetStr, err)
	}

	octet++
	if octet > 254 {
		return "", fmt.Errorf("cannot increment IP: last octet would exceed 254")
	}

	return fmt.Sprintf("%s.%s.%s.%d/%s", parts[0], parts[1], parts[2], octet, cidrSuffix), nil
}

// PromptForWireGuard asks if the user wants to configure WireGuard
func promptForWireGuard(cfg *config.Config, ui *ui.UI) (bool, error) {
	ui.Info("WireGuard is a modern, fast VPN protocol")
	ui.Info("It can be used to:")
	ui.Info("  - Securely connect to your homelab from anywhere")
	ui.Info("  - Create encrypted tunnels to a VPS for external access")
	ui.Info("  - Build a mesh network between devices")
	ui.Print("")

	useWireGuard, err := ui.PromptYesNo("Do you want to configure WireGuard?", false)
	if err != nil {
		return false, fmt.Errorf("failed to prompt for WireGuard: %w", err)
	}

	return useWireGuard, nil
}

// CheckWireGuardInstalled checks if WireGuard tools are installed
func checkWireGuardInstalled(cfg *config.Config, ui *ui.UI) error {
	ui.Info("Checking for WireGuard tools...")

	installed, err := system.IsPackageInstalled("wireguard-tools")
	if err != nil {
		return fmt.Errorf("failed to check wireguard-tools: %w", err)
	}

	if !installed {
		ui.Warning("wireguard-tools not installed")
		ui.Info("To install:")
		ui.Info("  sudo rpm-ostree install wireguard-tools")
		ui.Info("  sudo systemctl reboot")
		return fmt.Errorf("wireguard-tools not installed")
	}

	ui.Success("wireguard-tools is installed")

	// Check for wg command
	if !system.CommandExists("wg") {
		ui.Warning("wg command not found")
		return fmt.Errorf("wg command not available")
	}

	ui.Success("wg command is available")
	return nil
}

// PromptForConfig prompts for WireGuard configuration
func promptForConfig(cfg *config.Config, ui *ui.UI, publicKey string) (*WireGuardConfig, error) {
	ui.Print("")
	ui.Info("WireGuard Interface Configuration:")
	ui.Print("")

	wgCfg := &WireGuardConfig{
		PublicKey: publicKey,
	}

	// Prompt for interface name
	interfaceName, err := ui.PromptInput("Interface name", "wg0")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for interface name: %w", err)
	}
	wgCfg.InterfaceName = interfaceName

	// Prompt for interface IP
	interfaceIP, err := ui.PromptInput("Interface IP address (CIDR notation)", "10.253.0.1/24")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for interface IP: %w", err)
	}
	// Validate CIDR notation
	// Note: CIDR validation is intentionally inlined here rather than using a shared
	// validator function. This trades code reuse for simplicity. If validation logic
	// needs to change (e.g., adding IPv6 support), also update the same validation
	// in promptForPeer() below (line ~510).
	if interfaceIP == "" {
		return nil, fmt.Errorf("interface IP cannot be empty")
	}
	if ip, network, err := net.ParseCIDR(interfaceIP); err != nil || ip.To4() == nil || network == nil {
		return nil, fmt.Errorf("invalid IPv4 CIDR notation: %s", interfaceIP)
	}
	wgCfg.InterfaceIP = interfaceIP

	// Prompt for listen port
	listenPort, err := ui.PromptInput("Listen port", "51820")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for listen port: %w", err)
	}

	// Validate port (1-65535)
	if p, err := strconv.Atoi(listenPort); err != nil || p < 1 || p > 65535 {
		return nil, fmt.Errorf("invalid port number (must be 1-65535): %s", listenPort)
	}
	wgCfg.ListenPort = listenPort

	return wgCfg, nil
}

// WriteConfig writes the WireGuard configuration file
func writeConfig(cfgData *config.Config, ui *ui.UI, cfg *WireGuardConfig, privateKey string) error {
	ui.Infof("Writing WireGuard configuration for %s...", cfg.InterfaceName)

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

	ui.Print("")
	ui.Info("Configuration file content:")
	ui.Print(configContent)
	ui.Print("")

	configDirPath := configDir(cfgData)
	configPath := filepath.Join(configDirPath, fmt.Sprintf("%s.conf", cfg.InterfaceName))

	if err := system.EnsureDirectory(configDirPath, "root:root", 0750); err != nil {
		return fmt.Errorf("failed to ensure WireGuard directory %s: %w", configDirPath, err)
	}

	if err := system.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write WireGuard config: %w", err)
	}

	exists, err := system.FileExists(configPath)
	if err != nil {
		return fmt.Errorf("failed to verify config file: %w", err)
	}
	if !exists {
		return fmt.Errorf("WireGuard config %s was not created", configPath)
	}

	perms, err := system.GetPermissions(configPath)
	if err != nil {
		return fmt.Errorf("failed to check permissions on %s: %w", configPath, err)
	}
	if perms.Perm() != 0600 {
		return fmt.Errorf("WireGuard config %s must have 0600 permissions, found %o", configPath, perms.Perm())
	}

	ui.Successf("Configuration file created at %s", configPath)
	ui.Info("Review the file to add peers as needed")

	return nil
}

// EnableService enables and starts the WireGuard service
func enableService(cfg *config.Config, ui *ui.UI, interfaceName string) error {
	serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)

	ui.Print("")
	ui.Info("The WireGuard service needs to be enabled and started.")
	ui.Print("")

	autoEnable, err := ui.PromptYesNo("Do you want to enable and start the service now?", true)
	if err != nil {
		return fmt.Errorf("failed to prompt: %w", err)
	}

	if !autoEnable {
		ui.Print("")
		ui.Info("To enable and start the service manually:")
		ui.Infof("  sudo systemctl enable %s", serviceName)
		ui.Infof("  sudo systemctl start %s", serviceName)
		ui.Print("")
		ui.Warning("WireGuard service not started")
		return nil
	}

	ui.Print("")
	ui.Infof("Enabling %s...", serviceName)

	// Enable service
	if err := system.EnableService(serviceName); err != nil {
		ui.Warning(fmt.Sprintf("Failed to enable service: %v", err))
		ui.Info("You may need to run manually:")
		ui.Infof("  sudo systemctl enable %s", serviceName)
		return fmt.Errorf("failed to enable service: %w", err)
	}
	ui.Success("Service enabled")

	// Start service
	ui.Infof("Starting %s...", serviceName)
	if err := system.StartService(serviceName); err != nil {
		ui.Warning(fmt.Sprintf("Failed to start service: %v", err))
		ui.Info("You may need to run manually:")
		ui.Infof("  sudo systemctl start %s", serviceName)
		return fmt.Errorf("failed to start service: %w", err)
	}
	ui.Success("Service started")

	// Check if service is actually running
	active, err := system.IsServiceActive(serviceName)
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not verify service status: %v", err))
	} else if active {
		ui.Success("WireGuard service is running")
	} else {
		ui.Warning("Service may not be running correctly")
	}

	// Display status instructions
	ui.Print("")
	ui.Info("To check WireGuard status:")
	ui.Infof("  sudo systemctl status %s", serviceName)
	ui.Infof("  sudo wg show %s", interfaceName)

	return nil
}

// PromptForPeer prompts for peer configuration.
//
// Validation performed:
//   - Prompts for peer name, public key, allowed IPs, and endpoint.
//   - Returns an error if any prompt fails (e.g., EOF, user abort).
//   - Returns an error if the public key is empty.
//
// Note:
//   - Peer name sanitization is performed in the caller (AddPeerToConfig).
//
// Return value:
//   - Returns a WireGuardPeer and nil error on success.
//   - Returns nil and an error for non-recoverable input errors (such as EOF).
func promptForPeer(cfg *config.Config, ui *ui.UI, nextIP string) (*WireGuardPeer, error) {
	peer := &WireGuardPeer{}

	ui.Print("")

	// Prompt for peer name
	name, err := ui.PromptInput("Peer name (e.g., 'laptop', 'phone', 'vps')", "")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for peer name: %w", err)
	}
	if name == "" {
		name = "unnamed-peer"
	}
	// Sanitize the peer name immediately to prevent config injection
	peer.Name = sanitizePeerName(name)

	// Prompt for public key with validation loop
	for {
		publicKey, err := ui.PromptInput("Peer public key", "")
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for public key: %w", err)
		}
		if publicKey == "" {
			ui.Error("Public key is required")
			continue
		}
		// Validate WireGuard key format (44 chars, base64, ends with '=')
		if len(publicKey) != 44 || !strings.HasSuffix(publicKey, "=") {
			ui.Error("Invalid WireGuard key format")
			ui.Info("WireGuard keys are 44 characters, base64-encoded, ending with '='")
			continue
		}
		// Validate it's actually valid base64 by attempting to decode
		decoded, err := base64.StdEncoding.DecodeString(publicKey)
		if err != nil {
			ui.Error("Invalid WireGuard key: not valid base64 encoding")
			ui.Info("WireGuard keys must be properly base64-encoded")
			continue
		}
		// WireGuard keys should decode to exactly 32 bytes (Curve25519 public key)
		if len(decoded) != 32 {
			ui.Error("Invalid WireGuard key: incorrect key length")
			ui.Info("WireGuard keys must be 32 bytes (256 bits) when decoded")
			continue
		}
		peer.PublicKey = publicKey
		break
	}

	// Prompt for allowed IPs
	for {
		allowedIPs, err := ui.PromptInput("Allowed IPs for this peer", nextIP)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for allowed IPs: %w", err)
		}
		// Validate CIDR notation (matches validation in promptForConfig above)
		if allowedIPs == "" {
			ui.Error("Allowed IPs cannot be empty")
			continue
		}
		if ip, network, err := net.ParseCIDR(allowedIPs); err != nil || ip.To4() == nil || network == nil {
			ui.Error("Invalid CIDR notation. Please enter a valid IPv4 CIDR (e.g., '10.253.0.2/32').")
			continue
		}
		peer.AllowedIPs = allowedIPs
		break
	}

	// Prompt for endpoint (optional)
	ui.Info("Endpoint is optional - leave empty for road warrior clients")
	endpoint, err := ui.PromptInput("Endpoint (e.g., 'server.example.com:51820')", "")
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for endpoint: %w", err)
	}
	peer.Endpoint = endpoint

	return peer, nil
}

// AddPeerToConfig appends a peer configuration to the WireGuard config file.
//
// Security considerations:
// - Uses `sudo cat` to read the config file to handle permissions; this requires the user to have passwordless sudo access for `cat`.
// - All peer fields (Name, PublicKey, AllowedIPs, Endpoint) are sanitized to prevent config injection attacks.
// - Sanitization removes newlines, brackets, and hash characters that could be used to inject malicious configuration directives.
// - The function appends the new peer configuration to the existing config file rather than replacing the entire file, preserving existing peers.
func addPeerToConfig(cfg *config.Config, ui *ui.UI, interfaceName string, peer *WireGuardPeer) error {
	configPath := filepath.Join(configDir(cfg), fmt.Sprintf("%s.conf", interfaceName))

	// Read current config (using sudo cat to handle permissions)
	cmd := exec.Command("sudo", "-n", "cat", configPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Build peer section with sanitized values to prevent config injection
	sanitizedName := sanitizePeerName(peer.Name)
	sanitizedPublicKey := sanitizeConfigValue(peer.PublicKey)
	sanitizedAllowedIPs := sanitizeConfigValue(peer.AllowedIPs)
	peerSection := fmt.Sprintf("\n# Peer: %s\n[Peer]\nPublicKey = %s\nAllowedIPs = %s\n",
		sanitizedName, sanitizedPublicKey, sanitizedAllowedIPs)

	if peer.Endpoint != "" {
		sanitizedEndpoint := sanitizeConfigValue(peer.Endpoint)
		peerSection += fmt.Sprintf("Endpoint = %s\n", sanitizedEndpoint)
	}

	// Append peer to config
	newContent := string(output) + peerSection

	// Write updated config
	if err := system.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	ui.Successf("Peer '%s' added to configuration", peer.Name)
	return nil
}

// AddPeers interactively adds WireGuard peers to the WireGuard configuration.
//
// Workflow:
//   - Prompts the user to decide whether to add peers now.
//   - If the user declines, the function returns early with nil error and provides instructions
//     for manual peer addition and service restart.
//   - If the user agrees, enters an interactive loop to add one or more peers.
//   - For each peer, automatically suggests the next available IP address.
//   - After each peer is added, appends the peer configuration to the WireGuard config file.
//   - After all peers are added, instructs the user to restart the WireGuard service for changes to take effect.
//
// Returns nil error if the user declines to add peers.
func addPeers(cfg *config.Config, ui *ui.UI, keygen WireGuardKeyGenerator, interfaceName, publicKey, interfaceIP string) error {
	ui.Print("")
	ui.Info("WireGuard Peer Configuration:")
	ui.Separator()
	ui.Print("")

	ui.Info("Your server public key:")
	ui.Printf("  %s", publicKey)
	ui.Print("")

	addPeers, err := ui.PromptYesNo("Do you want to add peers now?", false)
	if err != nil {
		return fmt.Errorf("failed to prompt for adding peers: %w", err)
	}

	if !addPeers {
		ui.Print("")
		ui.Info("You can add peers later by editing:")
		ui.Infof("  %s", filepath.Join(configDir(cfg), fmt.Sprintf("%s.conf", interfaceName)))
		ui.Print("")
		ui.Info("After editing, restart the service:")
		ui.Infof("  sudo systemctl restart wg-quick@%s", interfaceName)
		return nil
	}

	// Parse the interface IP to suggest next IP for peers
	// For example, if server is 10.253.0.1/24, suggest 10.253.0.2/32 for first peer
	nextIP := "10.253.0.2/32"
	if strings.Contains(interfaceIP, "/") {
		// Start with the interface IP and increment to get the first peer IP
		// Convert /24 (or other CIDR) to /32 for peer
		parts := strings.Split(interfaceIP, "/")
		if len(parts) >= 2 {
			baseIP := parts[0] + "/32"
			// Increment from server IP (e.g., 10.253.0.1/32 → 10.253.0.2/32)
			if incremented, err := incrementIP(baseIP); err == nil {
				nextIP = incremented
			}
		} else {
			// Malformed interfaceIP, fallback to default suggestion
			nextIP = "10.253.0.2/32"
		}
	}

	peerCount := 0
	for {
		ui.Print("")
		ui.Infof("Adding peer #%d", peerCount+1)

		peer, err := promptForPeer(cfg, ui, nextIP)
		if err != nil {
			// Check if the error is non-recoverable (e.g., EOF, input stream closed)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe) {
				ui.Error(fmt.Sprintf("Input stream closed: %v", err))
				break
			}
			// For recoverable errors (e.g., validation errors), show warning and retry
			ui.Warning(fmt.Sprintf("Failed to get peer configuration: %v", err))
			continue
		}

		if err := addPeerToConfig(cfg, ui, interfaceName, peer); err != nil {
			ui.Warning(fmt.Sprintf("Failed to add peer: %v", err))
			continue
		}

		peerCount++

		// Increment suggested IP for next peer
		incrementedIP, err := incrementIP(nextIP)
		if err == nil {
			nextIP = incrementedIP
		} else {
			ui.Warning(fmt.Sprintf("Failed to increment IP: %v", err))
			// nextIP remains unchanged; last successful IP will be reused
		}

		ui.Print("")
		addMore, err := ui.PromptYesNo("Add another peer?", false)
		if err != nil || !addMore {
			break
		}
	}

	if peerCount > 0 {
		ui.Print("")
		ui.Successf("Added %d peer(s)", peerCount)

		// Check if service is running and offer to restart
		serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)
		active, _ := system.IsServiceActive(serviceName)

		if active {
			ui.Print("")
			ui.Info("The WireGuard service needs to be restarted to apply peer changes.")
			restart, err := ui.PromptYesNo("Restart the service now?", true)
			if err == nil && restart {
				ui.Info("Restarting service...")
				if err := system.RestartService(serviceName); err != nil {
					ui.Warning(fmt.Sprintf("Failed to restart service: %v", err))
					ui.Infof("Restart manually: sudo systemctl restart %s", serviceName)
				} else {
					ui.Success("Service restarted successfully")
				}
			}
		}
	}

	ui.Print("")
	ui.Info("For client configuration, provide them with:")
	ui.Infof("  - Server public key: %s", publicKey)
	ui.Info("  - Server endpoint: <your-public-ip>:51820")
	ui.Info("  - Client's AllowedIPs: 0.0.0.0/0 (to route all traffic) or specific subnets")

	return nil
}

// RunWireGuardSetup executes the WireGuard setup step
func RunWireGuardSetup(cfg *config.Config, ui *ui.UI) error {
	// Create default keygen
	keygen := CommandKeyGenerator{}
	// Check if already completed (and migrate legacy markers)
	completed, err := ensureCanonicalMarker(cfg, wireGuardCompletionMarker, "wireguard-configured", "wireguard-skipped")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if completed {
		ui.Info("WireGuard already configured (marker found)")
		ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + wireGuardCompletionMarker)
		return nil
	}

	ui.Header("WireGuard VPN Setup")
	ui.Info("Configure WireGuard VPN for secure remote access...")
	ui.Print("")

	// Ask if they want to configure WireGuard
	ui.Step("WireGuard Setup")
	useWireGuard, err := promptForWireGuard(cfg, ui)
	if err != nil {
		return fmt.Errorf("failed to prompt for WireGuard: %w", err)
	}

	if !useWireGuard {
		ui.Info("Skipping WireGuard configuration")
		ui.Info("To configure WireGuard later, remove marker: ~/.local/homelab-setup/" + wireGuardCompletionMarker)
		if err := cfg.Set("WIREGUARD_ENABLED", "false"); err != nil {
			return fmt.Errorf("failed to update WireGuard configuration: %w", err)
		}
		if err := cfg.MarkComplete(wireGuardCompletionMarker); err != nil {
			return fmt.Errorf("failed to create completion marker: %w", err)
		}
		return nil
	}

	// Check if WireGuard is installed
	ui.Step("Checking WireGuard Installation")
	if err := checkWireGuardInstalled(cfg, ui); err != nil {
		return fmt.Errorf("WireGuard check failed: %w", err)
	}

	// Generate keys
	ui.Step("Generating Encryption Keys")
	ui.Info("Generating WireGuard keys...")
	privateKey, publicKey, err := keygen.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	ui.Success("Keys generated successfully")
	ui.Print("")
	ui.Info("Public key (share with peers):")
	ui.Printf("  %s", publicKey)
	ui.Print("")
	ui.Warning("Private key (keep secret!):")
	ui.Printf("  %s", privateKey)
	ui.Print("")

	// Prompt for configuration
	ui.Step("Interface Configuration")
	wgCfg, err := promptForConfig(cfg, ui, publicKey)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}
	wgCfg.PrivateKey = privateKey

	// Write configuration
	ui.Step("Creating Configuration File")
	if err := writeConfig(cfg, ui, wgCfg, privateKey); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Enable service
	ui.Step("Enabling WireGuard Service")
	if err := enableService(cfg, ui, wgCfg.InterfaceName); err != nil {
		ui.Warning(fmt.Sprintf("Failed to enable service: %v", err))
		// Non-critical, continue
	}

	// Add peers interactively
	ui.Step("Peer Configuration")
	if err := addPeers(cfg, ui, keygen, wgCfg.InterfaceName, publicKey, wgCfg.InterfaceIP); err != nil {
		ui.Warning(fmt.Sprintf("Failed to add peers: %v", err))
		// Non-critical, continue
	}

	// Save configuration
	ui.Step("Saving Configuration")
	if err := cfg.Set("WIREGUARD_ENABLED", "true"); err != nil {
		return fmt.Errorf("failed to save WireGuard enabled: %w", err)
	}

	if err := cfg.Set("WIREGUARD_INTERFACE", wgCfg.InterfaceName); err != nil {
		return fmt.Errorf("failed to save WireGuard interface: %w", err)
	}

	if err := cfg.Set("WIREGUARD_PUBLIC_KEY", publicKey); err != nil {
		return fmt.Errorf("failed to save WireGuard public key: %w", err)
	}

	ui.Print("")
	ui.Separator()
	ui.Success("✓ WireGuard configuration completed")
	ui.Infof("Interface: %s", wgCfg.InterfaceName)
	ui.Infof("Address: %s", wgCfg.InterfaceIP)
	ui.Infof("Port: %s", wgCfg.ListenPort)

	// Create completion marker
	if err := cfg.MarkComplete(wireGuardCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
