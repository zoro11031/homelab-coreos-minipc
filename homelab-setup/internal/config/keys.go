package config

// Configuration key constants to prevent typos and enable autocomplete
const (
	// User configuration
	KeyHomelabUser     = "HOMELAB_USER"
	KeyHomelabUID      = "HOMELAB_UID"
	KeyHomelabGID      = "HOMELAB_GID"
	KeyHomelabTimezone = "HOMELAB_TIMEZONE"

	// Directory configuration
	KeyContainersBase = "CONTAINERS_BASE" // Base directory for container services (/srv/containers)

	// NFS configuration
	KeyNFSServer         = "NFS_SERVER"
	KeyNFSExport         = "NFS_EXPORT"
	KeyNFSMountPoint     = "NFS_MOUNT_POINT"      // User-provided mount point (may be symlink)
	KeyNFSMountPointReal = "NFS_MOUNT_POINT_REAL" // Resolved real path (critical for CoreOS)
	KeyNFSMountOptions   = "NFS_MOUNT_OPTIONS"

	// WireGuard configuration
	KeyWGInterface   = "WG_INTERFACE"
	KeyWGInterfaceIP = "WG_INTERFACE_IP"
	KeyWGListenPort  = "WG_LISTEN_PORT"
	KeyWGConfigPath  = "WG_CONFIG_PATH"

	// Container configuration
	KeyContainerRuntime   = "CONTAINER_RUNTIME"
	KeySelectedServices   = "SELECTED_SERVICES"
	KeyComposeProjectName = "COMPOSE_PROJECT_NAME"
	KeyComposeCommand     = "COMPOSE_COMMAND" // Resolved compose command (e.g., "docker compose" or "docker-compose")

	// Network configuration
	KeyNetworkTestRetries = "NETWORK_TEST_RETRIES"
	KeyNetworkTestTimeout = "NETWORK_TEST_TIMEOUT"

	// System configuration
	KeyConfigVersion = "CONFIG_VERSION"
)

// Default values for configuration keys
var Defaults = map[string]string{
	KeyContainersBase:     "/srv/containers",
	KeyContainerRuntime:   "docker", // Docker is the default runtime (Podman also supported)
	KeyNFSMountPoint:      "/mnt/nas",
	KeyNetworkTestRetries: "5",
	KeyNetworkTestTimeout: "10",
	KeyConfigVersion:      "1",
	KeyWGInterface:        "wg0",
	KeyWGListenPort:       "51820",
}
