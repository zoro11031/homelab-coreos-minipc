#!/usr/bin/env bash
#
# 06-service-deployment.sh
# Service deployment for UBlue uCore homelab
#
# This script deploys container services:
# - Detects pre-existing systemd services from BlueBuild image
# - Enables and starts existing services or creates new ones
# - Pulls container images
# - Starts services via podman-compose
# - Verifies containers are running
# - Displays access URLs

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

CONTAINERS_BASE="/srv/containers"

declare -A SERVICES=(
    [media]="podman-compose-media.service"
    [web]="podman-compose-web.service"
    [cloud]="podman-compose-cloud.service"
)

# Service ports for verification
declare -A SERVICE_PORTS=(
    [plex]="32400"
    [jellyfin]="8096"
    [tautulli]="8181"
    [overseerr]="5055"
    [wizarr]="5690"
    [organizr]="9983"
    [homepage]="3000"
    [nextcloud]="8080"
    [collabora]="9980"
    [immich]="2283"
)

# ============================================================================
# Service Detection Functions
# ============================================================================

check_existing_service() {
    local service_name="$1"

    if check_systemd_service "$service_name"; then
        local location
        location=$(get_service_location "$service_name")
        log_success "Found pre-configured service: $service_name"
        log_info "  Location: $location"
        return 0
    else
        log_info "Service not found: $service_name (will be created)"
        return 1
    fi
}

create_podman_compose_service() {
    local service_name="$1"
    local service_dir="$2"
    local description="$3"

    log_info "Creating systemd service: $service_name"

    local unit_file="/etc/systemd/system/${service_name}"

    # Get compose command based on container runtime
    local compose_cmd
    compose_cmd=$(get_compose_command)

    log_info "Using compose command: $compose_cmd"

    # Create service unit
    sudo tee "$unit_file" > /dev/null <<EOF
[Unit]
Description=$description
Wants=network-online.target
After=network-online.target
RequiresMountsFor=$service_dir

[Service]
Type=oneshot
RemainAfterExit=true
WorkingDirectory=$service_dir
ExecStartPre=$compose_cmd pull
ExecStart=$compose_cmd up -d
ExecStop=$compose_cmd down
TimeoutStartSec=600

[Install]
WantedBy=multi-user.target
EOF

    if [[ -f "$unit_file" ]]; then
        log_success "Created service unit: $unit_file"
        sudo chmod 644 "$unit_file"
        return 0
    else
        log_error "Failed to create service unit"
        return 1
    fi
}

# ============================================================================
# Image Pull Functions
# ============================================================================

pull_container_images() {
    local service_dir="$1"
    local service_name="$2"

    log_step "Pulling Container Images for ${service_name^}"

    if [[ ! -f "${service_dir}/compose.yml" ]] && [[ ! -f "${service_dir}/docker-compose.yml" ]]; then
        log_error "No compose file found in $service_dir"
        return 1
    fi

    log_info "This may take several minutes depending on your internet connection..."

    # Get compose command based on container runtime
    local compose_cmd
    compose_cmd=$(get_compose_command)

    # Pull images using detected compose command
    if (cd "$service_dir" && $compose_cmd pull 2>&1 | tee /tmp/compose-pull.log); then
        log_success "Images pulled successfully"
        return 0
    else
        log_error "Failed to pull images"
        log_info "Check log: /tmp/compose-pull.log"
        return 1
    fi
}

# ============================================================================
# Service Deployment Functions
# ============================================================================

deploy_service() {
    local service="$1"
    local service_name="${SERVICES[$service]}"
    local service_dir="${CONTAINERS_BASE}/${service}"

    log_step "Deploying ${service^} Stack"

    # Verify service directory and files exist
    if [[ ! -d "$service_dir" ]]; then
        log_error "Service directory not found: $service_dir"
        return 1
    fi

    if [[ ! -f "${service_dir}/compose.yml" ]] && [[ ! -f "${service_dir}/docker-compose.yml" ]]; then
        log_error "Compose file not found in $service_dir"
        return 1
    fi

    if [[ ! -f "${service_dir}/.env" ]]; then
        log_error "Environment file not found: ${service_dir}/.env"
        return 1
    fi

    # Pull images first
    if ! pull_container_images "$service_dir" "$service"; then
        log_warning "Image pull failed, but continuing with deployment"
    fi

    # Check for existing service
    local service_exists=false
    if check_existing_service "$service_name"; then
        service_exists=true
    else
        # Create service unit
        local description="Podman Compose - ${service^} Stack"
        if ! create_podman_compose_service "$service_name" "$service_dir" "$description"; then
            log_error "Failed to create service"
            return 1
        fi
    fi

    # Reload systemd
    reload_systemd

    # Enable service
    if enable_service "$service_name"; then
        log_success "Enabled: $service_name"
    else
        log_error "Failed to enable: $service_name"
        return 1
    fi

    # Start service
    log_info "Starting service (this may take a few minutes)..."
    if start_service "$service_name"; then
        log_success "Started: $service_name"
    else
        log_error "Failed to start: $service_name"
        log_info "Check logs: sudo journalctl -u $service_name"
        return 1
    fi

    # Wait for containers to start
    log_info "Waiting for containers to initialize..."
    sleep 10

    return 0
}

deploy_all_services() {
    log_step "Deploying All Container Services"

    local deployed_count=0
    local failed_count=0

    for service in "${!SERVICES[@]}"; do
        echo ""
        if deploy_service "$service"; then
            ((deployed_count++))
            log_success "✓ ${service^} stack deployed"
        else
            ((failed_count++))
            log_error "✗ ${service^} stack deployment failed"
        fi
    done

    echo ""
    log_info "Deployment summary: $deployed_count succeeded, $failed_count failed"

    if [[ $deployed_count -gt 0 ]]; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# Verification Functions
# ============================================================================

verify_service_status() {
    local service_name="$1"

    if service_status "$service_name"; then
        log_success "$service_name is active"
        return 0
    else
        log_error "$service_name is not active"
        return 1
    fi
}

list_running_containers() {
    log_step "Listing Running Containers"

    # Get container runtime
    local runtime
    runtime=$(detect_container_runtime)

    if ! check_command "$runtime"; then
        log_error "Container runtime not available: $runtime"
        return 1
    fi

    local container_count
    container_count=$($runtime ps --format "{{.Names}}" 2>/dev/null | wc -l)

    if [[ $container_count -eq 0 ]]; then
        log_warning "No containers are currently running"
        return 1
    fi

    log_success "Found $container_count running container(s):"
    echo ""

    $runtime ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || true

    echo ""
    return 0
}

check_container_health() {
    log_step "Checking Container Health"

    # Get container runtime and compose command
    local runtime compose_cmd
    runtime=$(detect_container_runtime)
    compose_cmd=$(get_compose_command)

    local all_healthy=true

    for service in "${!SERVICES[@]}"; do
        local service_name="${SERVICES[$service]}"

        log_info "Checking ${service^} stack..."

        # Check service status
        if service_status "$service_name"; then
            log_success "  Service is active"

            # List containers for this service
            local service_dir="${CONTAINERS_BASE}/${service}"
            if [[ -d "$service_dir" ]]; then
                local containers
                containers=$(cd "$service_dir" && $compose_cmd ps --format "{{.Name}}" 2>/dev/null | grep -v "^$" || echo "")

                if [[ -n "$containers" ]]; then
                    while IFS= read -r container; do
                        if $runtime ps --filter "name=${container}" --format "{{.Names}}" 2>/dev/null | grep -q "$container"; then
                            log_success "  ✓ $container is running"
                        else
                            log_error "  ✗ $container is not running"
                            all_healthy=false
                        fi
                    done <<< "$containers"
                fi
            fi
        else
            log_error "  Service is not active"
            all_healthy=false
        fi
    done

    if $all_healthy; then
        log_success "✓ All services are healthy"
        return 0
    else
        log_warning "⚠ Some services are not healthy"
        return 1
    fi
}

# ============================================================================
# Access Information
# ============================================================================

display_access_urls() {
    log_step "Service Access Information"

    local server_ip
    server_ip=$(hostname -I | awk '{print $1}')

    echo ""
    print_separator
    log_info "Access your services at the following URLs:"
    print_separator
    echo ""

    log_info "Media Services:"
    echo "  Plex:      http://${server_ip}:32400/web"
    echo "  Jellyfin:  http://${server_ip}:8096"
    echo "  Tautulli:  http://${server_ip}:8181"
    echo ""

    log_info "Web Services:"
    echo "  Overseerr: http://${server_ip}:5055"
    echo "  Wizarr:    http://${server_ip}:5690"
    echo "  Organizr:  http://${server_ip}:9983"
    echo "  Homepage:  http://${server_ip}:3000"
    echo ""

    log_info "Cloud Services:"
    echo "  Nextcloud:  http://${server_ip}:8080"
    echo "  Collabora:  http://${server_ip}:9980"
    echo "  Immich:     http://${server_ip}:2283"
    echo ""

    print_separator
    echo ""

    log_warning "Note: Some services may take a few minutes to fully initialize"
    log_info "Check service logs if you cannot access a service:"
    log_info "  sudo journalctl -u podman-compose-<service>.service"
}

test_service_connectivity() {
    log_step "Testing Service Connectivity"

    local server_ip
    server_ip=$(hostname -I | awk '{print $1}')

    local reachable_count=0
    local total_count=${#SERVICE_PORTS[@]}

    for service in "${!SERVICE_PORTS[@]}"; do
        local port="${SERVICE_PORTS[$service]}"

        if timeout 2 bash -c "cat < /dev/null > /dev/tcp/${server_ip}/${port}" 2>/dev/null; then
            log_success "$service is reachable on port $port"
            ((reachable_count++))
        else
            log_warning "$service is not responding on port $port"
        fi
    done

    echo ""
    log_info "Service connectivity: $reachable_count/$total_count services reachable"

    if [[ $reachable_count -eq 0 ]]; then
        log_warning "No services are reachable yet - they may still be starting"
        return 1
    fi

    return 0
}

# ============================================================================
# Management Commands
# ============================================================================

show_management_commands() {
    log_step "Useful Management Commands"

    echo ""
    print_separator
    log_info "Service Management:"
    print_separator
    echo ""
    echo "  # View service status"
    echo "  sudo systemctl status podman-compose-media.service"
    echo "  sudo systemctl status podman-compose-web.service"
    echo "  sudo systemctl status podman-compose-cloud.service"
    echo ""
    echo "  # Restart a service"
    echo "  sudo systemctl restart podman-compose-<service>.service"
    echo ""
    echo "  # View service logs"
    echo "  sudo journalctl -u podman-compose-<service>.service -f"
    echo ""

    print_separator
    log_info "Container Management:"
    print_separator
    echo ""
    echo "  # List running containers"
    echo "  podman ps"
    echo ""
    echo "  # View container logs"
    echo "  podman logs <container-name>"
    echo ""
    echo "  # Enter container shell"
    echo "  podman exec -it <container-name> /bin/bash"
    echo ""

    print_separator
    log_info "Using podman-compose:"
    print_separator
    echo ""
    echo "  cd $CONTAINERS_BASE/<service>"
    echo "  podman-compose ps      # List containers"
    echo "  podman-compose logs    # View logs"
    echo "  podman-compose down    # Stop containers"
    echo "  podman-compose up -d   # Start containers"
    echo ""

    print_separator
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root
    require_sudo

    print_header "UBlue uCore Homelab - Service Deployment"

    # Check prerequisites
    if ! config_exists "SETUP_USER"; then
        log_error "User setup not completed. Run 01-user-setup.sh first."
        exit 1
    fi

    if ! check_marker "container-setup-complete"; then
        log_error "Container setup not completed. Run 05-container-setup.sh first."
        exit 1
    fi

    # Check if already deployed
    if check_marker "service-deployment-complete"; then
        log_info "Services already deployed"

        if prompt_yes_no "Redeploy services?" "no"; then
            log_info "Redeploying services..."
            remove_marker "service-deployment-complete"
        else
            log_info "Skipping deployment"
            display_access_urls
            exit 0
        fi
    fi

    # Confirm deployment
    echo ""
    log_info "This will deploy all container services:"
    for service in "${!SERVICES[@]}"; do
        log_info "  - ${service^} stack (${SERVICES[$service]})"
    done
    echo ""

    if ! prompt_yes_no "Proceed with deployment?" "yes"; then
        log_info "Deployment cancelled"
        exit 0
    fi

    # Deploy all services
    if ! deploy_all_services; then
        log_warning "Some services failed to deploy"
    fi

    # List running containers
    list_running_containers || true

    # Check container health
    check_container_health || true

    # Display access URLs
    display_access_urls

    # Test connectivity (optional)
    if prompt_yes_no "Test service connectivity?" "yes"; then
        test_service_connectivity || true
    fi

    # Show management commands
    show_management_commands

    # Create completion marker
    create_marker "service-deployment-complete"

    log_success "✓ Service deployment completed"
    echo ""
    log_info "Your homelab is now running!"
    log_info "Run troubleshoot.sh if you encounter any issues"
}

# Run main function
main "$@"
