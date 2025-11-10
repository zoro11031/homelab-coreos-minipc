#!/usr/bin/env bash
#
# 04-nfs-setup.sh
# NFS mounts configuration for UBlue uCore homelab
#
# This script configures NFS mounts for network storage:
# - Checks for pre-existing systemd mount units from BlueBuild image
# - Updates NFS server IP in existing units or creates new ones
# - Creates mount point directories
# - Tests NFS connectivity
# - Enables and starts mount units

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

DEFAULT_NFS_SERVER="192.168.7.10"

# NFS mount configurations (mount_point:nfs_export:options)
declare -A NFS_MOUNTS=(
    ["/mnt/nas-media"]="/mnt/storage/Media:ro,nfsvers=4"
    ["/mnt/nas-nextcloud"]="/mnt/storage/Nextcloud:rw,nfsvers=4"
    ["/mnt/nas-immich"]="/mnt/storage/Immich:rw,nfsvers=4"
    ["/mnt/nas-photos"]="/mnt/storage/Photos:ro,nfsvers=4"
)

# ============================================================================
# NFS Detection Functions
# ============================================================================

check_nfs_server_connectivity() {
    local nfs_server="$1"

    log_step "Testing NFS Server Connectivity"

    if test_connectivity "$nfs_server" 5; then
        log_success "NFS server $nfs_server is reachable"
        return 0
    else
        log_error "NFS server $nfs_server is not reachable"
        return 1
    fi
}

test_nfs_export() {
    local nfs_server="$1"
    local export_path="$2"

    log_info "Testing NFS export: ${nfs_server}:${export_path}"

    if showmount -e "$nfs_server" 2>/dev/null | grep -q "$export_path"; then
        log_success "Export is available: $export_path"
        return 0
    else
        log_warning "Export not found or not accessible: $export_path"
        log_info "Run 'showmount -e $nfs_server' to see available exports"
        return 1
    fi
}

# ============================================================================
# Systemd Mount Unit Functions
# ============================================================================

check_existing_mount_unit() {
    local mount_point="$1"

    # Convert mount point to systemd unit name
    local unit_name
    unit_name=$(systemd-escape --path --suffix=mount "$mount_point")

    log_info "Checking for mount unit: $unit_name" >&2

    if check_systemd_service "$unit_name"; then
        local unit_location
        unit_location=$(get_service_location "$unit_name")
        log_success "Found pre-configured mount unit: $unit_location" >&2
        echo "$unit_location"
        return 0
    else
        log_info "No pre-configured mount unit found" >&2
        return 1
    fi
}

update_mount_unit_server() {
    local unit_file="$1"
    local nfs_server="$2"
    local mount_point="$3"

    log_info "Updating NFS server in mount unit"

    # If unit is in /usr/lib, copy to /etc to modify
    if [[ "$unit_file" == /usr/lib/* ]]; then
        local unit_name
        unit_name=$(basename "$unit_file")
        local new_location="/etc/systemd/system/${unit_name}"

        log_info "Copying unit to /etc/systemd/system for modification"
        sudo mkdir -p "/etc/systemd/system"
        sudo cp "$unit_file" "$new_location"
        unit_file="$new_location"
        log_success "Copied to: $unit_file"
    fi

    # Update What= line with new NFS server
    # This is a simplified update - actual implementation would need to parse the unit file properly
    if sudo sed -i "s|What=.*|What=${nfs_server}:$(echo "${NFS_MOUNTS[$mount_point]}" | cut -d: -f1)|" "$unit_file"; then
        log_success "Updated NFS server in: $unit_file"
        return 0
    else
        log_error "Failed to update mount unit"
        return 1
    fi
}

create_mount_unit() {
    local mount_point="$1"
    local nfs_server="$2"
    local export_path="$3"
    local options="$4"

    local unit_name
    unit_name=$(systemd-escape --path --suffix=mount "$mount_point")
    local unit_file="/etc/systemd/system/${unit_name}"

    log_info "Creating mount unit: $unit_name"

    # Create mount unit file
    sudo tee "$unit_file" > /dev/null <<EOF
[Unit]
Description=NFS mount for ${mount_point}
After=network-online.target
Wants=network-online.target
Requires=network.target

[Mount]
What=${nfs_server}:${export_path}
Where=${mount_point}
Type=nfs
Options=${options}
TimeoutSec=30

[Install]
WantedBy=multi-user.target
EOF

    if [[ -f "$unit_file" ]]; then
        log_success "Created mount unit: $unit_file"
        sudo chmod 644 "$unit_file"
        return 0
    else
        log_error "Failed to create mount unit"
        return 1
    fi
}

# ============================================================================
# Interactive Configuration
# ============================================================================

interactive_nfs_config() {
    log_step "NFS Server Configuration"

    # Get NFS server IP
    local nfs_server
    nfs_server=$(load_config "NFS_SERVER" "$DEFAULT_NFS_SERVER")

    echo ""
    log_info "Current NFS server: $nfs_server"
    nfs_server=$(prompt_input "NFS server IP address" "$nfs_server")

    if ! validate_ip "$nfs_server"; then
        log_error "Invalid IP address"
        return 1
    fi

    # Test connectivity
    if ! check_nfs_server_connectivity "$nfs_server"; then
        log_warning "NFS server is not reachable"
        if ! prompt_yes_no "Continue anyway?" "no"; then
            return 1
        fi
    fi

    # Save configuration
    save_config "NFS_SERVER" "$nfs_server"
    log_success "NFS server configured: $nfs_server"

    # Show available exports
    log_info "Attempting to list available NFS exports..."
    if showmount -e "$nfs_server" 2>/dev/null; then
        echo ""
    else
        log_warning "Could not list exports (server may not be reachable)"
    fi
}

configure_nfs_mounts() {
    log_step "Configuring NFS Mounts"

    local nfs_server
    nfs_server=$(load_config "NFS_SERVER")

    local configured_count=0

    for mount_point in "${!NFS_MOUNTS[@]}"; do
        echo ""
        log_info "Configuring mount: $mount_point"

        # Parse mount configuration
        local mount_config="${NFS_MOUNTS[$mount_point]}"
        local export_path
        local mount_options
        export_path=$(echo "$mount_config" | cut -d: -f1)
        mount_options=$(echo "$mount_config" | cut -d: -f2-)

        log_info "  Export: $export_path"
        log_info "  Options: $mount_options"

        # Check if mount point directory exists
        if [[ ! -d "$mount_point" ]]; then
            log_warning "Mount point does not exist: $mount_point"
            if ! ensure_directory "$mount_point" "root:root" "755"; then
                log_error "Failed to create mount point"
                continue
            fi
        fi

        # Check for existing mount unit
        local existing_unit
        if existing_unit=$(check_existing_mount_unit "$mount_point"); then
            log_info "Using pre-configured mount unit"

            # Ask if user wants to update the NFS server
            if prompt_yes_no "Update NFS server in this mount unit?" "yes"; then
                update_mount_unit_server "$existing_unit" "$nfs_server" "$mount_point" || continue
            fi
        else
            # Create new mount unit
            if ! create_mount_unit "$mount_point" "$nfs_server" "$export_path" "$mount_options"; then
                log_error "Failed to create mount unit"
                continue
            fi
        fi

        ((configured_count++))
    done

    if [[ $configured_count -eq 0 ]]; then
        log_error "No mounts were configured"
        return 1
    fi

    log_success "Configured $configured_count mount(s)"
    return 0
}

# ============================================================================
# Mount Management Functions
# ============================================================================

enable_and_start_mounts() {
    log_step "Enabling and Starting NFS Mounts"

    # Reload systemd to pick up new/modified units
    reload_systemd

    local success_count=0
    local fail_count=0

    for mount_point in "${!NFS_MOUNTS[@]}"; do
        local unit_name
        unit_name=$(systemd-escape --path --suffix=mount "$mount_point")

        echo ""
        log_info "Processing: $unit_name"

        # Enable mount
        if enable_service "$unit_name"; then
            log_success "Enabled: $unit_name"
        else
            log_error "Failed to enable: $unit_name"
            ((fail_count++))
            continue
        fi

        # Start mount
        if start_service "$unit_name"; then
            log_success "Started: $unit_name"
            ((success_count++))
        else
            log_error "Failed to start: $unit_name"
            log_info "Check logs: sudo journalctl -u $unit_name"
            ((fail_count++))
        fi
    done

    echo ""
    log_info "Summary: $success_count succeeded, $fail_count failed"

    if [[ $success_count -gt 0 ]]; then
        return 0
    else
        return 1
    fi
}

verify_mounts() {
    log_step "Verifying NFS Mounts"

    local all_good=true

    for mount_point in "${!NFS_MOUNTS[@]}"; do
        if mountpoint -q "$mount_point" 2>/dev/null; then
            log_success "$mount_point is mounted"

            # Show mount details
            local mount_info
            mount_info=$(mount | grep "$mount_point" | head -n1)
            log_info "  $mount_info"

            # Check if writable (for rw mounts)
            local mount_config="${NFS_MOUNTS[$mount_point]}"
            if echo "$mount_config" | grep -q "rw"; then
                if touch "${mount_point}/.write-test" 2>/dev/null; then
                    rm -f "${mount_point}/.write-test"
                    log_success "  Mount is writable"
                else
                    log_warning "  Mount is not writable (may be read-only or permission issue)"
                fi
            fi
        else
            log_error "$mount_point is NOT mounted"
            all_good=false
        fi
    done

    if $all_good; then
        log_success "✓ All NFS mounts verified"
        return 0
    else
        log_error "✗ Some NFS mounts failed"
        return 1
    fi
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root
    require_sudo

    print_header "UBlue uCore Homelab - NFS Setup"

    # Check if nfs-utils is installed
    if ! check_package "nfs-utils"; then
        log_error "nfs-utils package not installed"
        log_info "Install with: sudo rpm-ostree install nfs-utils && sudo systemctl reboot"
        exit 1
    fi

    # Check if already configured
    if check_marker "nfs-setup-complete"; then
        log_info "NFS setup already completed"

        if prompt_yes_no "Reconfigure NFS mounts?" "no"; then
            log_info "Reconfiguring NFS..."
            remove_marker "nfs-setup-complete"
        else
            log_info "Skipping NFS setup"
            verify_mounts || true
            exit 0
        fi
    fi

    # Interactive NFS server configuration
    if ! interactive_nfs_config; then
        log_error "NFS configuration failed"
        exit 1
    fi

    # Configure mount units
    if ! configure_nfs_mounts; then
        log_error "Failed to configure NFS mounts"
        exit 1
    fi

    # Enable and start mounts
    if ! enable_and_start_mounts; then
        log_warning "Some mounts failed to start"
        log_info "You can retry individual mounts with: sudo systemctl start <mount-unit>"
    fi

    # Verify mounts
    if ! verify_mounts; then
        log_warning "Mount verification failed"
        log_info "Check mount status with: mount | grep /mnt/nas"
        log_info "Check logs with: sudo journalctl -u <mount-unit>"
    fi

    # Create completion marker
    create_marker "nfs-setup-complete"

    log_success "✓ NFS setup completed"
    echo ""
    log_info "Next step: Run 05-container-setup.sh to configure container services"
}

# Run main function
main "$@"
