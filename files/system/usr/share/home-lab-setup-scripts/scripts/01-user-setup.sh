#!/usr/bin/env bash
#
# 01-user-setup.sh
# User account configuration for UBlue uCore homelab
#
# This script configures the user account for running container services:
# - Creates dockeruser (optional) or uses existing user
# - Adds user to necessary groups
# - Configures sudo access if needed
# - Detects and saves UID/GID for container mapping
# - Detects timezone for container configuration

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

DEFAULT_USER="core"
DOCKER_USER="dockeruser"

# ============================================================================
# User Management Functions
# ============================================================================

check_user_exists() {
    local username="$1"
    id "$username" &> /dev/null
}

create_docker_user() {
    local username="$1"

    log_info "Creating user: $username"

    # Create user with home directory
    if sudo useradd -m -s /bin/bash "$username"; then
        log_success "User $username created successfully"

        # Set password
        log_info "Setting password for $username"
        if sudo passwd "$username"; then
            log_success "Password set for $username"
        else
            log_error "Failed to set password for $username"
            return 1
        fi

        return 0
    else
        log_error "Failed to create user $username"
        return 1
    fi
}

add_user_to_groups() {
    local username="$1"
    local groups_to_add=()

    log_info "Checking user groups for $username"

    # Check and add to wheel group (sudo access)
    if getent group wheel &> /dev/null; then
        if ! groups "$username" | grep -q wheel; then
            groups_to_add+=("wheel")
        else
            log_success "$username is already in wheel group"
        fi
    fi

    # Check and add to podman group (if exists)
    if getent group podman &> /dev/null; then
        if ! groups "$username" | grep -q podman; then
            groups_to_add+=("podman")
        else
            log_success "$username is already in podman group"
        fi
    fi

    # Add to groups
    if [[ ${#groups_to_add[@]} -gt 0 ]]; then
        log_info "Adding $username to groups: ${groups_to_add[*]}"
        for group in "${groups_to_add[@]}"; do
            if sudo usermod -aG "$group" "$username"; then
                log_success "Added $username to $group group"
            else
                log_error "Failed to add $username to $group group"
                return 1
            fi
        done
    else
        log_success "$username is already in all necessary groups"
    fi

    # Display current groups
    local current_groups
    current_groups=$(groups "$username")
    log_info "Current groups for $username: $current_groups"
}

configure_sudo_access() {
    local username="$1"

    log_info "Checking sudo access for $username"

    # Check if user has sudo access
    if sudo -l -U "$username" &> /dev/null; then
        log_success "$username has sudo access"

        # Check if passwordless sudo is configured
        if sudo -l -U "$username" 2>/dev/null | grep -q NOPASSWD; then
            log_success "$username has passwordless sudo"
        else
            log_info "$username has sudo but requires password"

            if prompt_yes_no "Configure passwordless sudo for $username?" "no"; then
                local sudoers_file="/etc/sudoers.d/${username}-homelab"
                echo "$username ALL=(ALL) NOPASSWD: ALL" | sudo tee "$sudoers_file" > /dev/null
                sudo chmod 440 "$sudoers_file"
                log_success "Passwordless sudo configured for $username"
            fi
        fi
    else
        log_warning "$username does not have sudo access"
        if prompt_yes_no "Grant sudo access to $username?" "yes"; then
            if ! groups "$username" | grep -q wheel; then
                sudo usermod -aG wheel "$username"
                log_success "Added $username to wheel group (sudo access granted)"
            fi
        fi
    fi
}

detect_user_info() {
    local username="$1"

    log_step "Detecting User Information"

    # Get UID and GID
    local uid gid
    uid=$(get_user_uid "$username")
    gid=$(get_user_gid "$username")

    log_success "UID: $uid"
    log_success "GID: $gid"

    # Save to config
    save_config "PUID" "$uid"
    save_config "PGID" "$gid"
    save_config "CONTAINER_USER" "$username"

    # Detect timezone
    local timezone
    timezone=$(detect_timezone)
    log_success "Timezone: $timezone"
    save_config "TZ" "$timezone"

    # Get home directory
    local home_dir
    home_dir=$(eval echo "~${username}")
    log_success "Home directory: $home_dir"
    save_config "USER_HOME" "$home_dir"
}

configure_subuid_subgid() {
    local username="$1"

    log_step "Checking Subuid/Subgid Mappings"

    # Check /etc/subuid
    if [[ -f /etc/subuid ]]; then
        if grep -q "^${username}:" /etc/subuid; then
            log_success "Subuid mapping exists for $username"
            local subuid_range
            subuid_range=$(grep "^${username}:" /etc/subuid)
            log_info "  $subuid_range"
        else
            log_warning "No subuid mapping for $username"
            log_info "This may prevent rootless container operations"

            if prompt_yes_no "Create subuid mapping for $username?" "yes"; then
                # Find next available subuid range
                local next_subuid=100000
                if [[ -s /etc/subuid ]]; then
                    next_subuid=$(awk -F: '{print $2 + $3}' /etc/subuid | sort -n | tail -1)
                    ((next_subuid += 1))
                fi

                echo "${username}:${next_subuid}:65536" | sudo tee -a /etc/subuid > /dev/null
                log_success "Created subuid mapping: ${username}:${next_subuid}:65536"
            fi
        fi
    else
        log_warning "/etc/subuid does not exist"
        log_info "Creating /etc/subuid"
        echo "${username}:100000:65536" | sudo tee /etc/subuid > /dev/null
        log_success "Created /etc/subuid with mapping for $username"
    fi

    # Check /etc/subgid
    if [[ -f /etc/subgid ]]; then
        if grep -q "^${username}:" /etc/subgid; then
            log_success "Subgid mapping exists for $username"
            local subgid_range
            subgid_range=$(grep "^${username}:" /etc/subgid)
            log_info "  $subgid_range"
        else
            log_warning "No subgid mapping for $username"
            log_info "This may prevent rootless container operations"

            if prompt_yes_no "Create subgid mapping for $username?" "yes"; then
                # Find next available subgid range
                local next_subgid=100000
                if [[ -s /etc/subgid ]]; then
                    next_subgid=$(awk -F: '{print $2 + $3}' /etc/subgid | sort -n | tail -1)
                    ((next_subgid += 1))
                fi

                echo "${username}:${next_subgid}:65536" | sudo tee -a /etc/subgid > /dev/null
                log_success "Created subgid mapping: ${username}:${next_subgid}:65536"
            fi
        fi
    else
        log_warning "/etc/subgid does not exist"
        log_info "Creating /etc/subgid"
        echo "${username}:100000:65536" | sudo tee /etc/subgid > /dev/null
        log_success "Created /etc/subgid with mapping for $username"
    fi
}

verify_user_setup() {
    local username="$1"

    log_step "Verifying User Setup"

    local all_good=true

    # Check user exists
    if check_user_exists "$username"; then
        log_success "User $username exists"
    else
        log_error "User $username does not exist"
        all_good=false
    fi

    # Check sudo access
    if sudo -l -U "$username" &> /dev/null; then
        log_success "User $username has sudo access"
    else
        log_warning "User $username does not have sudo access"
    fi

    # Check groups
    local required_groups=("wheel")
    for group in "${required_groups[@]}"; do
        if getent group "$group" &> /dev/null && groups "$username" | grep -q "$group"; then
            log_success "User $username is in $group group"
        else
            log_warning "User $username is not in $group group"
        fi
    done

    # Check subuid/subgid
    if grep -q "^${username}:" /etc/subuid 2>/dev/null; then
        log_success "Subuid mapping configured"
    else
        log_warning "Subuid mapping not configured"
    fi

    if grep -q "^${username}:" /etc/subgid 2>/dev/null; then
        log_success "Subgid mapping configured"
    else
        log_warning "Subgid mapping not configured"
    fi

    if $all_good; then
        log_success "✓ User setup verification passed"
        return 0
    else
        log_warning "⚠ User setup verification completed with warnings"
        return 1
    fi
}

# ============================================================================
# Main Setup Functions
# ============================================================================

interactive_user_setup() {
    # First, select container runtime
    if ! config_exists "CONTAINER_RUNTIME"; then
        select_container_runtime || {
            log_error "Container runtime selection failed"
            return 1
        }
    else
        local runtime
        runtime=$(load_config "CONTAINER_RUNTIME")
        log_info "Using configured container runtime: $runtime"
    fi

    log_step "User Configuration"

    local target_user=""
    local current_user
    current_user=$(whoami)

    echo ""
    log_warning "SECURITY BEST PRACTICE:"
    log_info "Create a dedicated user for container management separate from your admin user."
    log_info "This user will own all container files but you won't log in as this user."
    log_info "You'll continue using '$current_user' to run these scripts and manage the system."
    echo ""
    log_info "Options:"
    log_info "  ${COLOR_BOLD}1. Create a new dedicated user (RECOMMENDED)${COLOR_RESET}"
    log_info "  2. Use current user ($current_user) - not recommended for production"
    echo ""

    while [[ -z "$target_user" ]]; do
        read -r -p "Choose option [1]: " choice
        choice=${choice:-1}

        case $choice in
            1)
                # Create new dedicated user (RECOMMENDED)
                local new_username
                new_username=$(prompt_input "Enter new username for container management" "containeruser")

                if check_user_exists "$new_username"; then
                    log_warning "User $new_username already exists"
                    if prompt_yes_no "Use existing user $new_username?" "yes"; then
                        target_user="$new_username"
                    else
                        log_info "Please choose a different username"
                        continue
                    fi
                else
                    if ! create_docker_user "$new_username"; then
                        log_error "Failed to create user $new_username"
                        continue
                    fi
                    target_user="$new_username"
                fi
                ;;
            2)
                # Use current user (not recommended for production)
                log_warning "Using admin user for containers is not recommended for security"
                if prompt_yes_no "Are you sure you want to use $current_user?" "no"; then
                    target_user="$current_user"
                    log_info "Using current user: $target_user"
                else
                    log_info "Please choose option 1 to create a dedicated user"
                    continue
                fi
                ;;
            *)
                log_error "Invalid choice. Please enter 1 or 2."
                ;;
        esac
    done

    echo ""
    log_success "Selected user: $target_user"

    # Configure user
    add_user_to_groups "$target_user"
    configure_sudo_access "$target_user"
    configure_subuid_subgid "$target_user"

    # Detect UID/GID from the target user (NOT the user running the script)
    detect_user_info "$target_user"

    log_info "Container files will be owned by $target_user (UID: $(get_user_uid "$target_user"), GID: $(get_user_gid "$target_user"))"

    # Verify setup
    verify_user_setup "$target_user"

    # Save final configuration
    save_config "SETUP_USER" "$target_user"

    log_success "User setup complete for: $target_user"

    # Note about relogin
    if [[ "$target_user" == "$current_user" ]]; then
        log_warning "Note: Group changes may require logout/login to take effect"
        log_info "If you experience permission issues, log out and log back in"
    fi
}

show_configuration_summary() {
    log_step "User Configuration Summary"

    echo ""
    print_separator
    log_info "Configuration saved to: $CONFIG_FILE"
    print_separator

    # Load and display saved configuration
    local setup_user puid pgid tz user_home container_runtime

    setup_user=$(load_config "SETUP_USER" "")
    puid=$(load_config "PUID" "")
    pgid=$(load_config "PGID" "")
    tz=$(load_config "TZ" "")
    user_home=$(load_config "USER_HOME" "")
    container_runtime=$(load_config "CONTAINER_RUNTIME" "")

    if [[ -n "$container_runtime" ]]; then
        echo "CONTAINER_RUNTIME=$container_runtime"
    fi

    if [[ -n "$setup_user" ]]; then
        echo "SETUP_USER=$setup_user"
        echo "CONTAINER_USER=$setup_user"
        echo "PUID=$puid"
        echo "PGID=$pgid"
        echo "TZ=$tz"
        echo "USER_HOME=$user_home"
    fi

    print_separator
    echo ""

    log_info "These values will be used in container environment variables"
    log_info "Next step: Run 02-directory-setup.sh to create directory structure"
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root
    require_sudo

    print_header "UBlue uCore Homelab - User Setup"

    log_info "Run this script as your admin user (e.g., 'core')"
    log_info "This script will create a separate dedicated user for container management"
    echo ""

    # Check if already configured
    if config_exists "SETUP_USER"; then
        local existing_user
        existing_user=$(load_config "SETUP_USER")

        log_info "User setup already configured for: $existing_user"

        if prompt_yes_no "Reconfigure user setup?" "no"; then
            log_info "Reconfiguring user setup..."
        else
            log_info "Skipping user setup"
            show_configuration_summary
            exit 0
        fi
    fi

    # Run interactive setup
    interactive_user_setup

    # Show summary
    show_configuration_summary

    # Create completion marker
    create_marker "user-setup-complete"

    log_success "✓ User setup completed successfully"
}

# Run main function
main "$@"
