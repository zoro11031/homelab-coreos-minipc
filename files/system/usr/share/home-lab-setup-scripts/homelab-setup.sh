#!/usr/bin/env bash
#
# homelab-setup.sh
# Main orchestrator for UBlue uCore homelab setup
#
# This script provides an interactive menu to run all setup steps
# or individual setup scripts for configuring the homelab environment.

set -euo pipefail

# ============================================================================
# Script Directory Detection
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="${SCRIPT_DIR}/scripts"

# Source common functions
# shellcheck source=scripts/common-functions.sh
if [[ -f "${SCRIPTS_DIR}/common-functions.sh" ]]; then
    source "${SCRIPTS_DIR}/common-functions.sh"
else
    echo "ERROR: common-functions.sh not found"
    echo "Expected location: ${SCRIPTS_DIR}/common-functions.sh"
    exit 1
fi

# ============================================================================
# Setup Steps Configuration
# ============================================================================

declare -A SETUP_STEPS=(
    [0]="00-preflight-check.sh:Pre-flight Check:Verify system requirements"
    [1]="01-user-setup.sh:User Setup:Configure user account and permissions"
    [2]="02-directory-setup.sh:Directory Setup:Create directory structure"
    [3]="03-wireguard-setup.sh:WireGuard Setup:Configure VPN (optional)"
    [4]="04-nfs-setup.sh:NFS Setup:Configure network storage"
    [5]="05-container-setup.sh:Container Setup:Configure container services"
    [6]="06-service-deployment.sh:Service Deployment:Deploy and start services"
)

# ============================================================================
# Menu Display Functions
# ============================================================================

show_main_menu() {
    clear
    print_header "UBlue uCore Homelab Setup"

    echo ""
    log_info "Welcome to the homelab setup wizard!"
    echo ""
    log_info "This tool will guide you through setting up your homelab environment"
    log_info "on UBlue uCore (immutable Fedora with rpm-ostree)."
    echo ""

    print_separator
    log_info "Setup Options:"
    print_separator
    echo ""

    echo -e "  ${COLOR_BOLD}[A]${COLOR_RESET} Run All Steps (Complete Setup)"
    echo -e "  ${COLOR_BOLD}[Q]${COLOR_RESET} Quick Setup (Skip WireGuard)"
    echo ""

    print_separator
    log_info "Individual Steps:"
    print_separator
    echo ""

    for i in {0..6}; do
        local step_info="${SETUP_STEPS[$i]}"
        local script=$(echo "$step_info" | cut -d: -f1)
        local title=$(echo "$step_info" | cut -d: -f2)
        local desc=$(echo "$step_info" | cut -d: -f3)

        # Check if step is completed
        local marker=$(echo "$script" | sed 's/.sh$//' | sed 's/^[0-9]*-//')
        local status="  "

        if check_marker "${marker}-complete" 2>/dev/null; then
            status="${COLOR_GREEN}✓${COLOR_RESET}"
        fi

        echo -e "  ${COLOR_BOLD}[$i]${COLOR_RESET} $status $title"
        echo -e "      ${COLOR_CYAN}→${COLOR_RESET} $desc"
        echo ""
    done

    print_separator
    log_info "Other Options:"
    print_separator
    echo ""
    echo -e "  ${COLOR_BOLD}[T]${COLOR_RESET} Troubleshooting Tool"
    echo -e "  ${COLOR_BOLD}[S]${COLOR_RESET} Show Setup Status"
    echo -e "  ${COLOR_BOLD}[R]${COLOR_RESET} Reset Setup (Clear markers)"
    echo -e "  ${COLOR_BOLD}[H]${COLOR_RESET} Help"
    echo -e "  ${COLOR_BOLD}[X]${COLOR_RESET} Exit"
    echo ""
}

show_setup_status() {
    print_header "Setup Status"

    echo ""
    log_info "Completed Steps:"
    echo ""

    local completed_count=0
    local total_count=${#SETUP_STEPS[@]}

    for i in {0..6}; do
        local step_info="${SETUP_STEPS[$i]}"
        local script=$(echo "$step_info" | cut -d: -f1)
        local title=$(echo "$step_info" | cut -d: -f2)

        local marker=$(echo "$script" | sed 's/.sh$//' | sed 's/^[0-9]*-//')

        if check_marker "${marker}-complete" 2>/dev/null; then
            log_success "✓ $title"
            ((completed_count++))
        else
            log_info "- $title (not completed)"
        fi
    done

    echo ""
    print_separator
    log_info "Progress: $completed_count/$total_count steps completed"
    print_separator
    echo ""

    # Show configuration file location
    local config_file="${HOME}/.homelab-setup.conf"
    if [[ -f "$config_file" ]]; then
        log_info "Configuration file: $config_file"
    fi

    # Show marker directory
    local marker_dir="${HOME}/.local/homelab-setup"
    if [[ -d "$marker_dir" ]]; then
        log_info "Marker directory: $marker_dir"
    fi

    echo ""
}

show_help() {
    print_header "Help"

    cat <<'EOF'

UBlue uCore Homelab Setup - Help

This setup tool configures your homelab environment on UBlue uCore, an
immutable Fedora-based operating system that uses rpm-ostree for updates.

SETUP WORKFLOW:

  The setup process consists of these steps:

  0. Pre-flight Check
     - Verifies system requirements
     - Checks for required packages
     - Detects pre-existing configurations

  1. User Setup
     - Configures user account for containers
     - Sets up groups and permissions
     - Detects UID/GID and timezone

  2. Directory Setup
     - Creates /srv/containers/ structure
     - Creates /var/lib/containers/appdata/
     - Creates NFS mount points

  3. WireGuard Setup (Optional)
     - Configures VPN server
     - Generates keys
     - Sets up peer connections

  4. NFS Setup
     - Configures network storage mounts
     - Sets up systemd mount units
     - Verifies connectivity

  5. Container Setup
     - Copies compose templates
     - Configures environment variables
     - Creates .env files

  6. Service Deployment
     - Enables systemd services
     - Pulls container images
     - Starts all services

RECOMMENDED APPROACH:

  For first-time setup, use option [A] "Run All Steps" or [Q] for
  "Quick Setup" (skips WireGuard).

  For troubleshooting or reconfiguration, run individual steps.

IMPORTANT NOTES:

  - This is an immutable system - use rpm-ostree (not dnf/yum)
  - Changes to /usr require reboot
  - Configuration is stored in /etc and /var
  - Services are pre-configured in the BlueBuild image

GETTING HELP:

  - Run troubleshooting tool: Option [T]
  - Check service status: sudo systemctl status <service>
  - View logs: sudo journalctl -u <service>
  - Check container logs: podman logs <container>

DOCUMENTATION:

  - README.md: Complete documentation
  - QUICKSTART.md: Quick reference guide

EOF

    echo ""
}

# ============================================================================
# Setup Execution Functions
# ============================================================================

run_setup_step() {
    local step_num="$1"

    if [[ ! "${SETUP_STEPS[$step_num]+isset}" ]]; then
        log_error "Invalid step number: $step_num"
        return 1
    fi

    local step_info="${SETUP_STEPS[$step_num]}"
    local script=$(echo "$step_info" | cut -d: -f1)
    local title=$(echo "$step_info" | cut -d: -f2)

    local script_path="${SCRIPTS_DIR}/${script}"

    if [[ ! -f "$script_path" ]]; then
        log_error "Script not found: $script_path"
        return 1
    fi

    log_step "Running: $title"
    echo ""

    if bash "$script_path"; then
        log_success "✓ $title completed successfully"
        return 0
    else
        log_error "✗ $title failed"
        return 1
    fi
}

run_all_steps() {
    print_header "Running Complete Setup"

    echo ""
    log_info "This will run all setup steps in sequence."
    log_warning "This may take 15-30 minutes depending on your internet connection."
    echo ""

    if ! prompt_yes_no "Proceed with complete setup?" "yes"; then
        log_info "Setup cancelled"
        return 1
    fi

    local start_time=$(date +%s)
    local failed_steps=()

    for i in {0..6}; do
        echo ""
        print_separator
        log_info "Step $((i+1))/7"
        print_separator
        echo ""

        if ! run_setup_step "$i"; then
            failed_steps+=("$i")
            log_error "Step $i failed"

            if ! prompt_yes_no "Continue with remaining steps?" "no"; then
                log_error "Setup aborted"
                return 1
            fi
        fi

        # Pause between steps
        if [[ $i -lt 6 ]]; then
            echo ""
            log_info "Pausing before next step..."
            sleep 2
        fi
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))

    echo ""
    print_separator

    if [[ ${#failed_steps[@]} -eq 0 ]]; then
        log_success "✓ Complete setup finished successfully!"
        log_info "Total time: ${minutes}m ${seconds}s"
    else
        log_warning "⚠ Setup completed with errors"
        log_info "Failed steps: ${failed_steps[*]}"
        log_info "Total time: ${minutes}m ${seconds}s"
    fi

    print_separator
    echo ""

    # Show next steps
    log_info "Next steps:"
    log_info "  1. Access your services (URLs shown in deployment output)"
    log_info "  2. Run troubleshooting if needed: ./scripts/troubleshoot.sh"
    log_info "  3. Check service status: sudo systemctl status podman-compose-*.service"
    echo ""
}

run_quick_setup() {
    print_header "Running Quick Setup (Skip WireGuard)"

    echo ""
    log_info "This will run all setup steps except WireGuard."
    log_warning "You can configure WireGuard later by running step [3]."
    echo ""

    if ! prompt_yes_no "Proceed with quick setup?" "yes"; then
        log_info "Setup cancelled"
        return 1
    fi

    local start_time=$(date +%s)
    local failed_steps=()

    # Run all steps except WireGuard (step 3)
    for i in 0 1 2 4 5 6; do
        echo ""
        print_separator
        log_info "Step $((i+1))"
        print_separator
        echo ""

        if ! run_setup_step "$i"; then
            failed_steps+=("$i")
            log_error "Step $i failed"

            if ! prompt_yes_no "Continue with remaining steps?" "no"; then
                log_error "Setup aborted"
                return 1
            fi
        fi

        # Pause between steps
        sleep 2
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))

    echo ""
    print_separator

    if [[ ${#failed_steps[@]} -eq 0 ]]; then
        log_success "✓ Quick setup finished successfully!"
        log_info "Total time: ${minutes}m ${seconds}s"
    else
        log_warning "⚠ Setup completed with errors"
        log_info "Failed steps: ${failed_steps[*]}"
        log_info "Total time: ${minutes}m ${seconds}s"
    fi

    print_separator
    echo ""
}

reset_setup() {
    print_header "Reset Setup"

    echo ""
    log_warning "This will remove all setup markers and configuration."
    log_warning "Your actual services and data will NOT be deleted."
    echo ""
    log_info "This is useful if you want to restart the setup process."
    echo ""

    if ! prompt_yes_no "Are you sure you want to reset?" "no"; then
        log_info "Reset cancelled"
        return 0
    fi

    local marker_dir="${HOME}/.local/homelab-setup"
    local config_file="${HOME}/.homelab-setup.conf"

    # Remove markers
    if [[ -d "$marker_dir" ]]; then
        rm -rf "$marker_dir"
        log_success "Removed marker directory"
    fi

    # Backup and remove config
    if [[ -f "$config_file" ]]; then
        local backup="${config_file}.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$config_file" "$backup"
        rm "$config_file"
        log_success "Backed up config to: $backup"
        log_success "Removed configuration file"
    fi

    log_success "✓ Setup reset complete"
    log_info "You can now run the setup process again"
    echo ""
}

run_troubleshoot() {
    clear
    local troubleshoot_script="${SCRIPTS_DIR}/troubleshoot.sh"

    if [[ -f "$troubleshoot_script" ]]; then
        bash "$troubleshoot_script"
    else
        log_error "Troubleshooting script not found: $troubleshoot_script"
    fi
}

# ============================================================================
# Main Interactive Loop
# ============================================================================

main() {
    # Check if running from correct directory
    if [[ ! -d "$SCRIPTS_DIR" ]]; then
        log_error "Scripts directory not found: $SCRIPTS_DIR"
        log_info "Please run this script from the homelab-setup-scripts directory"
        exit 1
    fi

    # Check if common functions are available
    if ! command -v log_info &> /dev/null; then
        echo "ERROR: Common functions not loaded properly"
        exit 1
    fi

    # Check if running as root
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should NOT be run as root"
        log_info "Please run as a regular user. Sudo will be used when needed."
        exit 1
    fi

    # Interactive loop
    while true; do
        show_main_menu

        read -r -p "Enter your choice: " choice

        case ${choice^^} in
            A)
                clear
                run_all_steps
                read -r -p "Press Enter to continue..."
                ;;
            Q)
                clear
                run_quick_setup
                read -r -p "Press Enter to continue..."
                ;;
            [0-6])
                clear
                run_setup_step "$choice"
                read -r -p "Press Enter to continue..."
                ;;
            T)
                run_troubleshoot
                ;;
            S)
                clear
                show_setup_status
                read -r -p "Press Enter to continue..."
                ;;
            R)
                clear
                reset_setup
                read -r -p "Press Enter to continue..."
                ;;
            H)
                clear
                show_help
                read -r -p "Press Enter to continue..."
                ;;
            X)
                log_info "Exiting..."
                exit 0
                ;;
            *)
                log_error "Invalid choice: $choice"
                sleep 2
                ;;
        esac
    done
}

# Run main function
main "$@"
