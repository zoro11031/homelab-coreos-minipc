#!/usr/bin/env bash
#
# 05-container-setup.sh
# Container service configuration for UBlue uCore homelab
#
# This script configures container services:
# - Copies compose templates from ~/setup/compose-setup/ to /srv/containers/
# - Creates subdirectory structure (media/, web/, cloud/)
# - Creates and configures .env files
# - Sets environment variables interactively
# - Sets proper ownership and permissions

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common-functions.sh
source "${SCRIPT_DIR}/common-functions.sh"

# ============================================================================
# Global Variables
# ============================================================================

TEMPLATE_DIR_HOME="${HOME}/setup/compose-setup"
TEMPLATE_DIR_USR="/usr/share/compose-setup"
CONTAINERS_BASE="/srv/containers"

# Service configurations
declare -A SERVICES=(
    [media]="media.yml"
    [web]="web.yml"
    [cloud]="cloud.yml"
)

# ============================================================================
# Template Detection Functions
# ============================================================================

find_compose_templates() {
    log_step "Locating Compose Templates"

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

    log_error "Compose templates not found"
    log_info "Expected locations:"
    log_info "  - $TEMPLATE_DIR_HOME"
    log_info "  - $TEMPLATE_DIR_USR"
    return 1
}

check_template_files() {
    local template_dir="$1"

    log_info "Checking template files..."

    local all_found=true

    for service in "${!SERVICES[@]}"; do
        local template_file="${SERVICES[$service]}"
        local file_path="${template_dir}/${template_file}"

        if [[ -f "$file_path" ]]; then
            log_success "Found: $template_file"
        else
            log_error "Missing: $template_file"
            all_found=false
        fi
    done

    # Check for .env.example
    if [[ -f "${template_dir}/.env.example" ]]; then
        log_success "Found: .env.example"
    else
        log_warning "Missing: .env.example (will create default)"
    fi

    $all_found
}

# ============================================================================
# Template Copy Functions
# ============================================================================

copy_compose_templates() {
    local template_dir="$1"

    log_step "Copying Compose Templates"

    local setup_user
    setup_user=$(load_config "SETUP_USER" "core")

    local copied_count=0

    for service in "${!SERVICES[@]}"; do
        local template_file="${SERVICES[$service]}"
        local src="${template_dir}/${template_file}"
        local dst_dir="${CONTAINERS_BASE}/${service}"
        local dst="${dst_dir}/compose.yml"

        # Ensure destination directory exists
        if [[ ! -d "$dst_dir" ]]; then
            log_error "Destination directory does not exist: $dst_dir"
            log_info "Run 02-directory-setup.sh first"
            return 1
        fi

        # Copy template
        log_info "Copying: $template_file → $dst"
        if sudo cp "$src" "$dst"; then
            sudo chown "${setup_user}:${setup_user}" "$dst"
            sudo chmod 644 "$dst"
            log_success "✓ $service/compose.yml"
            ((copied_count++))
        else
            log_error "Failed to copy: $template_file"
            return 1
        fi

        # Also try docker-compose.yml for compatibility
        local alt_dst="${dst_dir}/docker-compose.yml"
        if [[ ! -f "$alt_dst" ]]; then
            sudo ln -sf "compose.yml" "$alt_dst" 2>/dev/null || true
        fi
    done

    log_success "Copied $copied_count compose file(s)"
    return 0
}

# ============================================================================
# Environment Configuration Functions
# ============================================================================

create_base_env_config() {
    log_step "Creating Base Environment Configuration"

    # Load or detect configuration values
    local puid pgid tz appdata_path

    puid=$(load_config "PUID" "1000")
    pgid=$(load_config "PGID" "1000")
    tz=$(load_config "TZ" "America/Chicago")
    appdata_path=$(load_config "APPDATA_PATH" "/var/lib/containers/appdata")

    # Save base config
    save_config "ENV_PUID" "$puid"
    save_config "ENV_PGID" "$pgid"
    save_config "ENV_TZ" "$tz"
    save_config "ENV_APPDATA_PATH" "$appdata_path"

    log_success "Base configuration:"
    log_info "  PUID=$puid"
    log_info "  PGID=$pgid"
    log_info "  TZ=$tz"
    log_info "  APPDATA_PATH=$appdata_path"
}

configure_media_env() {
    local env_file="$1"

    log_step "Configuring Media Stack Environment"

    # Get Plex claim token
    log_info "Plex Setup:"
    log_info "  Get your claim token from: https://plex.tv/claim"
    local plex_claim
    plex_claim=$(prompt_input "Plex claim token (optional)" "")

    if [[ -n "$plex_claim" ]]; then
        save_config "PLEX_CLAIM_TOKEN" "$plex_claim"
    fi

    # Jellyfin public URL
    local jellyfin_url
    jellyfin_url=$(prompt_input "Jellyfin public URL (optional)" "")
    save_config "JELLYFIN_PUBLIC_URL" "$jellyfin_url"

    # Create env file
    create_env_file "$env_file" "media"
}

configure_web_env() {
    local env_file="$1"

    log_step "Configuring Web Stack Environment"

    # Overseerr API key (optional)
    local overseerr_api
    overseerr_api=$(prompt_input "Overseerr API key (optional, can configure later)" "")
    if [[ -n "$overseerr_api" ]]; then
        save_config "OVERSEERR_API_KEY" "$overseerr_api"
    fi

    # Create env file
    create_env_file "$env_file" "web"
}

configure_cloud_env() {
    local env_file="$1"

    log_step "Configuring Cloud Stack Environment"

    # Nextcloud configuration
    log_info "Nextcloud Setup:"

    local nextcloud_admin_user
    nextcloud_admin_user=$(prompt_input "Nextcloud admin username" "admin")
    save_config "NEXTCLOUD_ADMIN_USER" "$nextcloud_admin_user"

    local nextcloud_admin_pass
    nextcloud_admin_pass=$(prompt_password "Nextcloud admin password")
    save_config "NEXTCLOUD_ADMIN_PASSWORD" "$nextcloud_admin_pass"

    local nextcloud_db_pass
    nextcloud_db_pass=$(prompt_password "Nextcloud database password")
    save_config "NEXTCLOUD_DB_PASSWORD" "$nextcloud_db_pass"

    local nextcloud_domain
    nextcloud_domain=$(prompt_input "Nextcloud trusted domain (e.g., cloud.example.com)" "localhost")
    save_config "NEXTCLOUD_TRUSTED_DOMAINS" "$nextcloud_domain"

    # Collabora configuration
    log_info "Collabora Setup:"

    local collabora_pass
    collabora_pass=$(prompt_password "Collabora admin password")
    save_config "COLLABORA_PASSWORD" "$collabora_pass"

    # Escape domain for Collabora (dots need to be escaped)
    local collabora_domain
    collabora_domain=$(echo "$nextcloud_domain" | sed 's/\./\\\\./g')
    save_config "COLLABORA_DOMAIN" "$collabora_domain"

    # Immich configuration
    log_info "Immich Setup:"

    local immich_db_pass
    immich_db_pass=$(prompt_password "Immich database password")
    save_config "IMMICH_DB_PASSWORD" "$immich_db_pass"

    # Create env file
    create_env_file "$env_file" "cloud"
}

create_env_file() {
    local env_file="$1"
    local service="$2"

    local puid pgid tz appdata_path
    puid=$(load_config "ENV_PUID")
    pgid=$(load_config "ENV_PGID")
    tz=$(load_config "ENV_TZ")
    appdata_path=$(load_config "ENV_APPDATA_PATH")

    log_info "Creating environment file: $env_file"

    # Create base .env file
    cat > "$env_file" <<EOF
# UBlue uCore Homelab - ${service^} Stack Environment
# Generated: $(date)

# User/Group Configuration
PUID=$puid
PGID=$pgid
TZ=$tz

# Paths
APPDATA_PATH=$appdata_path

EOF

    # Add service-specific variables
    case $service in
        media)
            cat >> "$env_file" <<EOF
# Plex Configuration
PLEX_CLAIM_TOKEN=$(load_config "PLEX_CLAIM_TOKEN" "")

# Jellyfin Configuration
JELLYFIN_PUBLIC_URL=$(load_config "JELLYFIN_PUBLIC_URL" "")

# Hardware Transcoding
# Intel QuickSync device for hardware transcoding
TRANSCODE_DEVICE=/dev/dri

EOF
            ;;
        web)
            cat >> "$env_file" <<EOF
# Overseerr Configuration
OVERSEERR_API_KEY=$(load_config "OVERSEERR_API_KEY" "")

# Web Service Ports
OVERSEERR_PORT=5055
WIZARR_PORT=5690
ORGANIZR_PORT=9983
HOMEPAGE_PORT=3000

EOF
            ;;
        cloud)
            cat >> "$env_file" <<EOF
# Nextcloud Configuration
NEXTCLOUD_ADMIN_USER=$(load_config "NEXTCLOUD_ADMIN_USER" "admin")
NEXTCLOUD_ADMIN_PASSWORD=$(load_config "NEXTCLOUD_ADMIN_PASSWORD" "")
NEXTCLOUD_DB_PASSWORD=$(load_config "NEXTCLOUD_DB_PASSWORD" "")
NEXTCLOUD_TRUSTED_DOMAINS=$(load_config "NEXTCLOUD_TRUSTED_DOMAINS" "localhost")

# Collabora Configuration
COLLABORA_PASSWORD=$(load_config "COLLABORA_PASSWORD" "")
COLLABORA_DOMAIN=$(load_config "COLLABORA_DOMAIN" "localhost")

# Immich Configuration
IMMICH_DB_PASSWORD=$(load_config "IMMICH_DB_PASSWORD" "")

# Database Configuration
POSTGRES_USER=homelab
REDIS_PASSWORD=homelab-redis

EOF
            ;;
    esac

    # Set proper ownership
    local setup_user
    setup_user=$(load_config "SETUP_USER")
    sudo chown "${setup_user}:${setup_user}" "$env_file"
    sudo chmod 600 "$env_file"

    log_success "Created: $env_file"
}

# ============================================================================
# Interactive Configuration
# ============================================================================

interactive_container_setup() {
    log_step "Container Service Configuration"

    echo ""
    log_info "This will configure environment variables for all container services."
    log_info "You'll be prompted for passwords and configuration values."
    echo ""

    if ! prompt_yes_no "Proceed with container configuration?" "yes"; then
        return 1
    fi

    # Create base configuration
    create_base_env_config

    # Configure each service
    local setup_user
    setup_user=$(load_config "SETUP_USER")

    # Media stack
    local media_env="${CONTAINERS_BASE}/media/.env"
    if [[ -f "$media_env" ]]; then
        log_info "Media environment file already exists"
        if ! prompt_yes_no "Reconfigure media stack?" "no"; then
            log_info "Skipping media configuration"
        else
            configure_media_env "$media_env"
        fi
    else
        configure_media_env "$media_env"
    fi

    # Web stack
    local web_env="${CONTAINERS_BASE}/web/.env"
    if [[ -f "$web_env" ]]; then
        log_info "Web environment file already exists"
        if ! prompt_yes_no "Reconfigure web stack?" "no"; then
            log_info "Skipping web configuration"
        else
            configure_web_env "$web_env"
        fi
    else
        configure_web_env "$web_env"
    fi

    # Cloud stack
    local cloud_env="${CONTAINERS_BASE}/cloud/.env"
    if [[ -f "$cloud_env" ]]; then
        log_info "Cloud environment file already exists"
        if ! prompt_yes_no "Reconfigure cloud stack?" "no"; then
            log_info "Skipping cloud configuration"
        else
            configure_cloud_env "$cloud_env"
        fi
    else
        configure_cloud_env "$cloud_env"
    fi

    log_success "✓ Container configuration complete"
}

# ============================================================================
# Verification Functions
# ============================================================================

verify_container_setup() {
    log_step "Verifying Container Setup"

    local all_good=true

    for service in "${!SERVICES[@]}"; do
        local service_dir="${CONTAINERS_BASE}/${service}"
        local compose_file="${service_dir}/compose.yml"
        local env_file="${service_dir}/.env"

        log_info "Checking $service stack..."

        # Check compose file
        if [[ -f "$compose_file" ]]; then
            log_success "  compose.yml exists"
        else
            log_error "  compose.yml missing"
            all_good=false
        fi

        # Check env file
        if [[ -f "$env_file" ]]; then
            log_success "  .env exists"

            # Validate env file (basic check)
            if grep -q "PUID=" "$env_file"; then
                log_success "  .env contains required variables"
            else
                log_warning "  .env may be incomplete"
            fi
        else
            log_error "  .env missing"
            all_good=false
        fi
    done

    if $all_good; then
        log_success "✓ Container setup verification passed"
        return 0
    else
        log_error "✗ Container setup verification failed"
        return 1
    fi
}

# ============================================================================
# Summary Display
# ============================================================================

show_setup_summary() {
    log_step "Container Setup Summary"

    echo ""
    print_separator
    log_info "Service Configurations:"
    print_separator

    for service in "${!SERVICES[@]}"; do
        echo "${CONTAINERS_BASE}/${service}/"
        echo "  ├── compose.yml"
        echo "  └── .env"
    done

    print_separator
    echo ""

    log_info "Container services are ready for deployment"
    log_info "Environment files contain sensitive information - keep them secure!"
}

# ============================================================================
# Main Function
# ============================================================================

main() {
    require_root
    require_sudo

    print_header "UBlue uCore Homelab - Container Setup"

    # Check prerequisites
    if ! config_exists "SETUP_USER"; then
        log_error "User setup not completed. Run 01-user-setup.sh first."
        exit 1
    fi

    if ! check_marker "directory-setup-complete"; then
        log_error "Directory setup not completed. Run 02-directory-setup.sh first."
        exit 1
    fi

    # Check if already configured
    if check_marker "container-setup-complete"; then
        log_info "Container setup already completed"

        if prompt_yes_no "Reconfigure container setup?" "no"; then
            log_info "Reconfiguring containers..."
            remove_marker "container-setup-complete"
        else
            log_info "Skipping container setup"
            show_setup_summary
            exit 0
        fi
    fi

    # Find templates
    local template_dir
    if ! template_dir=$(find_compose_templates); then
        log_error "Cannot proceed without templates"
        exit 1
    fi

    # Check template files
    if ! check_template_files "$template_dir"; then
        log_error "Missing required template files"
        exit 1
    fi

    # Copy templates
    if ! copy_compose_templates "$template_dir"; then
        log_error "Failed to copy templates"
        exit 1
    fi

    # Interactive configuration
    if ! interactive_container_setup; then
        log_error "Container configuration failed"
        exit 1
    fi

    # Verify setup
    if ! verify_container_setup; then
        log_warning "Container setup verification failed"
    fi

    # Show summary
    show_setup_summary

    # Create completion marker
    create_marker "container-setup-complete"

    log_success "✓ Container setup completed successfully"
    echo ""
    log_info "Next step: Run 06-service-deployment.sh to deploy and start services"
}

# Run main function
main "$@"
