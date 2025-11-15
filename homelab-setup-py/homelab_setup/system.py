"""
System detection and validation utilities.

Provides functions for detecting system capabilities, checking packages,
and validating system requirements.
"""

import subprocess
from pathlib import Path
from typing import Optional, List, Tuple

from .utils import run_command, check_command, log_info, log_success, log_warning, log_error


# ============================================================================
# System Detection
# ============================================================================


def check_ucore() -> bool:
    """Check if running on UBlue uCore (rpm-ostree system)."""
    return check_command("rpm-ostree")


def check_package(package: str) -> bool:
    """
    Check if an RPM package is installed.

    Args:
        package: Package name

    Returns:
        True if installed, False otherwise
    """
    try:
        run_command(["rpm", "-q", package], check=True, capture_output=True)
        return True
    except Exception:
        return False


def check_systemd_service(service: str) -> bool:
    """
    Check if a systemd service unit file exists.

    Args:
        service: Service name (e.g., "podman-compose-media.service")

    Returns:
        True if service exists, False otherwise
    """
    locations = [
        Path("/etc/systemd/system") / service,
        Path("/usr/lib/systemd/system") / service,
        Path("/lib/systemd/system") / service,
    ]

    return any(loc.exists() for loc in locations)


def get_service_location(service: str) -> Optional[str]:
    """
    Get the location of a systemd service unit file.

    Args:
        service: Service name

    Returns:
        Path to service file, or None if not found
    """
    locations = [
        Path("/etc/systemd/system") / service,
        Path("/usr/lib/systemd/system") / service,
        Path("/lib/systemd/system") / service,
    ]

    for loc in locations:
        if loc.exists():
            return str(loc)

    return None


def service_is_active(service: str) -> bool:
    """Check if a systemd service is active."""
    try:
        result = run_command(
            ["systemctl", "is-active", "--quiet", service],
            check=False,
            capture_output=True,
        )
        return result.returncode == 0
    except Exception:
        return False


def service_is_enabled(service: str) -> bool:
    """Check if a systemd service is enabled."""
    try:
        result = run_command(
            ["systemctl", "is-enabled", "--quiet", service],
            check=False,
            capture_output=True,
        )
        return result.returncode == 0
    except Exception:
        return False


# ============================================================================
# Container Runtime Detection
# ============================================================================


def detect_container_runtime() -> Optional[str]:
    """
    Detect available container runtime (podman or docker).

    Returns:
        "podman" or "docker", or None if neither found
    """
    if check_command("podman"):
        return "podman"
    elif check_command("docker"):
        return "docker"
    return None


def get_compose_command(runtime: Optional[str] = None) -> Optional[str]:
    """
    Get the appropriate compose command for the container runtime.

    Args:
        runtime: Container runtime ("podman" or "docker"), auto-detect if None

    Returns:
        Compose command string, or None if not available
    """
    if runtime is None:
        runtime = detect_container_runtime()

    if runtime == "podman":
        if check_command("podman-compose"):
            return "podman-compose"
        else:
            return "podman compose"
    elif runtime == "docker":
        if check_command("docker-compose"):
            return "docker-compose"
        else:
            return "docker compose"

    return None


# ============================================================================
# Systemd Operations
# ============================================================================


def reload_systemd() -> bool:
    """Reload systemd daemon."""
    try:
        run_command(["systemctl", "daemon-reload"], sudo=True, check=True)
        return True
    except Exception:
        return False


def enable_service(service: str) -> bool:
    """Enable a systemd service."""
    try:
        run_command(["systemctl", "enable", service], sudo=True, check=True)
        log_success(f"Enabled: {service}")
        return True
    except Exception as e:
        log_error(f"Failed to enable {service}: {e}")
        return False


def start_service(service: str) -> bool:
    """Start a systemd service."""
    try:
        run_command(["systemctl", "start", service], sudo=True, check=True)
        log_success(f"Started: {service}")
        return True
    except Exception as e:
        log_error(f"Failed to start {service}: {e}")
        return False


def stop_service(service: str) -> bool:
    """Stop a systemd service."""
    try:
        run_command(["systemctl", "stop", service], sudo=True, check=True)
        log_success(f"Stopped: {service}")
        return True
    except Exception as e:
        log_error(f"Failed to stop {service}: {e}")
        return False


def restart_service(service: str) -> bool:
    """Restart a systemd service."""
    try:
        run_command(["systemctl", "restart", service], sudo=True, check=True)
        log_success(f"Restarted: {service}")
        return True
    except Exception as e:
        log_error(f"Failed to restart {service}: {e}")
        return False


# ============================================================================
# Mount Operations
# ============================================================================


def is_mounted(mount_point: Path) -> bool:
    """Check if a path is a mount point."""
    try:
        result = run_command(
            ["mountpoint", "-q", str(mount_point)],
            check=False,
            capture_output=True,
        )
        return result.returncode == 0
    except Exception:
        return False


def mount_nfs(server: str, export: str, mount_point: Path, options: str = "ro,nfsvers=4") -> bool:
    """
    Mount an NFS share.

    Args:
        server: NFS server address
        export: Export path on server
        mount_point: Local mount point
        options: Mount options

    Returns:
        True if successful, False otherwise
    """
    try:
        what = f"{server}:{export}"
        run_command(
            ["mount", "-t", "nfs", "-o", options, what, str(mount_point)],
            sudo=True,
            check=True,
        )
        log_success(f"Mounted: {mount_point}")
        return True
    except Exception as e:
        log_error(f"Failed to mount {mount_point}: {e}")
        return False


def unmount(mount_point: Path) -> bool:
    """
    Unmount a filesystem.

    Args:
        mount_point: Mount point to unmount

    Returns:
        True if successful, False otherwise
    """
    try:
        run_command(["umount", str(mount_point)], sudo=True, check=True)
        log_success(f"Unmounted: {mount_point}")
        return True
    except Exception as e:
        log_error(f"Failed to unmount {mount_point}: {e}")
        return False


# ============================================================================
# User Operations
# ============================================================================


def user_exists(username: str) -> bool:
    """Check if a user exists."""
    try:
        run_command(["id", username], check=True, capture_output=True)
        return True
    except Exception:
        return False


def create_user(username: str, home_dir: Optional[Path] = None) -> bool:
    """
    Create a new user.

    Args:
        username: Username to create
        home_dir: Home directory path (optional)

    Returns:
        True if successful, False otherwise
    """
    try:
        cmd = ["useradd", "-m", "-s", "/bin/bash"]
        if home_dir:
            cmd.extend(["-d", str(home_dir)])
        cmd.append(username)

        run_command(cmd, sudo=True, check=True)
        log_success(f"Created user: {username}")
        return True
    except Exception as e:
        log_error(f"Failed to create user {username}: {e}")
        return False


def add_user_to_group(username: str, group: str) -> bool:
    """
    Add a user to a group.

    Args:
        username: Username
        group: Group name

    Returns:
        True if successful, False otherwise
    """
    try:
        run_command(["usermod", "-aG", group, username], sudo=True, check=True)
        log_success(f"Added {username} to {group} group")
        return True
    except Exception as e:
        log_error(f"Failed to add {username} to {group}: {e}")
        return False


def get_user_groups(username: str) -> List[str]:
    """Get list of groups a user belongs to."""
    try:
        result = run_command(["groups", username], check=True)
        # Output format: "username : group1 group2 group3"
        parts = result.stdout.strip().split(":")
        if len(parts) >= 2:
            return parts[1].strip().split()
        return []
    except Exception:
        return []


# ============================================================================
# Firewall Operations
# ============================================================================


def firewalld_is_active() -> bool:
    """Check if firewalld is active."""
    return service_is_active("firewalld")


def firewall_add_port(port: int, protocol: str = "tcp", permanent: bool = True) -> bool:
    """
    Add a port to the firewall.

    Args:
        port: Port number
        protocol: Protocol (tcp or udp)
        permanent: Make change permanent

    Returns:
        True if successful, False otherwise
    """
    try:
        cmd = ["firewall-cmd"]
        if permanent:
            cmd.append("--permanent")
        cmd.extend(["--add-port", f"{port}/{protocol}"])

        run_command(cmd, sudo=True, check=True)
        log_success(f"Added firewall rule: {port}/{protocol}")
        return True
    except Exception as e:
        log_error(f"Failed to add firewall rule: {e}")
        return False


def firewall_reload() -> bool:
    """Reload firewall configuration."""
    try:
        run_command(["firewall-cmd", "--reload"], sudo=True, check=True)
        log_success("Firewall reloaded")
        return True
    except Exception as e:
        log_error(f"Failed to reload firewall: {e}")
        return False
