#!/usr/bin/env bash
#
# common-functions.sh
# Common functions library for UBlue uCore homelab setup scripts
#
# This file should be sourced by all setup scripts to provide
# consistent output formatting, error handling, and utility functions.

set -o errexit   # Exit on error
set -o nounset   # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# ============================================================================
# Color Definitions
# ============================================================================

if [[ -t 1 ]]; then
    COLOR_RESET="\033[0m"
    COLOR_BOLD="\033[1m"
    COLOR_RED="\033[31m"
    COLOR_GREEN="\033[32m"
    COLOR_YELLOW="\033[33m"
    COLOR_BLUE="\033[34m"
    COLOR_CYAN="\033[36m"
else
    COLOR_RESET=""
    COLOR_BOLD=""
    COLOR_RED=""
    COLOR_GREEN=""
    COLOR_YELLOW=""
    COLOR_BLUE=""
    COLOR_CYAN=""
fi

# ============================================================================
# Output Functions
# ============================================================================

log_info() {
    echo -e "${COLOR_BLUE}[INFO]${COLOR_RESET} $*"
}

log_success() {
    echo -e "${COLOR_GREEN}[✓]${COLOR_RESET} $*"
}

log_warning() {
    echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $*"
}

log_error() {
    echo -e "${COLOR_RED}[ERROR]${COLOR_RESET} $*" >&2
}

log_step() {
    echo -e "\n${COLOR_BOLD}${COLOR_CYAN}==>${COLOR_RESET} ${COLOR_BOLD}$*${COLOR_RESET}\n"
}

print_header() {
    local title="$1"
    local width=70
    echo ""
    echo -e "${COLOR_BOLD}${COLOR_CYAN}$(printf '=%.0s' $(seq 1 $width))${COLOR_RESET}"
    echo -e "${COLOR_BOLD}${COLOR_CYAN}  $title${COLOR_RESET}"
    echo -e "${COLOR_BOLD}${COLOR_CYAN}$(printf '=%.0s' $(seq 1 $width))${COLOR_RESET}"
    echo ""
}

print_separator() {
    echo -e "${COLOR_CYAN}$(printf -- '-%.0s' $(seq 1 70))${COLOR_RESET}"
}

# ============================================================================
# Configuration Management
# ============================================================================

CONFIG_FILE="${HOME}/.homelab-setup.conf"

save_config() {
    local key="$1"
    local value="$2"

    # Create config file if it doesn't exist
    touch "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"

    # Remove existing key and append new value
    sed -i "/^${key}=/d" "$CONFIG_FILE" 2>/dev/null || true
    echo "${key}=${value}" >> "$CONFIG_FILE"
}

load_config() {
    local key="$1"
    local default="${2:-}"

    if [[ -f "$CONFIG_FILE" ]]; then
        local value
        value=$(grep "^${key}=" "$CONFIG_FILE" 2>/dev/null | cut -d'=' -f2- || echo "")
        if [[ -n "$value" ]]; then
            echo "$value"
            return 0
        fi
    fi

    echo "$default"
}

config_exists() {
    local key="$1"
    if [[ -f "$CONFIG_FILE" ]] && grep -q "^${key}=" "$CONFIG_FILE" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# User Input Functions
# ============================================================================

prompt_yes_no() {
    local prompt="$1"
    local default="${2:-yes}"
    local yn

    if [[ "$default" == "yes" ]]; then
        prompt="${prompt} [Y/n]: "
    else
        prompt="${prompt} [y/N]: "
    fi

    while true; do
        read -r -p "$prompt" yn
        yn=${yn:-$default}
        case ${yn,,} in
            y|yes) return 0 ;;
            n|no) return 1 ;;
            *) echo "Please answer yes or no." ;;
        esac
    done
}

prompt_input() {
    local prompt="$1"
    local default="$2"
    local value

    if [[ -n "$default" ]]; then
        read -r -p "${prompt} [${default}]: " value
        echo "${value:-$default}"
    else
        while [[ -z "${value:-}" ]]; do
            read -r -p "${prompt}: " value
            if [[ -z "$value" ]]; then
                log_error "This field is required."
            fi
        done
        echo "$value"
    fi
}

prompt_password() {
    local prompt="$1"
    local password1
    local password2

    while true; do
        read -r -s -p "${prompt}: " password1
        echo ""
        read -r -s -p "Confirm password: " password2
        echo ""

        if [[ "$password1" == "$password2" ]]; then
            if [[ -n "$password1" ]]; then
                echo "$password1"
                return 0
            else
                log_error "Password cannot be empty."
            fi
        else
            log_error "Passwords do not match. Please try again."
        fi
    done
}

# ============================================================================
# Validation Functions
# ============================================================================

validate_ip() {
    local ip="$1"
    local valid_ip_regex='^([0-9]{1,3}\.){3}[0-9]{1,3}$'

    if [[ $ip =~ $valid_ip_regex ]]; then
        local IFS='.'
        local -a octets=($ip)
        for octet in "${octets[@]}"; do
            if ((octet > 255)); then
                return 1
            fi
        done
        return 0
    else
        return 1
    fi
}

validate_port() {
    local port="$1"
    if [[ "$port" =~ ^[0-9]+$ ]] && ((port >= 1 && port <= 65535)); then
        return 0
    else
        return 1
    fi
}

validate_path() {
    local path="$1"
    if [[ "$path" =~ ^/ ]]; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# System Detection Functions
# ============================================================================

check_ucore() {
    if command -v rpm-ostree &> /dev/null; then
        return 0
    else
        return 1
    fi
}

check_command() {
    local cmd="$1"
    if command -v "$cmd" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# Container Runtime Functions
# ============================================================================

detect_container_runtime() {
    # Check what's available and configured
    local runtime=$(load_config "CONTAINER_RUNTIME" "")

    if [[ -n "$runtime" ]]; then
        echo "$runtime"
        return 0
    fi

    # Auto-detect based on what's available
    if check_command podman; then
        echo "podman"
    elif check_command docker; then
        echo "docker"
    else
        return 1
    fi
}

get_compose_command() {
    local runtime=$(detect_container_runtime)

    case $runtime in
        podman)
            if check_command podman-compose; then
                echo "podman-compose"
            else
                echo "podman compose"
            fi
            ;;
        docker)
            if check_command docker-compose; then
                echo "docker-compose"
            else
                echo "docker compose"
            fi
            ;;
        *)
            return 1
            ;;
    esac
}

select_container_runtime() {
    log_step "Container Runtime Selection"

    local available_runtimes=()

    # Check what's available
    if check_command podman; then
        available_runtimes+=("podman")
    fi

    if check_command docker; then
        available_runtimes+=("docker")
    fi

    if [[ ${#available_runtimes[@]} -eq 0 ]]; then
        log_error "No container runtime found (podman or docker)"
        return 1
    fi

    # If only one is available, use it
    if [[ ${#available_runtimes[@]} -eq 1 ]]; then
        local runtime="${available_runtimes[0]}"
        log_success "Using ${runtime} (only runtime available)"
        save_config "CONTAINER_RUNTIME" "$runtime"
        return 0
    fi

    # Ask user to choose
    echo ""
    log_info "Multiple container runtimes detected:"
    echo "  1. Podman (rootless, recommended for UBlue uCore)"
    echo "  2. Docker"
    echo ""

    local choice
    while true; do
        read -r -p "Select container runtime [1]: " choice
        choice=${choice:-1}

        case $choice in
            1|podman|Podman)
                save_config "CONTAINER_RUNTIME" "podman"
                log_success "Using Podman"
                return 0
                ;;
            2|docker|Docker)
                save_config "CONTAINER_RUNTIME" "docker"
                log_success "Using Docker"
                return 0
                ;;
            *)
                log_error "Invalid choice. Please enter 1 or 2."
                ;;
        esac
    done
}

check_package() {
    local pkg="$1"
    if rpm -q "$pkg" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

check_systemd_service() {
    local service="$1"
    if systemctl list-unit-files | grep -q "^${service}"; then
        return 0
    else
        return 1
    fi
}

get_service_location() {
    local service="$1"

    # Check in order of override precedence
    if [[ -f "/etc/systemd/system/${service}" ]]; then
        echo "/etc/systemd/system/${service}"
    elif [[ -f "/usr/lib/systemd/system/${service}" ]]; then
        echo "/usr/lib/systemd/system/${service}"
    elif [[ -f "/lib/systemd/system/${service}" ]]; then
        echo "/lib/systemd/system/${service}"
    else
        return 1
    fi
}

# ============================================================================
# Network Functions
# ============================================================================

test_connectivity() {
    local host="$1"
    local timeout="${2:-5}"

    if ping -c 1 -W "$timeout" "$host" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

get_default_interface() {
    ip route | grep default | awk '{print $5}' | head -n1
}

get_interface_ip() {
    local interface="$1"
    ip addr show "$interface" | grep "inet " | awk '{print $2}' | cut -d/ -f1 | head -n1
}

# ============================================================================
# File System Functions
# ============================================================================

ensure_directory() {
    local dir="$1"
    local owner="${2:-}"
    local perms="${3:-755}"

    if [[ ! -d "$dir" ]]; then
        if sudo mkdir -p "$dir"; then
            if [[ -n "$owner" ]]; then
                sudo chown "$owner" "$dir"
            fi
            sudo chmod "$perms" "$dir"
            log_success "Created directory: $dir"
            return 0
        else
            log_error "Failed to create directory: $dir"
            return 1
        fi
    else
        log_info "Directory already exists: $dir"
        return 0
    fi
}

backup_file() {
    local file="$1"
    local backup="${file}.backup.$(date +%Y%m%d_%H%M%S)"

    if [[ -f "$file" ]]; then
        if sudo cp "$file" "$backup"; then
            log_success "Backed up: $file → $backup"
            return 0
        else
            log_error "Failed to backup: $file"
            return 1
        fi
    fi
    return 0
}

# ============================================================================
# Error Handling
# ============================================================================

handle_error() {
    local exit_code=$?
    local line_number=$1
    log_error "Script failed at line $line_number with exit code $exit_code"
    exit "$exit_code"
}

require_root() {
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should NOT be run as root"
        log_info "Please run as a regular user. Sudo will be used when needed."
        exit 1
    fi
}

require_sudo() {
    if ! sudo -n true 2>/dev/null; then
        log_info "This script requires sudo privileges."
        sudo -v || {
            log_error "Failed to obtain sudo privileges"
            exit 1
        }
    fi
}

# ============================================================================
# Progress Functions
# ============================================================================

show_progress() {
    local current=$1
    local total=$2
    local message="$3"

    local percent=$((current * 100 / total))
    local filled=$((current * 50 / total))
    local empty=$((50 - filled))

    printf "\r[%s%s] %d%% %s" \
        "$(printf '#%.0s' $(seq 1 $filled))" \
        "$(printf ' %.0s' $(seq 1 $empty))" \
        "$percent" \
        "$message"

    if [[ $current -eq $total ]]; then
        echo ""
    fi
}

spinner() {
    local pid=$1
    local message="${2:-Processing...}"
    local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local i=0

    while kill -0 "$pid" 2>/dev/null; do
        i=$(((i + 1) % 10))
        printf "\r${COLOR_CYAN}${spin:$i:1}${COLOR_RESET} %s" "$message"
        sleep 0.1
    done

    printf "\r%s\r" "$(printf ' %.0s' $(seq 1 $((${#message} + 3))))"
}

# ============================================================================
# Docker/Podman User Detection
# ============================================================================

get_user_uid() {
    local username="$1"
    id -u "$username" 2>/dev/null || echo "1000"
}

get_user_gid() {
    local username="$1"
    id -g "$username" 2>/dev/null || echo "1000"
}

detect_timezone() {
    if [[ -f /etc/localtime ]]; then
        timedatectl show --property=Timezone --value 2>/dev/null || echo "America/Chicago"
    else
        echo "America/Chicago"
    fi
}

# ============================================================================
# Template Functions
# ============================================================================

check_template_location() {
    local template="$1"

    # Check in home setup directory first (from first boot)
    if [[ -f "${HOME}/setup/${template}" ]]; then
        echo "${HOME}/setup/${template}"
        return 0
    fi

    # Check in /usr/share (baked into image)
    if [[ -f "/usr/share/${template}" ]]; then
        echo "/usr/share/${template}"
        return 0
    fi

    return 1
}

# ============================================================================
# Systemd Functions
# ============================================================================

reload_systemd() {
    sudo systemctl daemon-reload
}

enable_service() {
    local service="$1"
    if sudo systemctl enable "$service"; then
        log_success "Enabled: $service"
        return 0
    else
        log_error "Failed to enable: $service"
        return 1
    fi
}

start_service() {
    local service="$1"
    if sudo systemctl start "$service"; then
        log_success "Started: $service"
        return 0
    else
        log_error "Failed to start: $service"
        return 1
    fi
}

service_status() {
    local service="$1"
    sudo systemctl is-active --quiet "$service"
}

# ============================================================================
# Marker File Functions
# ============================================================================

MARKER_DIR="${HOME}/.local/homelab-setup"

create_marker() {
    local marker="$1"
    mkdir -p "$MARKER_DIR"
    touch "${MARKER_DIR}/${marker}"
}

check_marker() {
    local marker="$1"
    [[ -f "${MARKER_DIR}/${marker}" ]]
}

remove_marker() {
    local marker="$1"
    rm -f "${MARKER_DIR}/${marker}"
}

# ============================================================================
# Export Functions
# ============================================================================

export -f log_info log_success log_warning log_error log_step
export -f print_header print_separator
export -f save_config load_config config_exists
export -f prompt_yes_no prompt_input prompt_password
export -f validate_ip validate_port validate_path
export -f check_ucore check_command check_package check_systemd_service get_service_location
export -f detect_container_runtime get_compose_command select_container_runtime
export -f test_connectivity get_default_interface get_interface_ip
export -f ensure_directory backup_file
export -f handle_error require_root require_sudo
export -f show_progress spinner
export -f get_user_uid get_user_gid detect_timezone
export -f check_template_location
export -f reload_systemd enable_service start_service service_status
export -f create_marker check_marker remove_marker
