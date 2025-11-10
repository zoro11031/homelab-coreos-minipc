#!/usr/bin/env bash
#
# 00-preflight-check.sh
# Pre-flight environment validation for UBlue uCore homelab setup
#
# This script verifies that the system is ready for homelab setup by checking:
# - Operating system (uCore/rpm-ostree)
# - Required packages and commands
# - Pre-existing systemd services from BlueBuild image
# - Template locations
# - Network connectivity
# - Disk space
# - Home directory setup completion

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

ERRORS=0
WARNINGS=0

# Core required packages for homelab setup
CORE_PACKAGES=(
    "nfs-utils"
    "wireguard-tools"
)

# Container runtime packages (at least one required)
CONTAINER_PACKAGES=(
    "podman:podman-compose"
    "docker:docker-compose"
)

# Expected systemd services from BlueBuild image
EXPECTED_SERVICES=(
    "podman-compose-media.service"
    "podman-compose-web.service"
    "podman-compose-cloud.service"
    "mnt-nas-media.mount"
    "mnt-nas-nextcloud.mount"
    "mnt-nas-immich.mount"
)

# Expected template locations
TEMPLATE_DIRS=(
    "compose-setup"
    "wireguard-setup"
)

# Minimum disk space requirements (in GB)
MIN_DISK_SPACE_ROOT=20
MIN_DISK_SPACE_VAR=50

# ============================================================================
# Check Functions
# ============================================================================

check_operating_system() {
    log_step "Checking Operating System"

    if check_ucore; then
        log_success "rpm-ostree detected - running on UBlue uCore"

        # Get deployment info
        local deployment_info
        deployment_info=$(rpm-ostree status --json 2>/dev/null || echo "{}")

        if command -v jq &> /dev/null; then
            local current_deployment
            current_deployment=$(echo "$deployment_info" | jq -r '.deployments[0].id // "unknown"')
            log_info "Current deployment: $current_deployment"
        else
            log_info "Install 'jq' for detailed deployment information"
        fi

        # Check if this is a custom BlueBuild image
        if rpm-ostree status | grep -qi "bluebuild\|ucore"; then
            log_success "Custom BlueBuild image detected"
        else
            log_warning "Could not confirm BlueBuild custom image"
            ((WARNINGS++))
        fi
    else
        log_error "rpm-ostree not found - this system does not appear to be UBlue uCore"
        log_error "These scripts are designed specifically for UBlue uCore"
        ((ERRORS++))
        return 1
    fi
}

check_required_packages() {
    log_step "Checking Required Packages"

    local missing_packages=()

    # Check core packages
    for package in "${CORE_PACKAGES[@]}"; do
        if check_package "$package"; then
            log_success "$package is installed"
        else
            log_error "$package is NOT installed"
            missing_packages+=("$package")
            ((ERRORS++))
        fi
    done

    # Check container runtime packages (at least one set required)
    local found_container_runtime=false
    for runtime_set in "${CONTAINER_PACKAGES[@]}"; do
        local runtime=$(echo "$runtime_set" | cut -d: -f1)
        local compose=$(echo "$runtime_set" | cut -d: -f2)

        if check_package "$runtime"; then
            log_success "$runtime is installed"
            found_container_runtime=true

            if check_package "$compose" || check_command "$compose"; then
                log_success "$compose is available"
            else
                log_warning "$compose is not installed (may be available via plugin)"
            fi
            break
        fi
    done

    if ! $found_container_runtime; then
        log_error "No container runtime found (podman or docker required)"
        log_info "  For Podman: sudo rpm-ostree install podman podman-compose"
        log_info "  For Docker: sudo rpm-ostree install docker docker-compose"
        ((ERRORS++))
    fi

    if [[ ${#missing_packages[@]} -gt 0 ]]; then
        echo ""
        log_error "Missing required packages. To install them:"
        log_info "  sudo rpm-ostree install ${missing_packages[*]}"
        log_info "  sudo systemctl reboot"
        echo ""
        log_warning "Note: On immutable systems, you need to layer packages and reboot"
    fi
}

check_required_commands() {
    log_step "Checking Required Commands"

    # Core commands
    local core_commands=("wg" "mount.nfs" "systemctl")

    for cmd in "${core_commands[@]}"; do
        if check_command "$cmd"; then
            log_success "$cmd command available"
        else
            log_error "$cmd command NOT found"
            ((ERRORS++))
        fi
    done

    # Container runtime commands (at least one required)
    local found_runtime_cmd=false
    if check_command podman; then
        log_success "podman command available"
        found_runtime_cmd=true

        if check_command podman-compose; then
            log_success "podman-compose command available"
        elif check_command "podman compose"; then
            log_success "podman compose command available (via plugin)"
        else
            log_warning "podman-compose not found"
        fi
    elif check_command docker; then
        log_success "docker command available"
        found_runtime_cmd=true

        if check_command docker-compose; then
            log_success "docker-compose command available"
        elif check_command "docker compose"; then
            log_success "docker compose command available (via plugin)"
        else
            log_warning "docker-compose not found"
        fi
    fi

    if ! $found_runtime_cmd; then
        log_error "No container runtime command found"
        ((ERRORS++))
    fi
}

check_systemd_services() {
    log_step "Checking Pre-configured Systemd Services"

    local found_services=0
    local missing_services=0

    for service in "${EXPECTED_SERVICES[@]}"; do
        if check_systemd_service "$service"; then
            local location
            location=$(get_service_location "$service")
            log_success "$service found at $location"
            ((found_services++))
        else
            log_warning "$service not found (will be created during setup)"
            ((missing_services++))
        fi
    done

    echo ""
    if [[ $found_services -gt 0 ]]; then
        log_success "$found_services pre-configured services found from BlueBuild image"
        log_info "These services will be enabled and started (not recreated)"
    fi

    if [[ $missing_services -gt 0 ]]; then
        log_info "$missing_services services not found (will be created during setup)"
    fi
}

check_template_locations() {
    log_step "Checking Template Locations"

    local home_setup="${HOME}/setup"
    local found_templates=0

    # Check if home-directory-setup has completed
    if [[ -f "${HOME}/.local/.home-setup-complete" ]]; then
        log_success "Home directory setup marker found"

        # Check for template directories
        for template_dir in "${TEMPLATE_DIRS[@]}"; do
            local dir_path="${home_setup}/${template_dir}"
            if [[ -d "$dir_path" ]]; then
                local file_count
                file_count=$(find "$dir_path" -type f | wc -l)
                log_success "Template directory found: $dir_path ($file_count files)"
                ((found_templates++))
            else
                log_warning "Template directory not found: $dir_path"
                ((WARNINGS++))
            fi
        done
    else
        log_warning "Home directory setup marker not found"
        log_info "Expected marker: ${HOME}/.local/.home-setup-complete"
        log_info "This suggests home-directory-setup.service hasn't run yet"
        ((WARNINGS++))
    fi

    # Check /usr/share as fallback
    for template_dir in "${TEMPLATE_DIRS[@]}"; do
        local usr_share_path="/usr/share/${template_dir}"
        if [[ -d "$usr_share_path" ]]; then
            log_info "Fallback templates found in: $usr_share_path"
        fi
    done

    if [[ $found_templates -eq 0 ]]; then
        log_warning "No template directories found in ${home_setup}"
        log_info "Setup scripts will look for templates in /usr/share as fallback"
    fi
}

check_network_connectivity() {
    log_step "Checking Network Connectivity"

    # Check internet connectivity
    if test_connectivity "8.8.8.8" 3; then
        log_success "Internet connectivity available"
    else
        log_error "No internet connectivity (required for container image pulls)"
        ((ERRORS++))
    fi

    # Check default gateway
    local default_gw
    default_gw=$(ip route | grep default | awk '{print $3}' | head -n1)
    if [[ -n "$default_gw" ]]; then
        log_success "Default gateway: $default_gw"
        if test_connectivity "$default_gw" 2; then
            log_success "Default gateway is reachable"
        else
            log_warning "Default gateway is not responding to ping"
            ((WARNINGS++))
        fi
    else
        log_error "No default gateway configured"
        ((ERRORS++))
    fi

    # Check NFS server connectivity (if configured)
    local nfs_server
    nfs_server=$(load_config "NFS_SERVER" "")
    if [[ -n "$nfs_server" ]]; then
        log_info "Testing configured NFS server: $nfs_server"
        if test_connectivity "$nfs_server" 3; then
            log_success "NFS server $nfs_server is reachable"
        else
            log_warning "NFS server $nfs_server is not reachable"
            log_info "You may need to configure the NFS server during setup"
            ((WARNINGS++))
        fi
    else
        log_info "NFS server not yet configured (will be set during setup)"
    fi
}

check_disk_space() {
    log_step "Checking Disk Space"

    # Check root filesystem
    local root_avail
    root_avail=$(df / | awk 'NR==2 {print int($4/1024/1024)}')
    if [[ $root_avail -ge $MIN_DISK_SPACE_ROOT ]]; then
        log_success "Root filesystem: ${root_avail}GB available (minimum: ${MIN_DISK_SPACE_ROOT}GB)"
    else
        log_error "Root filesystem: ${root_avail}GB available (minimum: ${MIN_DISK_SPACE_ROOT}GB required)"
        ((ERRORS++))
    fi

    # Check /var filesystem
    local var_avail
    var_avail=$(df /var | awk 'NR==2 {print int($4/1024/1024)}')
    if [[ $var_avail -ge $MIN_DISK_SPACE_VAR ]]; then
        log_success "/var filesystem: ${var_avail}GB available (minimum: ${MIN_DISK_SPACE_VAR}GB)"
    else
        log_error "/var filesystem: ${var_avail}GB available (minimum: ${MIN_DISK_SPACE_VAR}GB required)"
        ((ERRORS++))
    fi

    # Check /srv if it exists as separate mount
    if mountpoint -q /srv 2>/dev/null; then
        local srv_avail
        srv_avail=$(df /srv | awk 'NR==2 {print int($4/1024/1024)}')
        log_success "/srv filesystem: ${srv_avail}GB available"
    else
        log_info "/srv is not a separate mount point (uses root filesystem)"
    fi
}

check_user_environment() {
    log_step "Checking User Environment"

    # Check current user
    local current_user
    current_user=$(whoami)
    log_success "Running as user: $current_user"

    # Check UID/GID
    local uid gid
    uid=$(id -u)
    gid=$(id -g)
    log_info "UID: $uid, GID: $gid"

    # Check sudo access
    if sudo -n true 2>/dev/null; then
        log_success "Passwordless sudo access available"
    else
        log_info "Sudo access available (may require password)"
        sudo -v || {
            log_error "Failed to obtain sudo privileges"
            ((ERRORS++))
        }
    fi

    # Check user groups
    local groups_list
    groups_list=$(groups)
    log_info "User groups: $groups_list"

    # Check for docker/podman group
    if groups | grep -qE "(wheel|podman)"; then
        log_success "User is in privileged group (wheel or podman)"
    else
        log_warning "User is not in wheel or podman group"
        ((WARNINGS++))
    fi

    # Check home directory
    if [[ -w "$HOME" ]]; then
        log_success "Home directory is writable: $HOME"
    else
        log_error "Home directory is not writable: $HOME"
        ((ERRORS++))
    fi
}

check_podman_configuration() {
    log_step "Checking Podman Configuration"

    # Check podman version
    if check_command podman; then
        local podman_version
        podman_version=$(podman --version)
        log_success "$podman_version"
    fi

    # Check for existing containers
    local container_count
    container_count=$(podman ps -a --format "{{.Names}}" 2>/dev/null | wc -l)
    if [[ $container_count -gt 0 ]]; then
        log_info "Found $container_count existing container(s)"
        log_warning "Existing containers may conflict with homelab setup"
        ((WARNINGS++))
    else
        log_success "No existing containers found"
    fi

    # Check podman network
    if podman network ls &> /dev/null; then
        log_success "Podman networking is functional"
    else
        log_error "Podman networking is not available"
        ((ERRORS++))
    fi

    # Check for subuid/subgid
    if [[ -f /etc/subuid ]] && grep -q "^$(whoami):" /etc/subuid; then
        log_success "User subuid mapping configured"
    else
        log_warning "User subuid mapping not found in /etc/subuid"
        ((WARNINGS++))
    fi

    if [[ -f /etc/subgid ]] && grep -q "^$(whoami):" /etc/subgid; then
        log_success "User subgid mapping configured"
    else
        log_warning "User subgid mapping not found in /etc/subgid"
        ((WARNINGS++))
    fi
}

check_firewall_status() {
    log_step "Checking Firewall Status"

    if systemctl is-active --quiet firewalld; then
        log_info "Firewalld is active"
        log_warning "You may need to configure firewall rules for container services"
        ((WARNINGS++))
    else
        log_info "Firewalld is not active"
    fi

    if check_command ufw && systemctl is-active --quiet ufw; then
        log_info "UFW is active"
        log_warning "You may need to configure UFW rules for container services"
        ((WARNINGS++))
    fi
}

check_selinux_status() {
    log_step "Checking SELinux Status"

    if check_command getenforce; then
        local selinux_status
        selinux_status=$(getenforce)
        log_info "SELinux status: $selinux_status"

        if [[ "$selinux_status" == "Enforcing" ]]; then
            log_info "SELinux is enforcing (this is good for security)"
            log_info "Podman should handle SELinux contexts automatically"
        fi
    else
        log_info "SELinux commands not available"
    fi
}

# ============================================================================
# Summary Functions
# ============================================================================

print_summary() {
    print_separator
    echo ""

    if [[ $ERRORS -eq 0 ]] && [[ $WARNINGS -eq 0 ]]; then
        log_success "✓ All pre-flight checks passed!"
        echo ""
        log_info "Your system is ready for homelab setup."
        log_info "You can proceed with the next setup steps."
        echo ""
        return 0
    elif [[ $ERRORS -eq 0 ]]; then
        log_warning "⚠ Pre-flight checks completed with $WARNINGS warning(s)"
        echo ""
        log_info "Your system should work, but review warnings above."
        log_info "You can proceed with caution."
        echo ""
        return 0
    else
        log_error "✗ Pre-flight checks failed with $ERRORS error(s) and $WARNINGS warning(s)"
        echo ""
        log_error "Please fix the errors above before proceeding."
        log_info "Critical issues must be resolved for successful setup."
        echo ""
        return 1
    fi
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root

    print_header "UBlue uCore Homelab - Pre-flight Check"

    log_info "This script will verify your system is ready for homelab setup."
    echo ""

    # Run all checks
    check_operating_system || true
    check_required_packages || true
    check_required_commands || true
    check_systemd_services || true
    check_template_locations || true
    check_network_connectivity || true
    check_disk_space || true
    check_user_environment || true
    check_podman_configuration || true
    check_firewall_status || true
    check_selinux_status || true

    # Print summary
    print_summary

    # Create marker if successful
    if [[ $ERRORS -eq 0 ]]; then
        create_marker "preflight-complete"
        log_info "Preflight check marker created"
    fi

    # Exit with appropriate code
    if [[ $ERRORS -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"
