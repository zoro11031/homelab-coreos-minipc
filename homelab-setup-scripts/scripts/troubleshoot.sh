#!/usr/bin/env bash
#
# troubleshoot.sh
# Troubleshooting tool for UBlue uCore homelab
#
# This script provides comprehensive diagnostics:
# - System status and deployment information
# - Service status checks
# - Container status and logs
# - Network connectivity tests
# - NFS mount verification
# - Disk usage analysis
# - Common issues and solutions

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# System Information
# ============================================================================

show_system_info() {
    print_header "System Information"

    # OS Information
    log_info "Operating System:"
    if [[ -f /etc/os-release ]]; then
        grep "^NAME=" /etc/os-release | cut -d= -f2 | tr -d '"' | sed 's/^/  /'
        grep "^VERSION=" /etc/os-release | cut -d= -f2 | tr -d '"' | sed 's/^/  /'
    fi

    # Kernel
    echo "  Kernel: $(uname -r)"

    # Hostname and IP
    echo "  Hostname: $(hostname)"
    echo "  IP Address: $(hostname -I | awk '{print $1}')"

    # Uptime
    echo "  Uptime: $(uptime -p)"

    echo ""
}

show_rpm_ostree_status() {
    print_header "RPM-OSTree Status"

    if check_command rpm-ostree; then
        rpm-ostree status
    else
        log_error "rpm-ostree not available"
    fi

    echo ""
}

# ============================================================================
# Service Status Checks
# ============================================================================

check_systemd_services() {
    print_header "Systemd Service Status"

    local services=(
        "podman-compose-media.service"
        "podman-compose-web.service"
        "podman-compose-cloud.service"
        "wg-quick@wg0.service"
        "mnt-nas-media.mount"
        "mnt-nas-nextcloud.mount"
        "mnt-nas-immich.mount"
    )

    for service in "${services[@]}"; do
        echo ""
        log_info "Checking: $service"

        if systemctl list-unit-files | grep -q "^${service}"; then
            if systemctl is-active --quiet "$service"; then
                log_success "  Status: Active"

                # Show brief status
                systemctl status "$service" --no-pager -l | head -5 | sed 's/^/  /'
            else
                log_error "  Status: Inactive"
                log_info "  Start with: sudo systemctl start $service"

                # Show why it failed
                if systemctl is-failed --quiet "$service"; then
                    log_error "  Service has failed!"
                    log_info "  View logs: sudo journalctl -u $service -n 50"
                fi
            fi
        else
            log_warning "  Service not found"
        fi
    done

    echo ""
}

# ============================================================================
# Container Status Checks
# ============================================================================

check_containers() {
    print_header "Container Status"

    if ! check_command podman; then
        log_error "Podman not available"
        return 1
    fi

    # Running containers
    local running_count
    running_count=$(podman ps --format "{{.Names}}" 2>/dev/null | wc -l)

    log_info "Running containers: $running_count"
    echo ""

    if [[ $running_count -gt 0 ]]; then
        podman ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null
        echo ""
    else
        log_warning "No containers are running"
        echo ""

        # Check if any containers exist but are stopped
        local stopped_count
        stopped_count=$(podman ps -a --filter "status=exited" --format "{{.Names}}" 2>/dev/null | wc -l)

        if [[ $stopped_count -gt 0 ]]; then
            log_warning "Found $stopped_count stopped container(s):"
            podman ps -a --filter "status=exited" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null
            echo ""
        fi
    fi

    # Check for errors
    local error_count
    error_count=$(podman ps -a --filter "status=error" --format "{{.Names}}" 2>/dev/null | wc -l)

    if [[ $error_count -gt 0 ]]; then
        log_error "Found $error_count container(s) in error state:"
        podman ps -a --filter "status=error" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null
        echo ""
    fi
}

check_container_logs() {
    print_header "Recent Container Errors"

    if ! check_command podman; then
        return 1
    fi

    local containers
    containers=$(podman ps --format "{{.Names}}" 2>/dev/null)

    if [[ -z "$containers" ]]; then
        log_warning "No running containers to check"
        return 0
    fi

    local found_errors=false

    while IFS= read -r container; do
        if [[ -n "$container" ]]; then
            # Check last 20 lines for errors
            if podman logs --tail 20 "$container" 2>&1 | grep -iE "(error|failed|exception)" > /dev/null; then
                if ! $found_errors; then
                    found_errors=true
                fi

                log_warning "Errors found in: $container"
                log_info "  View logs: podman logs $container"
            fi
        fi
    done <<< "$containers"

    if ! $found_errors; then
        log_success "No obvious errors in container logs"
    fi

    echo ""
}

# ============================================================================
# Network Diagnostics
# ============================================================================

check_network() {
    print_header "Network Diagnostics"

    # Default interface
    local default_iface
    default_iface=$(ip route | grep default | awk '{print $5}' | head -n1)

    if [[ -n "$default_iface" ]]; then
        log_success "Default interface: $default_iface"

        local iface_ip
        iface_ip=$(ip addr show "$default_iface" | grep "inet " | awk '{print $2}' | head -n1)
        log_info "  IP: $iface_ip"
    else
        log_error "No default interface found"
    fi

    # Default gateway
    local default_gw
    default_gw=$(ip route | grep default | awk '{print $3}' | head -n1)

    if [[ -n "$default_gw" ]]; then
        log_success "Default gateway: $default_gw"

        if ping -c 1 -W 2 "$default_gw" &> /dev/null; then
            log_success "  Gateway is reachable"
        else
            log_error "  Gateway is not reachable"
        fi
    else
        log_error "No default gateway configured"
    fi

    # Internet connectivity
    log_info "Testing internet connectivity..."
    if ping -c 1 -W 3 8.8.8.8 &> /dev/null; then
        log_success "  Internet is reachable"
    else
        log_error "  Internet is not reachable"
    fi

    # DNS
    log_info "Testing DNS resolution..."
    if host google.com &> /dev/null; then
        log_success "  DNS is working"
    else
        log_error "  DNS resolution failed"
    fi

    echo ""
}

check_wireguard() {
    print_header "WireGuard VPN Status"

    if ! check_command wg; then
        log_warning "WireGuard tools not installed"
        return 0
    fi

    if systemctl is-active --quiet wg-quick@wg0.service; then
        log_success "WireGuard service is active"

        if ip link show wg0 &> /dev/null; then
            log_success "WireGuard interface exists"

            local wg_ip
            wg_ip=$(ip addr show wg0 | grep "inet " | awk '{print $2}' | head -n1)
            log_info "  Interface IP: $wg_ip"

            # Show peers
            echo ""
            sudo wg show wg0
        else
            log_error "WireGuard interface not found"
        fi
    else
        log_warning "WireGuard service is not active"
        log_info "  Start with: sudo systemctl start wg-quick@wg0.service"
    fi

    echo ""
}

check_firewall() {
    print_header "Firewall Status"

    if systemctl is-active --quiet firewalld; then
        log_info "Firewalld is active"

        # Show active zones
        echo ""
        log_info "Active zones:"
        sudo firewall-cmd --get-active-zones 2>/dev/null | sed 's/^/  /'

        # Show open ports
        echo ""
        log_info "Open ports (public zone):"
        sudo firewall-cmd --zone=public --list-ports 2>/dev/null | sed 's/^/  /'
    else
        log_info "Firewalld is not active"
    fi

    echo ""
}

# ============================================================================
# Storage Diagnostics
# ============================================================================

check_nfs_mounts() {
    print_header "NFS Mount Status"

    local mounts=(
        "/mnt/nas-media"
        "/mnt/nas-nextcloud"
        "/mnt/nas-immich"
        "/mnt/nas-photos"
    )

    for mount in "${mounts[@]}"; do
        if mountpoint -q "$mount" 2>/dev/null; then
            log_success "$mount is mounted"

            # Show mount details
            mount | grep "$mount" | sed 's/^/  /'

            # Test read access
            if ls "$mount" &> /dev/null; then
                log_success "  Readable: Yes"
            else
                log_error "  Readable: No"
            fi

            # Test write access (for rw mounts)
            if touch "${mount}/.write-test" 2>/dev/null; then
                rm -f "${mount}/.write-test"
                log_success "  Writable: Yes"
            else
                log_info "  Writable: No (may be read-only)"
            fi
        else
            log_error "$mount is NOT mounted"

            # Check if mount unit exists
            local unit_name
            unit_name=$(systemd-escape --path --suffix=mount "$mount")

            if systemctl list-unit-files | grep -q "^${unit_name}"; then
                log_info "  Mount unit exists: $unit_name"
                log_info "  Start with: sudo systemctl start $unit_name"

                if systemctl is-failed --quiet "$unit_name"; then
                    log_error "  Mount unit has failed!"
                    log_info "  View logs: sudo journalctl -u $unit_name"
                fi
            else
                log_warning "  Mount unit not found"
            fi
        fi
        echo ""
    done
}

check_disk_usage() {
    print_header "Disk Usage"

    # Important filesystems
    local filesystems=("/" "/var" "/srv" "/mnt")

    for fs in "${filesystems[@]}"; do
        if df "$fs" &> /dev/null; then
            local usage
            usage=$(df -h "$fs" | awk 'NR==2 {print $5}' | tr -d '%')

            local avail
            avail=$(df -h "$fs" | awk 'NR==2 {print $4}')

            if [[ $usage -ge 90 ]]; then
                log_error "$fs: ${usage}% used (${avail} available) - CRITICAL"
            elif [[ $usage -ge 80 ]]; then
                log_warning "$fs: ${usage}% used (${avail} available) - WARNING"
            else
                log_success "$fs: ${usage}% used (${avail} available)"
            fi
        fi
    done

    echo ""

    # Container image storage
    if check_command podman; then
        log_info "Container storage:"
        podman system df 2>/dev/null | sed 's/^/  /'
    fi

    echo ""
}

# ============================================================================
# Configuration Checks
# ============================================================================

check_configuration() {
    print_header "Configuration Status"

    local config_file="${HOME}/.homelab-setup.conf"

    if [[ -f "$config_file" ]]; then
        log_success "Configuration file exists: $config_file"

        # Show key configurations
        echo ""
        log_info "Key configurations:"
        grep -E "^(SETUP_USER|PUID|PGID|TZ|NFS_SERVER)=" "$config_file" 2>/dev/null | sed 's/^/  /' || true
    else
        log_warning "Configuration file not found"
        log_info "  Run setup scripts to create configuration"
    fi

    echo ""

    # Check setup markers
    local marker_dir="${HOME}/.local/homelab-setup"

    if [[ -d "$marker_dir" ]]; then
        log_info "Completed setup steps:"

        local markers=(
            "preflight-complete"
            "user-setup-complete"
            "directory-setup-complete"
            "wireguard-setup-complete"
            "nfs-setup-complete"
            "container-setup-complete"
            "service-deployment-complete"
        )

        for marker in "${markers[@]}"; do
            if [[ -f "${marker_dir}/${marker}" ]]; then
                log_success "  ✓ ${marker}"
            else
                log_info "  - ${marker} (not completed)"
            fi
        done
    else
        log_warning "No setup markers found"
    fi

    echo ""
}

# ============================================================================
# Common Issues and Solutions
# ============================================================================

show_common_issues() {
    print_header "Common Issues and Solutions"

    cat <<'EOF'

Issue: Containers not starting
Solution:
  1. Check service status:
     sudo systemctl status podman-compose-<service>.service
  2. Check service logs:
     sudo journalctl -u podman-compose-<service>.service -n 50
  3. Try restarting the service:
     sudo systemctl restart podman-compose-<service>.service

Issue: NFS mounts failing
Solution:
  1. Check NFS server connectivity:
     ping <nfs-server-ip>
  2. Check if exports are available:
     showmount -e <nfs-server-ip>
  3. Check mount unit logs:
     sudo journalctl -u mnt-nas-<name>.mount
  4. Try mounting manually:
     sudo mount -t nfs <server>:<export> /mnt/nas-<name>

Issue: WireGuard not connecting
Solution:
  1. Check service status:
     sudo systemctl status wg-quick@wg0.service
  2. Check configuration:
     sudo cat /etc/wireguard/wg0.conf
  3. Check firewall:
     sudo firewall-cmd --list-ports
  4. Restart WireGuard:
     sudo systemctl restart wg-quick@wg0.service

Issue: Services accessible locally but not remotely
Solution:
  1. Check firewall rules:
     sudo firewall-cmd --list-all
  2. Add required ports:
     sudo firewall-cmd --permanent --add-port=<port>/tcp
     sudo firewall-cmd --reload
  3. Check if services are binding to correct interface

Issue: Permission denied errors
Solution:
  1. Check ownership of directories:
     ls -la /srv/containers
     ls -la /var/lib/containers/appdata
  2. Fix ownership:
     sudo chown -R <user>:<user> /srv/containers
     sudo chown -R <user>:<user> /var/lib/containers/appdata
  3. Check subuid/subgid mappings:
     cat /etc/subuid /etc/subgid

Issue: Out of disk space
Solution:
  1. Check disk usage:
     df -h
  2. Clean up old container images:
     podman system prune -a
  3. Check container logs size:
     du -sh /var/lib/containers
  4. Clean up old deployments (if using rpm-ostree):
     rpm-ostree cleanup -b

EOF

    echo ""
}

# ============================================================================
# Log Collection
# ============================================================================

collect_logs() {
    print_header "Collecting Diagnostic Logs"

    local log_dir="${HOME}/homelab-diagnostics-$(date +%Y%m%d_%H%M%S)"

    log_info "Creating diagnostic log archive: $log_dir"

    mkdir -p "$log_dir"

    # System info
    rpm-ostree status > "${log_dir}/rpm-ostree-status.txt" 2>&1 || true
    systemctl list-units --type=service --all > "${log_dir}/systemctl-services.txt" 2>&1 || true

    # Container info
    podman ps -a > "${log_dir}/podman-ps.txt" 2>&1 || true
    podman system df > "${log_dir}/podman-df.txt" 2>&1 || true

    # Network info
    ip addr > "${log_dir}/ip-addr.txt" 2>&1 || true
    ip route > "${log_dir}/ip-route.txt" 2>&1 || true

    # Mount info
    mount > "${log_dir}/mounts.txt" 2>&1 || true
    df -h > "${log_dir}/disk-usage.txt" 2>&1 || true

    # Service logs
    local services=("podman-compose-media" "podman-compose-web" "podman-compose-cloud")
    for service in "${services[@]}"; do
        sudo journalctl -u "${service}.service" -n 100 > "${log_dir}/${service}.log" 2>&1 || true
    done

    # Configuration (sanitized)
    if [[ -f "${HOME}/.homelab-setup.conf" ]]; then
        grep -v -E "(PASSWORD|TOKEN|KEY)" "${HOME}/.homelab-setup.conf" > "${log_dir}/config.txt" 2>&1 || true
    fi

    # Create archive
    tar -czf "${log_dir}.tar.gz" -C "$(dirname "$log_dir")" "$(basename "$log_dir")" 2>/dev/null || true

    if [[ -f "${log_dir}.tar.gz" ]]; then
        log_success "Diagnostic archive created: ${log_dir}.tar.gz"
        rm -rf "$log_dir"
    else
        log_success "Diagnostic logs collected in: $log_dir"
    fi

    echo ""
}

# ============================================================================
# Interactive Menu
# ============================================================================

show_menu() {
    print_header "UBlue uCore Homelab - Troubleshooting Tool"

    echo ""
    log_info "Select a diagnostic option:"
    echo ""
    echo "  1. Run all diagnostics"
    echo "  2. System information"
    echo "  3. Service status"
    echo "  4. Container status"
    echo "  5. Network diagnostics"
    echo "  6. Storage diagnostics"
    echo "  7. Configuration check"
    echo "  8. Common issues and solutions"
    echo "  9. Collect diagnostic logs"
    echo "  0. Exit"
    echo ""
}

run_all_diagnostics() {
    show_system_info
    show_rpm_ostree_status
    check_systemd_services
    check_containers
    check_container_logs
    check_network
    check_wireguard
    check_firewall
    check_nfs_mounts
    check_disk_usage
    check_configuration
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    # If arguments provided, run specific check
    if [[ $# -gt 0 ]]; then
        case $1 in
            --all|-a)
                run_all_diagnostics
                ;;
            --services|-s)
                check_systemd_services
                check_containers
                ;;
            --network|-n)
                check_network
                check_wireguard
                ;;
            --storage|-d)
                check_nfs_mounts
                check_disk_usage
                ;;
            --logs|-l)
                collect_logs
                ;;
            *)
                echo "Usage: $0 [--all|--services|--network|--storage|--logs]"
                exit 1
                ;;
        esac
        exit 0
    fi

    # Interactive mode
    while true; do
        show_menu

        read -r -p "Enter choice [1-9, 0 to exit]: " choice

        case $choice in
            1) run_all_diagnostics ;;
            2) show_system_info; show_rpm_ostree_status ;;
            3) check_systemd_services ;;
            4) check_containers; check_container_logs ;;
            5) check_network; check_wireguard; check_firewall ;;
            6) check_nfs_mounts; check_disk_usage ;;
            7) check_configuration ;;
            8) show_common_issues ;;
            9) collect_logs ;;
            0) log_info "Exiting..."; exit 0 ;;
            *) log_error "Invalid choice" ;;
        esac

        echo ""
        read -r -p "Press Enter to continue..."
    done
}

# Run main function
main "$@"
