#!/usr/bin/env bash
#
# 02-directory-setup.sh
# Directory structure creation for UBlue uCore homelab
#
# This script creates the required directory structure for container services:
# - /srv/containers/{media,web,cloud}/ for compose files
# - /var/lib/containers/appdata/ for persistent container data
# - Mount point directories for NFS
# - Sets proper ownership and permissions

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

# Base directories
CONTAINERS_BASE="/srv/containers"
APPDATA_BASE="/var/lib/containers/appdata"

# Container service directories
CONTAINER_SERVICES=(
    "media"
    "web"
    "cloud"
)

# Application data directories
APPDATA_DIRS=(
    "plex"
    "jellyfin"
    "tautulli"
    "overseerr"
    "wizarr"
    "organizr"
    "homepage"
    "nextcloud"
    "nextcloud-db"
    "nextcloud-redis"
    "collabora"
    "immich"
    "immich-db"
    "immich-redis"
    "immich-ml"
)

# NFS mount points
NFS_MOUNTS=(
    "/mnt/nas-media"
    "/mnt/nas-nextcloud"
    "/mnt/nas-immich"
    "/mnt/nas-photos"
)

# ============================================================================
# Directory Creation Functions
# ============================================================================

create_container_directories() {
    log_step "Creating Container Service Directories"

    local setup_user
    setup_user=$(load_config "SETUP_USER" "core")

    # Create base container directory
    if ! ensure_directory "$CONTAINERS_BASE" "$setup_user:$setup_user" "755"; then
        log_error "Failed to create base containers directory"
        return 1
    fi

    # Create subdirectories for each service
    for service in "${CONTAINER_SERVICES[@]}"; do
        local service_dir="${CONTAINERS_BASE}/${service}"
        if ensure_directory "$service_dir" "$setup_user:$setup_user" "755"; then
            log_success "Created: $service_dir"
        else
            log_error "Failed to create: $service_dir"
            return 1
        fi
    done

    log_success "✓ All container directories created"
}

create_appdata_directories() {
    log_step "Creating Application Data Directories"

    local setup_user
    setup_user=$(load_config "SETUP_USER" "core")

    # Create base appdata directory
    if ! ensure_directory "$APPDATA_BASE" "$setup_user:$setup_user" "755"; then
        log_error "Failed to create base appdata directory"
        return 1
    fi

    # Create subdirectories for each application
    local total=${#APPDATA_DIRS[@]}
    local current=0

    for app in "${APPDATA_DIRS[@]}"; do
        ((current++))
        local app_dir="${APPDATA_BASE}/${app}"

        if ensure_directory "$app_dir" "$setup_user:$setup_user" "755"; then
            show_progress "$current" "$total" "$app"
        else
            echo ""
            log_error "Failed to create: $app_dir"
            return 1
        fi
    done

    echo ""
    log_success "✓ All appdata directories created"
}

create_mount_directories() {
    log_step "Creating NFS Mount Point Directories"

    # Create mount point directories
    for mount_point in "${NFS_MOUNTS[@]}"; do
        if ensure_directory "$mount_point" "root:root" "755"; then
            log_success "Created: $mount_point"
        else
            log_error "Failed to create: $mount_point"
            return 1
        fi
    done

    log_success "✓ All mount point directories created"
}

create_additional_directories() {
    log_step "Creating Additional Support Directories"

    local setup_user
    setup_user=$(load_config "SETUP_USER" "core")
    local user_home
    user_home=$(load_config "USER_HOME" "/var/home/${setup_user}")

    # Create backup directory for configuration files
    local backup_dir="${user_home}/homelab-backups"
    if ensure_directory "$backup_dir" "$setup_user:$setup_user" "700"; then
        log_success "Created backup directory: $backup_dir"
        save_config "BACKUP_DIR" "$backup_dir"
    fi

    # Create logs directory
    local logs_dir="${user_home}/homelab-logs"
    if ensure_directory "$logs_dir" "$setup_user:$setup_user" "755"; then
        log_success "Created logs directory: $logs_dir"
        save_config "LOGS_DIR" "$logs_dir"
    fi
}

# ============================================================================
# Verification Functions
# ============================================================================

verify_directory_structure() {
    log_step "Verifying Directory Structure"

    local all_good=true
    local setup_user
    setup_user=$(load_config "SETUP_USER" "core")

    # Verify base directories
    for base_dir in "$CONTAINERS_BASE" "$APPDATA_BASE"; do
        if [[ -d "$base_dir" ]]; then
            log_success "$base_dir exists"

            # Check ownership
            local owner
            owner=$(stat -c '%U:%G' "$base_dir")
            if [[ "$owner" == "$setup_user:$setup_user" ]] || [[ "$owner" == "root:root" ]]; then
                log_success "  Ownership: $owner"
            else
                log_warning "  Unexpected ownership: $owner (expected $setup_user:$setup_user)"
            fi

            # Check permissions
            local perms
            perms=$(stat -c '%a' "$base_dir")
            log_info "  Permissions: $perms"
        else
            log_error "$base_dir does not exist"
            all_good=false
        fi
    done

    # Verify container service directories
    for service in "${CONTAINER_SERVICES[@]}"; do
        local service_dir="${CONTAINERS_BASE}/${service}"
        if [[ -d "$service_dir" ]]; then
            log_success "$service_dir exists"
        else
            log_error "$service_dir does not exist"
            all_good=false
        fi
    done

    # Count appdata directories
    local appdata_count
    appdata_count=$(find "$APPDATA_BASE" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
    log_success "Found $appdata_count appdata directories in $APPDATA_BASE"

    # Verify mount points
    for mount_point in "${NFS_MOUNTS[@]}"; do
        if [[ -d "$mount_point" ]]; then
            log_success "$mount_point exists"
        else
            log_error "$mount_point does not exist"
            all_good=false
        fi
    done

    if $all_good; then
        log_success "✓ Directory structure verification passed"
        return 0
    else
        log_error "✗ Directory structure verification failed"
        return 1
    fi
}

check_disk_space() {
    log_step "Checking Available Disk Space"

    # Check /srv
    local srv_avail
    if mountpoint -q /srv 2>/dev/null; then
        srv_avail=$(df -h /srv | awk 'NR==2 {print $4}')
        log_info "/srv available space: $srv_avail"
    else
        srv_avail=$(df -h "$CONTAINERS_BASE" | awk 'NR==2 {print $4}')
        log_info "$CONTAINERS_BASE available space: $srv_avail"
    fi

    # Check /var
    local var_avail
    var_avail=$(df -h /var | awk 'NR==2 {print $4}')
    log_info "/var available space: $var_avail"

    # Check /mnt
    local mnt_avail
    mnt_avail=$(df -h /mnt | awk 'NR==2 {print $4}')
    log_info "/mnt available space: $mnt_avail"
}

# ============================================================================
# Configuration Summary
# ============================================================================

save_directory_configuration() {
    log_step "Saving Directory Configuration"

    save_config "CONTAINERS_BASE" "$CONTAINERS_BASE"
    save_config "APPDATA_BASE" "$APPDATA_BASE"
    save_config "APPDATA_PATH" "$APPDATA_BASE"  # Used in .env files

    # Save mount points
    save_config "MOUNT_NAS_MEDIA" "/mnt/nas-media"
    save_config "MOUNT_NAS_NEXTCLOUD" "/mnt/nas-nextcloud"
    save_config "MOUNT_NAS_IMMICH" "/mnt/nas-immich"
    save_config "MOUNT_NAS_PHOTOS" "/mnt/nas-photos"

    log_success "Directory configuration saved"
}

show_directory_tree() {
    log_step "Directory Structure"

    echo ""
    print_separator
    log_info "Container Services:"
    print_separator

    for service in "${CONTAINER_SERVICES[@]}"; do
        echo "${CONTAINERS_BASE}/${service}/"
    done

    echo ""
    print_separator
    log_info "Application Data (sample):"
    print_separator

    local count=0
    for app in "${APPDATA_DIRS[@]}"; do
        echo "${APPDATA_BASE}/${app}/"
        ((count++))
        if [[ $count -ge 5 ]]; then
            echo "... and $((${#APPDATA_DIRS[@]} - count)) more"
            break
        fi
    done

    echo ""
    print_separator
    log_info "NFS Mount Points:"
    print_separator

    for mount in "${NFS_MOUNTS[@]}"; do
        echo "$mount/"
    done

    print_separator
    echo ""
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root
    require_sudo

    print_header "UBlue uCore Homelab - Directory Setup"

    # Check if user setup is complete
    if ! config_exists "SETUP_USER"; then
        log_error "User setup not completed. Please run 01-user-setup.sh first."
        exit 1
    fi

    local setup_user
    setup_user=$(load_config "SETUP_USER")
    log_info "Setting up directories for user: $setup_user"

    # Check if already configured
    if check_marker "directory-setup-complete"; then
        log_info "Directory setup already completed"

        if prompt_yes_no "Recreate directory structure?" "no"; then
            log_info "Recreating directory structure..."
            remove_marker "directory-setup-complete"
        else
            log_info "Skipping directory setup"
            show_directory_tree
            exit 0
        fi
    fi

    # Create directories
    echo ""
    log_info "This will create the following directory structure:"
    log_info "  - Container service directories in $CONTAINERS_BASE"
    log_info "  - Application data directories in $APPDATA_BASE"
    log_info "  - NFS mount points in /mnt"
    echo ""

    if ! prompt_yes_no "Proceed with directory creation?" "yes"; then
        log_info "Directory setup cancelled"
        exit 0
    fi

    # Create all directories
    create_container_directories || exit 1
    create_appdata_directories || exit 1
    create_mount_directories || exit 1
    create_additional_directories || exit 1

    # Verify structure
    verify_directory_structure || exit 1

    # Check disk space
    check_disk_space

    # Save configuration
    save_directory_configuration

    # Show directory tree
    show_directory_tree

    # Create completion marker
    create_marker "directory-setup-complete"

    log_success "✓ Directory setup completed successfully"
    echo ""
    log_info "Next step: Run 03-wireguard-setup.sh to configure WireGuard VPN"
}

# Run main function
main "$@"
