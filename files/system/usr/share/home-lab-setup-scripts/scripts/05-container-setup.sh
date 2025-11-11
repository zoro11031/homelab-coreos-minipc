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

# Files to exclude when discovering stacks (patterns)
EXCLUDE_PATTERNS=(
    ".*"              # Hidden files
    "*.example"       # Example files
    "README*"         # Documentation files
    "*.md"            # Markdown files
)

# Service configurations (dynamically populated)
declare -A SERVICES=()

# Selected services to setup (will be populated by user selection)
declare -a SELECTED_SERVICES=()

# ============================================================================
# Template Detection Functions
# ============================================================================

# Helper function to count YAML files in a directory
count_yaml_files() {
    local dir="$1"
    find "$dir" -maxdepth 1 -type f \( -name "*.yml" -o -name "*.yaml" \) 2>/dev/null | wc -l
}

# Helper function to check if directory has YAML files

find_compose_templates() {
    log_step "Locating Compose Templates"

    # Check home setup directory first
    if [[ -d "$TEMPLATE_DIR_HOME" ]]; then
        log_info "Checking: $TEMPLATE_DIR_HOME"
        local yaml_count
        yaml_count=$(count_yaml_files "$TEMPLATE_DIR_HOME")
        if [[ $yaml_count -gt 0 ]]; then
            log_success "Found templates in: $TEMPLATE_DIR_HOME ($yaml_count YAML file(s))"
            echo "$TEMPLATE_DIR_HOME"
            return 0
        else
            log_warning "Directory exists but contains no YAML files: $TEMPLATE_DIR_HOME"
            log_info "Checking fallback location..."
        fi
    fi

    # Check /usr/share as fallback
    if [[ -d "$TEMPLATE_DIR_USR" ]]; then
        log_info "Checking: $TEMPLATE_DIR_USR"
        local yaml_count
        yaml_count=$(count_yaml_files "$TEMPLATE_DIR_USR")
        if [[ $yaml_count -gt 0 ]]; then
            log_success "Found templates in: $TEMPLATE_DIR_USR ($yaml_count YAML file(s))"
            echo "$TEMPLATE_DIR_USR"
            return 0
        else
            log_warning "Directory exists but contains no YAML files: $TEMPLATE_DIR_USR"
        fi
    fi

    log_error "No compose templates found in any location"
    log_info "Searched locations:"
    log_info "  - $TEMPLATE_DIR_HOME"
    log_info "  - $TEMPLATE_DIR_USR"
    log_info ""
    log_info "Expected to find .yml or .yaml files in one of these directories"
    return 1
}

discover_available_stacks() {
    local template_dir="$1"

    log_step "Discovering Available Container Stacks"

    # Clear existing services
    SERVICES=()

    log_info "Scanning directory: $template_dir"

    # Count total YAML files before filtering
    local total_yaml_count
    total_yaml_count=$(count_yaml_files "$template_dir")
    log_info "Found $total_yaml_count total YAML file(s) in directory"

    # Find all .yml and .yaml files in the template directory
    local count=0
    local excluded_count=0
    while IFS= read -r -d '' yaml_file; do
        local filename
        filename=$(basename "$yaml_file")

        # Check if filename matches any exclude pattern
        local should_exclude=false
        local matched_pattern=""
        for pattern in "${EXCLUDE_PATTERNS[@]}"; do
            # shellcheck disable=SC2053
            if [[ "$filename" == $pattern ]]; then
                should_exclude=true
                matched_pattern="$pattern"
                break
            fi
        done

        if $should_exclude; then
            log_info "Excluding: $filename (matches pattern: $matched_pattern)"
            ((excluded_count++))
            continue
        fi

        # Get service name (filename without extension)
        local service_name="${filename%.yml}"
        service_name="${service_name%.yaml}"

        # Add to SERVICES array
        SERVICES["$service_name"]="$filename"
        log_success "Found stack: $service_name ($filename)"
        ((count++))
    done < <(find "$template_dir" -maxdepth 1 -type f \( -name "*.yml" -o -name "*.yaml" \) -print0 2>/dev/null)

    if [[ $count -eq 0 ]]; then
        log_error "No valid compose stack files discovered"
        log_error "Directory checked: $template_dir"
        log_info "Total YAML files found: $total_yaml_count"
        log_info "Files excluded by patterns: $excluded_count"
        log_info ""
        log_info "Exclude patterns:"
        for pattern in "${EXCLUDE_PATTERNS[@]}"; do
            log_info "  - $pattern"
        done
        log_info ""
        log_info "Stack files should be named like: media.yml, web.yml, cloud.yml"
        log_info "Excluded files: .env.example, README.md, .hidden files"
        return 1
    fi

    log_success "Discovered $count valid container stack(s) (excluded $excluded_count file(s))"
    return 0
}

check_template_files() {
    local template_dir="$1"

    log_info "Checking selected template files..."

    local all_found=true

    # Only check selected services, not all discovered services
    for service in "${SELECTED_SERVICES[@]}"; do
        local template_file="${SERVICES[$service]}"
        local file_path="${template_dir}/${template_file}"

        if [[ -f "$file_path" ]]; then
            log_success "Found: $template_file"
        else
            log_error "Missing: $template_file"
            all_found=false
        fi
    done

    # Check for .env.example (optional but helpful)
    if [[ -f "${template_dir}/.env.example" ]]; then
        log_success "Found: .env.example"
    else
        log_warning "Missing: .env.example (will create default)"
    fi

    $all_found
}

# ============================================================================
# Stack Selection Functions
# ============================================================================

select_container_stacks() {
    log_step "Container Stack Selection"

    echo "" >&2
    log_info "Available container stacks:"
    echo "" >&2

    # Create sorted array of service names for consistent ordering
    local -a unsorted_services=()
    for service in "${!SERVICES[@]}"; do
        unsorted_services+=("$service")
    done
    local -a service_list=()
    mapfile -t service_list < <(printf '%s\n' "${unsorted_services[@]}" | sort)

    # Display available stacks
    local i=1
    for service in "${service_list[@]}"; do
        echo "  $i) ${service} (${SERVICES[$service]})" >&2
        ((i++))
    done
    echo "  $i) All stacks" >&2
    echo "" >&2

    # Prompt for selection
    log_info "Select which container stacks to setup:"
    log_info "  - Enter numbers separated by spaces (e.g., '1 3' for first and third)"
    log_info "  - Enter '$i' to setup all stacks"
    log_info "  - Press Enter to setup all stacks (default)"
    echo "" >&2

    local selection
    selection=$(prompt_with_color "Your selection")

    # Default to all if empty
    if [[ -z "$selection" ]]; then
        selection="$i"
    fi

    # Clear selected services array
    SELECTED_SERVICES=()

    # Parse selection
    if [[ "$selection" == "$i" ]]; then
        # All stacks selected
        log_success "Selected: All stacks"
        SELECTED_SERVICES=("${service_list[@]}")
    else
        # Individual stacks selected - use associative array to prevent duplicates
        declare -A selected_map=()

        for num in $selection; do
            # Validate number is numeric
            if [[ ! "$num" =~ ^[0-9]+$ ]]; then
                log_warning "Invalid selection: $num (not a number, skipping)"
                continue
            fi
            
            # Explicitly check for "All stacks" option number
            if [[ "$num" -eq "$i" ]]; then
                log_warning "Cannot combine 'All stacks' option with individual stack selections. Either enter the number for 'All stacks' (or press Enter) alone, or select specific stack numbers."
                continue
            fi
            
            # Validate number is within valid range
            if [[ "$num" -lt 1 ]] || [[ "$num" -gt ${#service_list[@]} ]]; then
                log_warning "Invalid selection: $num (valid range: 1-${#service_list[@]}, skipping)"
                continue
            fi

            # Add to selected services map (arrays are 0-indexed)
            local idx=$((num - 1))
            local service_name="${service_list[$idx]}"
            selected_map["$service_name"]=1
        done

        # Convert map to array
        for service in "${!selected_map[@]}"; do
            SELECTED_SERVICES+=("$service")
        done

        # Show selected stacks
        if [[ ${#SELECTED_SERVICES[@]} -eq 0 ]]; then
            log_error "No valid stacks selected"
            return 1
        fi

        log_success "Selected stacks:"
        for service in "${SELECTED_SERVICES[@]}"; do
            log_info "  - $service"
        done
    fi

    echo "" >&2

    # Save selected services to config for potential re-runs
    save_config "SELECTED_SERVICES" "${SELECTED_SERVICES[*]}"

    return 0
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

    # Only copy selected services
    for service in "${SELECTED_SERVICES[@]}"; do
        local template_file="${SERVICES[$service]}"
        local src="${template_dir}/${template_file}"
        local dst_dir="${CONTAINERS_BASE}/${service}"
        local dst="${dst_dir}/compose.yml"

        # Ensure destination directory exists
        if [[ ! -d "$dst_dir" ]]; then
            log_warning "Destination directory does not exist: $dst_dir"
            log_info "Creating directory: $dst_dir"
            if ! sudo mkdir -p "$dst_dir"; then
                log_error "Failed to create directory: $dst_dir"
                return 1
            fi
            sudo chown "${setup_user}:${setup_user}" "$dst_dir"
            sudo chmod 755 "$dst_dir"
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
    if [[ -n "$jellyfin_url" ]]; then
        save_config "JELLYFIN_PUBLIC_URL" "$jellyfin_url"
    fi

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

configure_generic_env() {
    local env_file="$1"
    local service="$2"

    log_step "Configuring ${service^} Stack Environment"

    log_info "No specific configuration prompts for this stack."
    log_info "Creating environment file with base configuration (PUID, PGID, timezone)."

    # Create env file with just base configuration
    create_env_file "$env_file" "$service"
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
    if ! cat > "$env_file" <<EOF
# UBlue uCore Homelab - ${service^} Stack Environment
# Generated: $(date)

# User/Group Configuration
PUID=$puid
PGID=$pgid
TZ=$tz

# Paths
APPDATA_PATH=$appdata_path

EOF
    then
        log_error "Failed to create base environment file: $env_file"
        return 1
    fi

    # Add service-specific variables
    case $service in
        media)
            if ! cat >> "$env_file" <<EOF
# Plex Configuration
PLEX_CLAIM_TOKEN=$(load_config "PLEX_CLAIM_TOKEN" "")

# Jellyfin Configuration
JELLYFIN_PUBLIC_URL=$(load_config "JELLYFIN_PUBLIC_URL" "")

# Hardware Transcoding
# Intel QuickSync device for hardware transcoding
TRANSCODE_DEVICE=/dev/dri

EOF
            then
                log_error "Failed to add media configuration to: $env_file"
                return 1
            fi
            ;;
        web)
            if ! cat >> "$env_file" <<EOF
# Overseerr Configuration
OVERSEERR_API_KEY=$(load_config "OVERSEERR_API_KEY" "")

# Web Service Ports
OVERSEERR_PORT=5055
WIZARR_PORT=5690
ORGANIZR_PORT=9983
HOMEPAGE_PORT=3000

EOF
            then
                log_error "Failed to add web configuration to: $env_file"
                return 1
            fi
            ;;
        cloud)
            if ! cat >> "$env_file" <<EOF
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
            then
                log_error "Failed to add cloud configuration to: $env_file"
                return 1
            fi
            ;;
    esac

    # Set proper ownership
    local setup_user
    setup_user=$(load_config "SETUP_USER")

    if ! sudo chown "${setup_user}:${setup_user}" "$env_file"; then
        log_error "Failed to set ownership on: $env_file"
        return 1
    fi

    if ! sudo chmod 600 "$env_file"; then
        log_error "Failed to set permissions on: $env_file"
        return 1
    fi

    log_success "Created: $env_file"
}

# ============================================================================
# Interactive Configuration
# ============================================================================

interactive_container_setup() {
    log_step "Container Service Configuration"

    echo "" >&2
    log_info "This will configure environment variables for selected container services."
    log_info "You'll be prompted for passwords and configuration values."
    echo "" >&2

    if ! prompt_yes_no "Proceed with container configuration?" "yes"; then
        return 1
    fi

    # Create base configuration
    create_base_env_config

    # Configure each selected service
    for service in "${SELECTED_SERVICES[@]}"; do
        local env_file="${CONTAINERS_BASE}/${service}/.env"

        # Check if env file already exists
        if [[ -f "$env_file" ]]; then
            log_info "${service^} environment file already exists"
            if ! prompt_yes_no "Reconfigure ${service} stack?" "no"; then
                log_info "Skipping ${service} configuration"
                continue
            fi
        fi

        # Configure based on service type
        case $service in
            media)
                configure_media_env "$env_file"
                ;;
            web)
                configure_web_env "$env_file"
                ;;
            cloud)
                configure_cloud_env "$env_file"
                ;;
            *)
                # Generic configuration for unknown stacks
                log_info "Configuring ${service} stack with base settings"
                configure_generic_env "$env_file" "$service"
                ;;
        esac
    done

    log_success "✓ Container configuration complete"
}

# ============================================================================
# Verification Functions
# ============================================================================

verify_container_setup() {
    log_step "Verifying Container Setup"

    local all_good=true

    # Only verify selected services
    for service in "${SELECTED_SERVICES[@]}"; do
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

    echo "" >&2
    print_separator
    log_info "Service Configurations:"
    print_separator

    # Only show selected services
    for service in "${SELECTED_SERVICES[@]}"; do
        echo "${CONTAINERS_BASE}/${service}/" >&2
        echo "  ├── compose.yml" >&2
        echo "  └── .env" >&2
    done

    print_separator
    echo "" >&2

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

            # Load previous selection for summary display
            local template_dir
            if template_dir=$(find_compose_templates); then
                if ! discover_available_stacks "$template_dir" > /dev/null 2>&1; then
                    log_warning "Could not discover available container stacks for summary display. The summary may be incomplete."
                fi

                # Try to load previous selection
                local saved_services
                saved_services=$(load_config "SELECTED_SERVICES" "")
                if [[ -n "$saved_services" ]]; then
                    read -ra SELECTED_SERVICES <<< "$saved_services"
                else
                    # Fallback: show all configured services if no saved selection
                    log_info "Previous selection not found - showing all configured stacks"
                    for service in "${!SERVICES[@]}"; do
                        SELECTED_SERVICES+=("$service")
                    done
                fi

                show_setup_summary
            fi
            exit 0
        fi
    fi

    # Find templates
    local template_dir
    if ! template_dir=$(find_compose_templates); then
        log_error "Cannot proceed without compose templates"
        echo "" >&2
        log_info "Troubleshooting tips:"
        log_info "  1. Ensure templates are in ~/setup/compose-setup/ or /usr/share/compose-setup/"
        log_info "  2. Templates should be .yml or .yaml files (e.g., media.yml, web.yml)"
        log_info "  3. Check that the template files are readable"
        exit 1
    fi

    # Discover available stacks
    if ! discover_available_stacks "$template_dir"; then
        log_error "Failed to discover container stacks"
        echo "" >&2
        log_info "Troubleshooting tips:"
        log_info "  1. Check if the template directory contains .yml/.yaml files"
        log_info "  2. Ensure files are not excluded by patterns (.example, .*, README*, *.md)"
        log_info "  3. Verify file permissions allow reading the template files"
        log_info "  4. Look for errors in the detailed output above"
        exit 1
    fi

    # Let user select which stacks to setup
    if ! select_container_stacks; then
        log_error "Stack selection cancelled or failed"
        exit 1
    fi

    # Check template files for selected stacks
    if ! check_template_files "$template_dir"; then
        log_error "Missing required template files"
        exit 1
    fi

    # Copy templates for selected stacks
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
    echo "" >&2
    log_info "Next step: Run 06-service-deployment.sh to deploy and start services"
}

# Run main function
main "$@"
