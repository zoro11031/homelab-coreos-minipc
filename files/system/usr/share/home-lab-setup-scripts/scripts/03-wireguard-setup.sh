#!/usr/bin/env bash
#
# 03-wireguard-setup.sh
# WireGuard VPN configuration for UBlue uCore homelab
#
# This script configures WireGuard VPN:
# - Checks for templates from BlueBuild image
# - Generates keys if they don't exist
# - Auto-detects WAN interface
# - Creates wg0.conf configuration
# - Optionally exports peer configurations
# - Enables and starts WireGuard service

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

WG_CONFIG_DIR="/etc/wireguard"
WG_INTERFACE="wg0"
WG_PORT="51820"
WG_NETWORK="10.253.0.0/24"
WG_SERVER_IP="10.253.0.1/24"

TEMPLATE_DIR_HOME="${HOME}/setup/wireguard-setup"
TEMPLATE_DIR_USR="/usr/share/wireguard-setup"

# ============================================================================
# Template Detection Functions
# ============================================================================

find_wireguard_templates() {
    log_step "Locating WireGuard Templates"

    # Check home setup directory first
    if [[ -d "$TEMPLATE_DIR_HOME" ]]; then
        log_success "Found templates in: $TEMPLATE_DIR_HOME"
        echo "$TEMPLATE_DIR_HOME"
        return 0
    fi

    # Check /usr/share as fallback
    if [[ -d "$TEMPLATE_DIR_USR" ]]; then
        log_success "Found templates in: $TEMPLATE_DIR_USR"
        echo "$TEMPLATE_DIR_USR"
        return 0
    fi

    log_error "WireGuard templates not found"
    log_info "Expected locations:"
    log_info "  - $TEMPLATE_DIR_HOME"
    log_info "  - $TEMPLATE_DIR_USR"
    return 1
}

check_template_scripts() {
    local template_dir="$1"

    local required_files=(
        "generate-keys.sh"
        "apply-config.sh"
        "wg0.conf.template"
    )

    log_info "Checking template files..."

    for file in "${required_files[@]}"; do
        local file_path="${template_dir}/${file}"
        if [[ -f "$file_path" ]]; then
            log_success "Found: $file"
        else
            log_error "Missing: $file"
            return 1
        fi
    done

    return 0
}

# ============================================================================
# Key Generation Functions
# ============================================================================

generate_wireguard_keys() {
    local template_dir="$1"

    log_step "Generating WireGuard Keys"

    local keys_dir="${template_dir}/keys"
    local server_private_key="${keys_dir}/server_private.key"

    # Check if keys already exist
    if [[ -f "$server_private_key" ]]; then
        log_info "Server keys already exist"

        if prompt_yes_no "Regenerate all keys?" "no"; then
            log_warning "This will invalidate all peer configurations!"
            if ! prompt_yes_no "Are you sure?" "no"; then
                log_info "Keeping existing keys"
                return 0
            fi
            rm -rf "$keys_dir"
        else
            log_info "Using existing keys"
            return 0
        fi
    fi

    # Run generate-keys.sh script
    local generate_script="${template_dir}/generate-keys.sh"

    if [[ -x "$generate_script" ]]; then
        log_info "Running key generation script..."
        if bash "$generate_script"; then
            log_success "Keys generated successfully"
            return 0
        else
            log_error "Key generation failed"
            return 1
        fi
    else
        log_error "Generate keys script is not executable"
        log_info "Attempting to make it executable..."
        chmod +x "$generate_script"
        bash "$generate_script"
    fi
}

# ============================================================================
# Configuration Functions
# ============================================================================

detect_wan_interface() {
    log_step "Detecting WAN Interface"

    local default_iface
    default_iface=$(get_default_interface)

    if [[ -n "$default_iface" ]]; then
        log_success "Detected default interface: $default_iface"

        local iface_ip
        iface_ip=$(get_interface_ip "$default_iface")
        if [[ -n "$iface_ip" ]]; then
            log_success "Interface IP: $iface_ip"
        fi

        echo "$default_iface"
        return 0
    else
        log_error "Could not detect default interface"
        return 1
    fi
}

interactive_wireguard_config() {
    log_step "WireGuard Configuration"

    # Get WAN interface
    local wan_interface
    wan_interface=$(detect_wan_interface)

    if [[ -z "$wan_interface" ]]; then
        wan_interface=$(prompt_input "Enter WAN interface name" "eth0")
    fi

    log_info "Using WAN interface: $wan_interface"
    save_config "WG_WAN_INTERFACE" "$wan_interface"

    # Get listen port
    local listen_port
    listen_port=$(load_config "WG_PORT" "$WG_PORT")
    listen_port=$(prompt_input "WireGuard listen port" "$listen_port")

    if ! validate_port "$listen_port"; then
        log_error "Invalid port number"
        return 1
    fi

    save_config "WG_PORT" "$listen_port"

    # Get server IP
    local server_ip
    server_ip=$(load_config "WG_SERVER_IP" "$WG_SERVER_IP")
    server_ip=$(prompt_input "WireGuard server IP (with CIDR)" "$server_ip")

    save_config "WG_SERVER_IP" "$server_ip"
    save_config "WG_NETWORK" "$WG_NETWORK"

    log_success "WireGuard configuration saved"
}

apply_wireguard_config() {
    local template_dir="$1"

    log_step "Applying WireGuard Configuration"

    local apply_script="${template_dir}/apply-config.sh"

    if [[ ! -f "$apply_script" ]]; then
        log_error "Apply config script not found: $apply_script"
        return 1
    fi

    # Make script executable
    chmod +x "$apply_script" 2>/dev/null || true

    # Export configuration variables
    export WG_WAN_INTERFACE=$(load_config "WG_WAN_INTERFACE")
    export WG_PORT=$(load_config "WG_PORT" "$WG_PORT")
    export WG_SERVER_IP=$(load_config "WG_SERVER_IP" "$WG_SERVER_IP")

    log_info "Generating wg0.conf..."

    # Run apply script
    if bash "$apply_script"; then
        log_success "Configuration applied successfully"

        # Check if wg0.conf was created
        local wg_conf="${template_dir}/wg0.conf"
        if [[ -f "$wg_conf" ]]; then
            log_success "Generated: $wg_conf"
            return 0
        else
            log_error "Configuration file was not created"
            return 1
        fi
    else
        log_error "Failed to apply configuration"
        return 1
    fi
}

install_wireguard_config() {
    local template_dir="$1"

    log_step "Installing WireGuard Configuration"

    local wg_conf_src="${template_dir}/wg0.conf"
    local wg_conf_dst="${WG_CONFIG_DIR}/${WG_INTERFACE}.conf"

    # Create WireGuard config directory
    if ! sudo mkdir -p "$WG_CONFIG_DIR"; then
        log_error "Failed to create $WG_CONFIG_DIR"
        return 1
    fi

    # Backup existing configuration
    if [[ -f "$wg_conf_dst" ]]; then
        backup_file "$wg_conf_dst"
    fi

    # Copy configuration
    if sudo cp "$wg_conf_src" "$wg_conf_dst"; then
        log_success "Installed: $wg_conf_dst"

        # Set proper permissions (WireGuard requires strict permissions)
        sudo chmod 600 "$wg_conf_dst"
        sudo chown root:root "$wg_conf_dst"
        log_success "Set permissions: 600 (root:root)"

        return 0
    else
        log_error "Failed to install configuration"
        return 1
    fi
}

# ============================================================================
# Peer Configuration Functions
# ============================================================================

export_peer_configs() {
    local template_dir="$1"

    log_step "Exporting Peer Configurations"

    local export_script="${template_dir}/export-peer-configs.sh"

    if [[ ! -f "$export_script" ]]; then
        log_warning "Export script not found: $export_script"
        log_info "You'll need to manually create peer configurations"
        return 0
    fi

    if prompt_yes_no "Export peer configurations?" "yes"; then
        chmod +x "$export_script" 2>/dev/null || true

        if bash "$export_script"; then
            log_success "Peer configurations exported"

            local peer_configs_dir="${template_dir}/peer-configs"
            if [[ -d "$peer_configs_dir" ]]; then
                log_info "Peer configs saved to: $peer_configs_dir"

                # List peer configs
                local peer_count
                peer_count=$(find "$peer_configs_dir" -name "*.conf" 2>/dev/null | wc -l)
                log_success "Generated $peer_count peer configuration(s)"
            fi
        else
            log_warning "Failed to export peer configurations"
        fi
    fi
}

# ============================================================================
# Service Management Functions
# ============================================================================

enable_wireguard_service() {
    log_step "Enabling WireGuard Service"

    local service="wg-quick@${WG_INTERFACE}.service"

    # Enable service
    if enable_service "$service"; then
        log_success "Enabled: $service"
    else
        log_error "Failed to enable: $service"
        return 1
    fi

    # Start service
    if start_service "$service"; then
        log_success "Started: $service"
    else
        log_error "Failed to start: $service"
        log_info "Check logs: sudo journalctl -u $service"
        return 1
    fi

    return 0
}

verify_wireguard_status() {
    log_step "Verifying WireGuard Status"

    # Check service status
    if systemctl is-active --quiet "wg-quick@${WG_INTERFACE}.service"; then
        log_success "WireGuard service is active"
    else
        log_error "WireGuard service is not active"
        return 1
    fi

    # Check interface
    if ip link show "$WG_INTERFACE" &> /dev/null; then
        log_success "WireGuard interface exists: $WG_INTERFACE"

        # Get interface details
        local wg_ip
        wg_ip=$(ip addr show "$WG_INTERFACE" | grep "inet " | awk '{print $2}')
        if [[ -n "$wg_ip" ]]; then
            log_success "Interface IP: $wg_ip"
        fi
    else
        log_error "WireGuard interface not found: $WG_INTERFACE"
        return 1
    fi

    # Show WireGuard status
    if check_command wg; then
        log_info "WireGuard status:"
        sudo wg show "$WG_INTERFACE" | while IFS= read -r line; do
            echo "  $line"
        done
    fi

    return 0
}

# ============================================================================
# Firewall Configuration
# ============================================================================

configure_firewall() {
    log_step "Checking Firewall Configuration"

    local wg_port
    wg_port=$(load_config "WG_PORT" "$WG_PORT")

    if systemctl is-active --quiet firewalld; then
        log_info "Firewalld is active"

        if prompt_yes_no "Configure firewalld for WireGuard?" "yes"; then
            # Add WireGuard port
            if sudo firewall-cmd --permanent --add-port="${wg_port}/udp"; then
                log_success "Added port ${wg_port}/udp to firewall"
            fi

            # Add WireGuard interface to trusted zone (optional)
            if prompt_yes_no "Add WireGuard interface to trusted zone?" "yes"; then
                if sudo firewall-cmd --permanent --zone=trusted --add-interface="$WG_INTERFACE"; then
                    log_success "Added $WG_INTERFACE to trusted zone"
                fi
            fi

            # Reload firewall
            sudo firewall-cmd --reload
            log_success "Firewall reloaded"
        fi
    else
        log_info "Firewalld is not active"
        log_warning "Ensure port ${wg_port}/udp is open in your firewall"
    fi
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root
    require_sudo

    print_header "UBlue uCore Homelab - WireGuard Setup"

    # Check if WireGuard tools are installed
    if ! check_command wg; then
        log_error "WireGuard tools not installed"
        log_info "Install with: sudo rpm-ostree install wireguard-tools && sudo systemctl reboot"
        exit 1
    fi

    # Check if already configured
    if check_marker "wireguard-setup-complete"; then
        log_info "WireGuard setup already completed"

        if prompt_yes_no "Reconfigure WireGuard?" "no"; then
            log_info "Reconfiguring WireGuard..."
            remove_marker "wireguard-setup-complete"
        else
            log_info "Skipping WireGuard setup"
            verify_wireguard_status || true
            exit 0
        fi
    fi

    # Find templates
    local template_dir
    if ! template_dir=$(find_wireguard_templates); then
        log_error "Cannot proceed without templates"
        exit 1
    fi

    # Check template files
    if ! check_template_scripts "$template_dir"; then
        log_error "Missing required template files"
        exit 1
    fi

    # Generate keys
    if ! generate_wireguard_keys "$template_dir"; then
        log_error "Failed to generate keys"
        exit 1
    fi

    # Interactive configuration
    interactive_wireguard_config

    # Apply configuration
    if ! apply_wireguard_config "$template_dir"; then
        log_error "Failed to apply configuration"
        exit 1
    fi

    # Install configuration
    if ! install_wireguard_config "$template_dir"; then
        log_error "Failed to install configuration"
        exit 1
    fi

    # Export peer configs
    export_peer_configs "$template_dir" || true

    # Configure firewall
    configure_firewall || true

    # Enable and start service
    if ! enable_wireguard_service; then
        log_error "Failed to enable WireGuard service"
        exit 1
    fi

    # Verify status
    if ! verify_wireguard_status; then
        log_warning "WireGuard verification failed"
        log_info "Check logs: sudo journalctl -u wg-quick@${WG_INTERFACE}.service"
    fi

    # Create completion marker
    create_marker "wireguard-setup-complete"

    log_success "✓ WireGuard setup completed successfully"
    echo ""
    log_info "Next step: Run 04-nfs-setup.sh to configure NFS mounts"
}

# Run main function
main "$@"
