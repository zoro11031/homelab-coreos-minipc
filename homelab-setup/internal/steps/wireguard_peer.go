package steps

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
)

// WireGuardPeerWorkflowOptions allows the CLI or tests to pre-seed inputs.
type WireGuardPeerWorkflowOptions struct {
	InterfaceName              string
	PeerName                   string
	Endpoint                   string
	DNS                        string
	ClientAllowedIPs           string
	RouteAll                   *bool
	OutputDir                  string
	PersistentKeepaliveSeconds *int
	GeneratePresharedKey       *bool
	ProvidedPresharedKey       string
	NonInteractive             bool
	SkipQRCode                 bool
	SkipServiceRestart         bool
}

func defaultPeerExportDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/var/home/core"
	}
	return filepath.Join(home, "setup", "export", "wireguard-peers")
}

type parsedWireGuardConfig struct {
	Interface map[string]string
	Peers     []wireGuardPeerBlock
}

type wireGuardPeerBlock struct {
	Comment string
	Values  map[string]string
}

func parseWireGuardConfig(content string) *parsedWireGuardConfig {
	cfg := &parsedWireGuardConfig{Interface: make(map[string]string)}
	lines := strings.Split(content, "\n")
	var current map[string]string
	var pendingPeerComment string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# Peer:") {
			pendingPeerComment = strings.TrimSpace(strings.TrimPrefix(trimmed, "# Peer:"))
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section := strings.ToLower(strings.Trim(trimmed, "[]"))
			switch section {
			case "interface":
				current = cfg.Interface
			case "peer":
				block := wireGuardPeerBlock{Values: make(map[string]string), Comment: pendingPeerComment}
				cfg.Peers = append(cfg.Peers, block)
				current = cfg.Peers[len(cfg.Peers)-1].Values
				pendingPeerComment = ""
			default:
				current = nil
			}
			continue
		}
		if current == nil {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		current[key] = value
	}
	return cfg
}

func firstInterfaceAddress(cfg *parsedWireGuardConfig) (string, error) {
	addr, ok := cfg.Interface["Address"]
	if !ok || addr == "" {
		return "", fmt.Errorf("interface section missing Address")
	}
	parts := strings.Split(addr, ",")
	primary := strings.TrimSpace(parts[0])
	if _, _, err := net.ParseCIDR(primary); err != nil {
		return "", fmt.Errorf("failed to parse interface address %s: %w", primary, err)
	}
	return primary, nil
}

func collectUsedPeerIPs(cfg *parsedWireGuardConfig) map[string]struct{} {
	used := make(map[string]struct{})
	for _, peer := range cfg.Peers {
		allowed, ok := lookupPeerValue(peer.Values, "AllowedIPs")
		if !ok {
			continue
		}
		entries := strings.Split(allowed, ",")
		for _, entry := range entries {
			canonical, ipStr, ok := normalizeAllowedIPToken(entry)
			if !ok {
				continue
			}
			if canonical != "" {
				used[canonical] = struct{}{}
			}
			if ipStr != "" {
				used[ipStr] = struct{}{}
			}
		}
	}
	return used
}

func lookupPeerValue(values map[string]string, target string) (string, bool) {
	for key, value := range values {
		if strings.EqualFold(key, target) {
			return value, true
		}
	}
	return "", false
}

func normalizeAllowedIPToken(entry string) (string, string, bool) {
	cleaned := strings.TrimSpace(entry)
	if idx := strings.IndexAny(cleaned, "#;"); idx >= 0 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}
	if cleaned == "" {
		return "", "", false
	}
	if strings.Contains(cleaned, "/") {
		ip, network, err := net.ParseCIDR(cleaned)
		if err != nil {
			return "", "", false
		}
		ones, _ := network.Mask.Size()
		canonical := fmt.Sprintf("%s/%d", network.IP.String(), ones)
		return canonical, ip.String(), true
	}
	ip := net.ParseIP(cleaned)
	if ip == nil {
		return "", "", false
	}
	canonical := ip.String()
	if ip.To4() != nil {
		canonical = fmt.Sprintf("%s/32", ip.String())
	}
	return canonical, ip.String(), true
}

func deriveNetworkCIDR(address string) (string, *net.IPNet, net.IP, error) {
	ip, network, err := net.ParseCIDR(address)
	if err != nil {
		return "", nil, nil, fmt.Errorf("invalid CIDR %s: %w", address, err)
	}
	ones, _ := network.Mask.Size()
	cidr := fmt.Sprintf("%s/%d", network.IP.String(), ones)
	return cidr, network, ip, nil
}

func nextPeerAddress(interfaceCIDR string, used map[string]struct{}) (string, error) {
	_, network, serverIP, err := deriveNetworkCIDR(interfaceCIDR)
	if err != nil {
		return "", err
	}
	ones, bits := network.Mask.Size()
	if bits != 32 {
		return "", fmt.Errorf("only IPv4 addresses are supported for auto-allocation")
	}
	total := 1 << uint(32-ones)
	current := make(net.IP, len(network.IP))
	copy(current, network.IP)
	// skip network address
	incrementIPBytes(current)
	broadcast := make(net.IP, len(network.IP))
	copy(broadcast, network.IP)
	mask := network.Mask
	for i := range broadcast {
		broadcast[i] = network.IP[i] | ^mask[i]
	}
	usedMap := make(map[string]struct{}, len(used)+2)
	for k := range used {
		usedMap[k] = struct{}{}
	}
	usedMap[fmt.Sprintf("%s/32", serverIP.String())] = struct{}{}
	usedMap[serverIP.String()] = struct{}{}
	for i := 0; i < total; i++ {
		if !network.Contains(current) {
			break
		}
		if current.Equal(network.IP) || current.Equal(broadcast) {
			incrementIPBytes(current)
			continue
		}
		candidate := fmt.Sprintf("%s/32", current.String())
		if _, exists := usedMap[candidate]; !exists {
			return candidate, nil
		}
		incrementIPBytes(current)
	}
	return "", fmt.Errorf("no available IPs remaining in %s", interfaceCIDR)
}

func incrementIPBytes(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

func safePeerFilename(name string) string {
	sanitized := strings.ToLower(sanitizePeerName(name))
	if sanitized == "" {
		sanitized = "peer"
	}
	builder := strings.Builder{}
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		if r == ' ' {
			builder.WriteRune('-')
		}
	}
	result := builder.String()
	if result == "" {
		result = "peer"
	}
	return result
}

func (w *WireGuardSetup) AddPeerWorkflow(opts *WireGuardPeerWorkflowOptions) error {
	if opts == nil {
		opts = &WireGuardPeerWorkflowOptions{}
	}
	interfaceName := strings.TrimSpace(opts.InterfaceName)
	if interfaceName == "" {
		interfaceName = w.config.GetOrDefault("WIREGUARD_INTERFACE", "wg0")
	}
	if interfaceName == "" {
		if opts.NonInteractive {
			return fmt.Errorf("interface name is required in non-interactive mode")
		}
		input, err := w.ui.PromptInput("WireGuard interface", "wg0")
		if err != nil {
			return err
		}
		interfaceName = input
	}

	configPath := filepath.Join(w.configDir(), fmt.Sprintf("%s.conf", interfaceName))
	exists, err := system.FileExists(configPath)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("WireGuard config %s does not exist", configPath)
	}

	rawConfig, err := system.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", configPath, err)
	}
	parsed := parseWireGuardConfig(string(rawConfig))
	interfaceAddress, err := firstInterfaceAddress(parsed)
	if err != nil {
		return err
	}

	usedIPs := collectUsedPeerIPs(parsed)
	nextIP, err := nextPeerAddress(interfaceAddress, usedIPs)
	if err != nil {
		return err
	}

	networkCIDR, _, _, err := deriveNetworkCIDR(interfaceAddress)
	if err != nil {
		return err
	}

	peerName := strings.TrimSpace(opts.PeerName)
	if peerName == "" {
		defaultName := fmt.Sprintf("peer-%d", time.Now().Unix())
		if opts.NonInteractive {
			peerName = defaultName
		} else {
			peerName, err = w.ui.PromptInput("Peer name", defaultName)
			if err != nil {
				return err
			}
		}
	}
	peerName = sanitizePeerName(peerName)

	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = w.config.GetOrDefault("WIREGUARD_ENDPOINT", "")
	}
	if endpoint == "" {
		if opts.NonInteractive {
			return fmt.Errorf("endpoint is required in non-interactive mode")
		}
		endpoint, err = w.ui.PromptInput("Server endpoint (host:port)", "")
		if err != nil {
			return err
		}
	}
	if endpoint != "" {
		if err := w.config.Set("WIREGUARD_ENDPOINT", endpoint); err != nil {
			w.ui.Warningf("failed to persist endpoint: %v", err)
		}
	}

	dns := strings.TrimSpace(opts.DNS)
	if dns == "" {
		dns = w.config.GetOrDefault("WIREGUARD_PEER_DNS", "")
	}
	if dns == "" && !opts.NonInteractive {
		dns, err = w.ui.PromptInput("Client DNS server (optional)", "")
		if err != nil {
			return err
		}
	}
	if dns != "" {
		if err := w.config.Set("WIREGUARD_PEER_DNS", dns); err != nil {
			w.ui.Warningf("failed to persist DNS: %v", err)
		}
	}

	// Validate that ClientAllowedIPs and RouteAll are not both set
	if opts.ClientAllowedIPs != "" && opts.RouteAll != nil {
		return fmt.Errorf("cannot specify both ClientAllowedIPs and RouteAll options - they are mutually exclusive")
	}

	var routeAll bool
	if opts.RouteAll != nil {
		routeAll = *opts.RouteAll
	} else if opts.ClientAllowedIPs == "" {
		if opts.NonInteractive {
			routeAll = true
		} else {
			routeAll, err = w.ui.PromptYesNo("Route all client traffic through the VPN?", true)
			if err != nil {
				return err
			}
		}
	}

	clientAllowed := strings.TrimSpace(opts.ClientAllowedIPs)
	if clientAllowed == "" {
		if routeAll {
			clientAllowed = "0.0.0.0/0, ::/0"
		} else {
			clientAllowed = networkCIDR
		}
	}

	keepalive := 25
	if opts.PersistentKeepaliveSeconds != nil {
		if *opts.PersistentKeepaliveSeconds <= 0 {
			keepalive = 0
		} else {
			keepalive = *opts.PersistentKeepaliveSeconds
		}
	}

	serverPublicKey := strings.TrimSpace(w.config.GetOrDefault("WIREGUARD_PUBLIC_KEY", ""))
	if serverPublicKey == "" {
		if privateKey, ok := parsed.Interface["PrivateKey"]; ok && privateKey != "" {
			serverPublicKey, err = w.keygen.DerivePublicKey(privateKey)
			if err != nil {
				return fmt.Errorf("failed to derive server public key: %w", err)
			}
			if err := w.config.Set("WIREGUARD_PUBLIC_KEY", serverPublicKey); err != nil {
				w.ui.Warningf("failed to persist server public key: %v", err)
			}
		} else {
			if opts.NonInteractive {
				return fmt.Errorf("server public key missing from configuration")
			}
			serverPublicKey, err = w.ui.PromptInput("Server public key", "")
			if err != nil {
				return err
			}
		}
	}

	clientPrivate, clientPublic, err := w.keygen.GenerateKeyPair()
	if err != nil {
		return err
	}

	var presharedKey string
	var usePSK bool
	if opts.GeneratePresharedKey != nil {
		usePSK = *opts.GeneratePresharedKey
	} else if opts.NonInteractive {
		usePSK = true
	} else {
		usePSK, err = w.ui.PromptYesNo("Generate a preshared key for this peer?", true)
		if err != nil {
			return err
		}
	}

	if opts.ProvidedPresharedKey != "" {
		presharedKey = sanitizeConfigValue(opts.ProvidedPresharedKey)
		usePSK = true
	}

	if usePSK && presharedKey == "" {
		presharedKey, err = w.keygen.GeneratePresharedKey()
		if err != nil {
			return err
		}
	}

	clientConfig := renderClientConfig(clientPrivate, nextIP, dns, serverPublicKey, presharedKey, endpoint, clientAllowed, keepalive)
	exportDir := opts.OutputDir
	if exportDir == "" {
		exportDir = defaultPeerExportDir()
	}
	exportPath, err := writeClientConfigExport(peerName, exportDir, clientConfig)
	if err != nil {
		w.ui.Warningf("Failed to export client config: %v", err)
		w.ui.Info("Client configuration (not exported):")
		w.ui.Print(clientConfig)
		return fmt.Errorf("failed to export client config: %w", err)
	}

	var qrOutput string
	var qrErr error
	if !opts.SkipQRCode {
		qrOutput, qrErr = renderASCIIQRCode(clientConfig)
	}

	serverBlock := buildServerPeerBlock(peerName, clientPublic, presharedKey, nextIP, keepalive)
	newConfig := appendPeerBlock(string(rawConfig), serverBlock)
	if err := system.WriteFile(configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to update %s: %w", configPath, err)
	}

	w.ui.Successf("Peer %s added. Client config: %s", peerName, exportPath)
	w.ui.Print("")
	w.ui.Info("Client configuration:")
	w.ui.Print(clientConfig)

	if !opts.SkipQRCode {
		if qrErr != nil {
			w.ui.Warningf("Failed to render QR code: %v", qrErr)
		} else {
			w.ui.Info("Scan this QR code from the WireGuard mobile app:")
			w.ui.Print(qrOutput)
		}
	}

	if !opts.SkipServiceRestart {
		restart, err := w.ui.PromptYesNo(fmt.Sprintf("Restart wg-quick@%s now?", interfaceName), true)
		if err == nil && restart {
			serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)
			if err := system.RestartService(serviceName); err != nil {
				w.ui.Warningf("Failed to restart %s: %v", serviceName, err)
			} else {
				w.ui.Successf("Service %s restarted", serviceName)
			}
		}
	}

	return nil
}

var clientConfigFileWriter = func(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func writeClientConfigExport(peerName, exportDir, clientConfig string) (string, error) {
	if err := os.MkdirAll(exportDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}
	if err := os.Chmod(exportDir, 0700); err != nil {
		return "", fmt.Errorf("failed to set export directory permissions: %w", err)
	}
	fileBase := safePeerFilename(peerName)
	fileName := fmt.Sprintf("%s.conf", fileBase)
	exportPath := filepath.Join(exportDir, fileName)
	if _, err := os.Stat(exportPath); err == nil {
		exportPath = filepath.Join(exportDir, fmt.Sprintf("%s-%d.conf", fileBase, time.Now().Unix()))
	}
	if err := clientConfigFileWriter(exportPath, []byte(clientConfig), 0600); err != nil {
		return "", fmt.Errorf("failed to write client config: %w", err)
	}
	return exportPath, nil
}

func buildServerPeerBlock(name, publicKey, presharedKey, allowedIP string, keepalive int) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("# Peer: %s\n", sanitizePeerName(name)))
	builder.WriteString("[Peer]\n")
	builder.WriteString(fmt.Sprintf("PublicKey = %s\n", sanitizeConfigValue(publicKey)))
	if presharedKey != "" {
		builder.WriteString(fmt.Sprintf("PresharedKey = %s\n", sanitizeConfigValue(presharedKey)))
	}
	builder.WriteString(fmt.Sprintf("AllowedIPs = %s\n", sanitizeConfigValue(allowedIP)))
	builder.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", keepalive))
	return builder.String()
}

func appendPeerBlock(existing, block string) string {
	trimmed := strings.TrimRight(existing, "\n")
	if trimmed == "" {
		return block
	}
	return trimmed + "\n\n" + block + "\n"
}

func renderClientConfig(privateKey, address, dns, serverPublicKey, presharedKey, endpoint, allowedIPs string, keepalive int) string {
	builder := strings.Builder{}
	builder.WriteString("[Interface]\n")
	builder.WriteString(fmt.Sprintf("PrivateKey = %s\n", sanitizeConfigValue(privateKey)))
	builder.WriteString(fmt.Sprintf("Address = %s\n", sanitizeConfigValue(address)))
	if dns != "" {
		builder.WriteString(fmt.Sprintf("DNS = %s\n", sanitizeConfigValue(dns)))
	}
	builder.WriteString("\n[Peer]\n")
	builder.WriteString(fmt.Sprintf("PublicKey = %s\n", sanitizeConfigValue(serverPublicKey)))
	if presharedKey != "" {
		builder.WriteString(fmt.Sprintf("PresharedKey = %s\n", sanitizeConfigValue(presharedKey)))
	}
	if endpoint != "" {
		builder.WriteString(fmt.Sprintf("Endpoint = %s\n", sanitizeConfigValue(endpoint)))
	}
	builder.WriteString(fmt.Sprintf("AllowedIPs = %s\n", sanitizeConfigValue(allowedIPs)))
	builder.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", keepalive))
	return builder.String()
}

func renderASCIIQRCode(content string) (string, error) {
	if !system.CommandExists("qrencode") {
		return "", errors.New("qrencode binary not found; install qrencode to enable QR output")
	}
	cmd := exec.Command("qrencode", "-t", "ASCIIi", "-o", "-", "-m", "2")
	cmd.Stdin = strings.NewReader(content)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("qrencode failed: %v (%s)", err, stderr.String())
	}
	return stdout.String(), nil
}
