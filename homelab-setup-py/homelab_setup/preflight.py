"""
Pre-flight environment validation for UBlue uCore homelab setup.

Verifies that the system is ready for homelab setup by checking:
- Operating system (uCore/rpm-ostree)
- Required packages and commands
- Pre-existing systemd services from BlueBuild image
- Template locations
- Network connectivity
"""

import sys
from pathlib import Path
from typing import List, Tuple

from .config import get_config
from .system import (
    check_ucore,
    check_package,
    check_command,
    check_systemd_service,
    get_service_location,
    detect_container_runtime,
    user_exists,
    firewalld_is_active,
)
from .utils import (
    log_info,
    log_success,
    log_warning,
    log_error,
    log_step,
    print_header,
    print_separator,
    run_command,
    test_connectivity,
    get_default_interface,
    get_interface_ip,
    require_not_root,
    require_sudo,
)


# ============================================================================
# Global Variables
# ============================================================================

CORE_PACKAGES = [
    "nfs-utils",
    "wireguard-tools",
]

CONTAINER_PACKAGES = [
    ("podman", "podman-compose"),
    ("docker", "docker-compose"),
]

EXPECTED_SERVICES = [
    "podman-compose-media.service",
    "podman-compose-web.service",
    "podman-compose-cloud.service",
]

TEMPLATE_DIRS = [
    "compose-setup",
    "wireguard-setup",
]


# ============================================================================
# Check Functions
# ============================================================================


def check_operating_system() -> Tuple[int, int]:
    """Check operating system and deployment info."""
    log_step("Checking Operating System")

    errors = 0
    warnings = 0

    if check_ucore():
        log_success("rpm-ostree detected - running on UBlue uCore")

        # Get deployment info
        try:
            result = run_command(["rpm-ostree", "status", "--json"], check=True)
            if check_command("jq"):
                # Parse JSON to get current deployment
                import json

                try:
                    data = json.loads(result.stdout)
                    if "deployments" in data and len(data["deployments"]) > 0:
                        deployment_id = data["deployments"][0].get("id", "unknown")
                        log_info(f"Current deployment: {deployment_id}")
                except json.JSONDecodeError:
                    log_info("Install 'jq' for detailed deployment information")
        except Exception:
            pass

        # Check if this is a custom BlueBuild image
        try:
            result = run_command(["rpm-ostree", "status"], check=True)
            if "bluebuild" in result.stdout.lower() or "ucore" in result.stdout.lower():
                log_success("Custom BlueBuild image detected")
            else:
                log_warning("Could not confirm BlueBuild custom image")
                warnings += 1
        except Exception:
            pass
    else:
        log_error("rpm-ostree not found - this system does not appear to be UBlue uCore")
        log_error("These scripts are designed specifically for UBlue uCore")
        errors += 1

    return errors, warnings


def check_required_packages() -> Tuple[int, int]:
    """Check for required packages."""
    log_step("Checking Required Packages")

    errors = 0
    warnings = 0
    missing_packages = []

    # Check core packages
    for package in CORE_PACKAGES:
        if check_package(package):
            log_success(f"{package} is installed")
        else:
            log_error(f"{package} is NOT installed")
            missing_packages.append(package)
            errors += 1

    # Check container runtime packages (at least one set required)
    found_container_runtime = False
    for runtime, compose in CONTAINER_PACKAGES:
        if check_package(runtime):
            log_success(f"{runtime} is installed")
            found_container_runtime = True

            if check_package(compose) or check_command(compose):
                log_success(f"{compose} is available")
            else:
                log_warning(f"{compose} is not installed (may be available via plugin)")

            break

    if not found_container_runtime:
        log_error("No container runtime found (podman or docker required)")
        log_info("  For Podman: sudo rpm-ostree install podman podman-compose")
        log_info("  For Docker: sudo rpm-ostree install docker docker-compose")
        errors += 1

    if missing_packages:
        print()
        log_error("Missing required packages. To install them:")
        log_info(f"  sudo rpm-ostree install {' '.join(missing_packages)}")
        log_info("  sudo systemctl reboot")
        print()
        log_warning("Note: On immutable systems, you need to layer packages and reboot")

    return errors, warnings


def check_required_commands() -> Tuple[int, int]:
    """Check for required commands."""
    log_step("Checking Required Commands")

    errors = 0
    warnings = 0

    # Core commands
    core_commands = ["wg", "mount.nfs", "systemctl"]

    for cmd in core_commands:
        if check_command(cmd):
            log_success(f"{cmd} command available")
        else:
            log_error(f"{cmd} command NOT found")
            errors += 1

    # Container runtime commands (at least one required)
    found_runtime_cmd = False
    if check_command("podman"):
        log_success("podman command available")
        found_runtime_cmd = True

        if check_command("podman-compose"):
            log_success("podman-compose command available")
        elif check_command("podman"):
            # Check if podman compose (plugin) works
            try:
                run_command(["podman", "compose", "version"], check=True, capture_output=True)
                log_success("podman compose command available (via plugin)")
            except Exception:
                log_warning("podman-compose not found")
    elif check_command("docker"):
        log_success("docker command available")
        found_runtime_cmd = True

        if check_command("docker-compose"):
            log_success("docker-compose command available")
        elif check_command("docker"):
            # Check if docker compose (plugin) works
            try:
                run_command(["docker", "compose", "version"], check=True, capture_output=True)
                log_success("docker compose command available (via plugin)")
            except Exception:
                log_warning("docker-compose not found")

    if not found_runtime_cmd:
        log_error("No container runtime command found")
        errors += 1

    return errors, warnings


def check_systemd_services() -> Tuple[int, int]:
    """Check for pre-configured systemd services."""
    log_step("Checking Pre-configured Systemd Services")

    errors = 0
    warnings = 0
    found_services = 0
    missing_services = 0

    for service in EXPECTED_SERVICES:
        if check_systemd_service(service):
            location = get_service_location(service)
            log_success(f"{service} found at {location}")
            found_services += 1
        else:
            log_warning(f"{service} not found (will be created during setup)")
            missing_services += 1

    print()
    if found_services > 0:
        log_success(f"{found_services} pre-configured services found from BlueBuild image")
        log_info("These services will be enabled and started (not recreated)")

    if missing_services > 0:
        log_info(f"{missing_services} services not found (will be created during setup)")

    return errors, warnings


def check_template_locations() -> Tuple[int, int]:
    """Check for template directories."""
    log_step("Checking Template Locations")

    errors = 0
    warnings = 0
    home_setup = Path.home() / "setup"
    found_templates = 0

    # Check if home-directory-setup has completed
    marker_file = Path.home() / ".local" / ".home-setup-complete"
    if marker_file.exists():
        log_success("Home directory setup marker found")

        # Check for template directories
        for template_dir in TEMPLATE_DIRS:
            dir_path = home_setup / template_dir
            if dir_path.exists():
                file_count = len(list(dir_path.rglob("*")))
                log_success(f"Template directory found: {dir_path} ({file_count} files)")
                found_templates += 1
            else:
                log_warning(f"Template directory not found: {dir_path}")
                warnings += 1
    else:
        log_warning("Home directory setup marker not found")
        log_info(f"Expected marker: {marker_file}")
        log_info("This suggests home-directory-setup.service hasn't run yet")
        warnings += 1

    # Check /usr/share as fallback
    for template_dir in TEMPLATE_DIRS:
        usr_share_path = Path("/usr/share") / template_dir
        if usr_share_path.exists():
            log_info(f"Fallback templates found in: {usr_share_path}")

    if found_templates == 0:
        log_warning(f"No template directories found in {home_setup}")
        log_info("Setup scripts will look for templates in /usr/share as fallback")

    return errors, warnings


def check_network_connectivity() -> Tuple[int, int]:
    """Check network connectivity."""
    log_step("Checking Network Connectivity")

    errors = 0
    warnings = 0

    # Check internet connectivity
    if test_connectivity("8.8.8.8", 3):
        log_success("Internet connectivity available")
    else:
        log_error("No internet connectivity (required for container image pulls)")
        errors += 1

    # Check default gateway
    default_gw_iface = get_default_interface()
    if default_gw_iface:
        try:
            result = run_command(["ip", "route"], check=True)
            for line in result.stdout.splitlines():
                if line.startswith("default"):
                    parts = line.split()
                    if len(parts) >= 3:
                        default_gw = parts[2]
                        log_success(f"Default gateway: {default_gw}")
                        if test_connectivity(default_gw, 2):
                            log_success("Default gateway is reachable")
                        else:
                            log_warning("Default gateway is not responding to ping")
                            warnings += 1
                        break
        except Exception:
            log_error("No default gateway configured")
            errors += 1
    else:
        log_error("No default gateway configured")
        errors += 1

    # Check NFS server connectivity (if configured)
    config = get_config()
    nfs_server = config.nfs_server
    if nfs_server:
        log_info(f"Testing configured NFS server: {nfs_server}")
        if test_connectivity(nfs_server, 3):
            log_success(f"NFS server {nfs_server} is reachable")
        else:
            log_warning(f"NFS server {nfs_server} is not reachable")
            log_info("You may need to configure the NFS server during setup")
            warnings += 1
    else:
        log_info("NFS server not yet configured (will be set during setup)")

    return errors, warnings


def check_user_environment() -> Tuple[int, int]:
    """Check user environment and permissions."""
    log_step("Checking User Environment")

    errors = 0
    warnings = 0

    # Check current user
    import os

    current_user = os.getenv("USER", "unknown")
    log_success(f"Running as user: {current_user}")

    # Check UID/GID
    uid = os.getuid()
    gid = os.getgid()
    log_info(f"UID: {uid}, GID: {gid}")

    # Check sudo access
    try:
        result = run_command(["sudo", "-n", "true"], check=False, capture_output=True)
        if result.returncode == 0:
            log_success("Passwordless sudo access available")
        else:
            log_info("Sudo access available (may require password)")
            try:
                run_command(["sudo", "-v"], check=True)
            except Exception:
                log_error("Failed to obtain sudo privileges")
                errors += 1
    except Exception:
        log_error("Failed to obtain sudo privileges")
        errors += 1

    # Check user groups
    try:
        result = run_command(["groups"], check=True)
        groups_list = result.stdout.strip()
        log_info(f"User groups: {groups_list}")

        if "wheel" in groups_list or "podman" in groups_list:
            log_success("User is in privileged group (wheel or podman)")
        else:
            log_warning("User is not in wheel or podman group")
            warnings += 1
    except Exception:
        pass

    # Check home directory
    home_dir = Path.home()
    if os.access(home_dir, os.W_OK):
        log_success(f"Home directory is writable: {home_dir}")
    else:
        log_error(f"Home directory is not writable: {home_dir}")
        errors += 1

    return errors, warnings


def check_podman_configuration() -> Tuple[int, int]:
    """Check podman configuration."""
    log_step("Checking Podman Configuration")

    errors = 0
    warnings = 0

    if not check_command("podman"):
        log_info("Podman not available, skipping podman checks")
        return errors, warnings

    # Check podman version
    try:
        result = run_command(["podman", "--version"], check=True)
        log_success(result.stdout.strip())
    except Exception:
        pass

    # Check for existing containers
    try:
        result = run_command(["podman", "ps", "-a", "--format", "{{.Names}}"], check=True)
        container_count = len([line for line in result.stdout.splitlines() if line.strip()])
        if container_count > 0:
            log_info(f"Found {container_count} existing container(s)")
            log_warning("Existing containers may conflict with homelab setup")
            warnings += 1
        else:
            log_success("No existing containers found")
    except Exception:
        pass

    # Check podman network
    try:
        run_command(["podman", "network", "ls"], check=True, capture_output=True)
        log_success("Podman networking is functional")
    except Exception:
        log_error("Podman networking is not available")
        errors += 1

    # Check for subuid/subgid
    import os

    current_user = os.getenv("USER", "")
    if current_user:
        subuid_file = Path("/etc/subuid")
        subgid_file = Path("/etc/subgid")

        if subuid_file.exists():
            try:
                content = subuid_file.read_text()
                if f"{current_user}:" in content:
                    log_success("User subuid mapping configured")
                else:
                    log_warning("User subuid mapping not found in /etc/subuid")
                    warnings += 1
            except Exception:
                pass
        else:
            log_warning("/etc/subuid not found")
            warnings += 1

        if subgid_file.exists():
            try:
                content = subgid_file.read_text()
                if f"{current_user}:" in content:
                    log_success("User subgid mapping configured")
                else:
                    log_warning("User subgid mapping not found in /etc/subgid")
                    warnings += 1
            except Exception:
                pass
        else:
            log_warning("/etc/subgid not found")
            warnings += 1

    return errors, warnings


def check_firewall_status() -> Tuple[int, int]:
    """Check firewall status."""
    log_step("Checking Firewall Status")

    errors = 0
    warnings = 0

    if firewalld_is_active():
        log_info("Firewalld is active")
        log_warning("You may need to configure firewall rules for container services")
        warnings += 1
    else:
        log_info("Firewalld is not active")

    return errors, warnings


def check_selinux_status() -> Tuple[int, int]:
    """Check SELinux status."""
    log_step("Checking SELinux Status")

    errors = 0
    warnings = 0

    if check_command("getenforce"):
        try:
            result = run_command(["getenforce"], check=True)
            selinux_status = result.stdout.strip()
            log_info(f"SELinux status: {selinux_status}")

            if selinux_status == "Enforcing":
                log_info("SELinux is enforcing (this is good for security)")
                log_info("Podman should handle SELinux contexts automatically")
        except Exception:
            pass
    else:
        log_info("SELinux commands not available")

    return errors, warnings


# ============================================================================
# Summary Functions
# ============================================================================


def print_summary(total_errors: int, total_warnings: int) -> bool:
    """
    Print summary of preflight checks.

    Returns:
        True if checks passed (no errors), False otherwise
    """
    print_separator()
    print()

    if total_errors == 0 and total_warnings == 0:
        log_success("✓ All pre-flight checks passed!")
        print()
        log_info("Your system is ready for homelab setup.")
        log_info("You can proceed with the next setup steps.")
        print()
        return True
    elif total_errors == 0:
        log_warning(f"⚠ Pre-flight checks completed with {total_warnings} warning(s)")
        print()
        log_info("Your system should work, but review warnings above.")
        log_info("You can proceed with caution.")
        print()
        return True
    else:
        log_error(f"✗ Pre-flight checks failed with {total_errors} error(s) and {total_warnings} warning(s)")
        print()
        log_error("Please fix the errors above before proceeding.")
        log_info("Critical issues must be resolved for successful setup.")
        print()
        return False


# ============================================================================
# Main Function
# ============================================================================


def run_preflight() -> int:
    """
    Run all preflight checks.

    Returns:
        0 if all checks passed, 1 otherwise
    """
    require_not_root()
    require_sudo()

    print_header("UBlue uCore Homelab - Pre-flight Check")

    log_info("This script will verify your system is ready for homelab setup.")
    print()

    total_errors = 0
    total_warnings = 0

    # Run all checks
    checks = [
        check_operating_system,
        check_required_packages,
        check_required_commands,
        check_systemd_services,
        check_template_locations,
        check_network_connectivity,
        check_user_environment,
        check_podman_configuration,
        check_firewall_status,
        check_selinux_status,
    ]

    for check_func in checks:
        try:
            errors, warnings = check_func()
            total_errors += errors
            total_warnings += warnings
        except Exception as e:
            log_error(f"Check failed with exception: {e}")
            total_errors += 1

    # Print summary
    success = print_summary(total_errors, total_warnings)

    # Create marker if successful
    if success and total_errors == 0:
        config = get_config()
        config.create_marker("preflight-complete")
        log_info("Preflight check marker created")

    return 0 if total_errors == 0 else 1


if __name__ == "__main__":
    sys.exit(run_preflight())
