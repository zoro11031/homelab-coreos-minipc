package system

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Network handles network operations
type Network struct{}

// NewNetwork creates a new Network instance
func NewNetwork() *Network {
	return &Network{}
}

// TestConnectivity tests connectivity to a host using ping
func (n *Network) TestConnectivity(host string, timeoutSeconds int) (bool, error) {
	// Use ping with specified timeout
	cmd := exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", timeoutSeconds), host)
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		// Ping returns non-zero if host is unreachable
		if exitErr.ExitCode() != 0 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to ping %s: %w", host, err)
}

// GetDefaultInterface returns the default network interface
func (n *Network) GetDefaultInterface() (string, error) {
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "default") {
			fields := strings.Fields(line)
			// Format: default via <gateway> dev <interface>
			for i, field := range fields {
				if field == "dev" && i+1 < len(fields) {
					return fields[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default interface found")
}

// GetInterfaceIP returns the IP address of a network interface
func (n *Network) GetInterfaceIP(interfaceName string) (string, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("failed to get interface %s: %w", interfaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for %s: %w", interfaceName, err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IPv4 address found for interface %s", interfaceName)
}

// GetDefaultGateway returns the default gateway IP address
func (n *Network) GetDefaultGateway() (string, error) {
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default gateway: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "default") {
			fields := strings.Fields(line)
			// Format: default via <gateway> dev <interface>
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					return fields[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default gateway found")
}

// GetHostname returns the system hostname
func (n *Network) GetHostname() (string, error) {
	cmd := exec.Command("hostname")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetAllInterfaces returns a list of all network interfaces
func (n *Network) GetAllInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var names []string
	for _, iface := range interfaces {
		names = append(names, iface.Name)
	}

	return names, nil
}

// IsPortOpen checks if a TCP port is open on a host
func (n *Network) IsPortOpen(host string, port int, timeoutSeconds int) (bool, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	timeout := time.Duration(timeoutSeconds) * time.Second

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		// Check if it's a timeout or connection refused
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return false, nil
		}
		// Connection refused or other error
		return false, nil
	}

	conn.Close()
	return true, nil
}

// ResolveDNS resolves a hostname to IP addresses
func (n *Network) ResolveDNS(hostname string) ([]string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", hostname, err)
	}

	return addrs, nil
}

// TestTCPConnection tests if a TCP connection can be established
func (n *Network) TestTCPConnection(host string, port int) (bool, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false, nil
	}
	conn.Close()
	return true, nil
}

// GetLocalIP returns the local non-loopback IP address
func (n *Network) GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no local IP address found")
}

// CheckNFSServer checks if an NFS server is reachable and has exports
func (n *Network) CheckNFSServer(serverIP string) (bool, error) {
	// First check if server is reachable
	reachable, err := n.TestConnectivity(serverIP, 5)
	if err != nil {
		return false, err
	}
	if !reachable {
		return false, nil
	}

	// Try to list exports (requires nfs-utils package)
	cmd := exec.Command("showmount", "-e", serverIP)
	err = cmd.Run()

	if err == nil {
		return true, nil
	}

	// If showmount fails, server might not be configured for NFS
	return false, nil
}

// GetNFSExports returns the list of NFS exports from a server
func (n *Network) GetNFSExports(serverIP string) (string, error) {
	cmd := exec.Command("showmount", "-e", serverIP)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get NFS exports from %s: %w", serverIP, err)
	}

	return string(output), nil
}
