"""
Common utility functions for homelab setup.

Provides logging, subprocess execution, validation, and helper functions.
"""

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Optional, List, Tuple

from rich.console import Console
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn

# Rich console for styled output
console = Console(stderr=True)


# ============================================================================
# Logging and Output Functions
# ============================================================================


def log_info(message: str) -> None:
    """Log an informational message."""
    console.print(f"[blue][INFO][/blue] {message}")


def log_success(message: str) -> None:
    """Log a success message."""
    console.print(f"[green]✓[/green] {message}")


def log_warning(message: str) -> None:
    """Log a warning message."""
    console.print(f"[yellow][WARNING][/yellow] {message}")


def log_error(message: str) -> None:
    """Log an error message."""
    console.print(f"[red][ERROR][/red] {message}")


def log_step(message: str) -> None:
    """Log a major step/section header."""
    console.print()
    console.rule(f"[bold cyan]{message}[/bold cyan]", style="cyan")
    console.print()


def print_header(title: str) -> None:
    """Print a formatted header."""
    console.print()
    console.print("=" * 70, style="bold cyan")
    console.print(f"  {title}", style="bold cyan")
    console.print("=" * 70, style="bold cyan")
    console.print()


def print_separator() -> None:
    """Print a horizontal separator."""
    console.print("-" * 70, style="cyan")


# ============================================================================
# Subprocess Execution
# ============================================================================


class CommandError(Exception):
    """Exception raised when a command fails."""

    def __init__(self, cmd: str, returncode: int, stderr: str = ""):
        self.cmd = cmd
        self.returncode = returncode
        self.stderr = stderr
        super().__init__(f"Command failed: {cmd} (exit code {returncode})")


def run_command(
    cmd: List[str],
    check: bool = True,
    capture_output: bool = True,
    sudo: bool = False,
    **kwargs,
) -> subprocess.CompletedProcess:
    """
    Run a command and return the result.

    Args:
        cmd: Command and arguments as a list
        check: Raise exception if command fails
        capture_output: Capture stdout/stderr
        sudo: Run command with sudo
        **kwargs: Additional arguments to subprocess.run

    Returns:
        CompletedProcess object

    Raises:
        CommandError: If command fails and check=True
    """
    if sudo:
        cmd = ["sudo"] + cmd

    try:
        result = subprocess.run(
            cmd,
            check=check,
            capture_output=capture_output,
            text=True,
            **kwargs,
        )
        return result
    except subprocess.CalledProcessError as e:
        if check:
            raise CommandError(
                cmd=" ".join(cmd),
                returncode=e.returncode,
                stderr=e.stderr if e.stderr else "",
            )
        raise


def check_command(command: str) -> bool:
    """Check if a command exists in PATH."""
    try:
        run_command(["command", "-v", command], check=True, capture_output=True)
        return True
    except (CommandError, subprocess.CalledProcessError):
        return False


# ============================================================================
# File System Functions
# ============================================================================


def ensure_directory(
    path: Path,
    owner: Optional[str] = None,
    mode: int = 0o755,
    sudo: bool = False,
) -> bool:
    """
    Ensure a directory exists with proper ownership and permissions.

    Args:
        path: Directory path
        owner: Owner in format "user:group" (optional)
        mode: Permission mode (default: 0o755)
        sudo: Use sudo for creation

    Returns:
        True if successful, False otherwise
    """
    try:
        if not path.exists():
            if sudo:
                run_command(["mkdir", "-p", str(path)], sudo=True)
            else:
                path.mkdir(parents=True, exist_ok=True)

            # Set ownership if specified
            if owner and sudo:
                run_command(["chown", owner, str(path)], sudo=True)

            # Set permissions
            if sudo:
                run_command(["chmod", oct(mode)[2:], str(path)], sudo=True)
            else:
                path.chmod(mode)

            log_success(f"Created directory: {path}")
        else:
            log_info(f"Directory already exists: {path}")

        return True
    except Exception as e:
        log_error(f"Failed to create directory {path}: {e}")
        return False


def backup_file(file_path: Path, sudo: bool = False) -> Optional[Path]:
    """
    Create a backup of a file with timestamp.

    Args:
        file_path: File to backup
        sudo: Use sudo for backup

    Returns:
        Path to backup file, or None if failed
    """
    if not file_path.exists():
        return None

    from datetime import datetime

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_path = file_path.with_suffix(f".backup.{timestamp}")

    try:
        if sudo:
            run_command(["cp", str(file_path), str(backup_path)], sudo=True)
        else:
            import shutil

            shutil.copy2(file_path, backup_path)

        log_success(f"Backed up: {file_path} → {backup_path}")
        return backup_path
    except Exception as e:
        log_error(f"Failed to backup {file_path}: {e}")
        return None


# ============================================================================
# Validation Functions
# ============================================================================


def validate_ip(ip: str) -> bool:
    """Validate an IPv4 address."""
    pattern = re.compile(r"^(\d{1,3}\.){3}\d{1,3}$")
    if not pattern.match(ip):
        return False

    octets = [int(x) for x in ip.split(".")]
    return all(0 <= octet <= 255 for octet in octets)


def validate_port(port: int) -> bool:
    """Validate a port number."""
    return 1 <= port <= 65535


def validate_path(path: str) -> bool:
    """Validate that a path is absolute."""
    return path.startswith("/")


# ============================================================================
# Network Functions
# ============================================================================


def test_connectivity(host: str, timeout: int = 5) -> bool:
    """Test network connectivity to a host using ping."""
    try:
        run_command(
            ["ping", "-c", "1", "-W", str(timeout), host],
            check=True,
            capture_output=True,
        )
        return True
    except (CommandError, subprocess.CalledProcessError):
        return False


def get_default_interface() -> Optional[str]:
    """Get the default network interface."""
    try:
        result = run_command(["ip", "route"], check=True)
        for line in result.stdout.splitlines():
            if line.startswith("default"):
                parts = line.split()
                if len(parts) >= 5:
                    return parts[4]
        return None
    except Exception:
        return None


def get_interface_ip(interface: str) -> Optional[str]:
    """Get the IP address of a network interface."""
    try:
        result = run_command(["ip", "addr", "show", interface], check=True)
        for line in result.stdout.splitlines():
            line = line.strip()
            if line.startswith("inet "):
                parts = line.split()
                if len(parts) >= 2:
                    return parts[1].split("/")[0]
        return None
    except Exception:
        return None


# ============================================================================
# User Functions
# ============================================================================


def get_user_uid(username: str) -> int:
    """Get the UID of a user."""
    try:
        result = run_command(["id", "-u", username], check=True)
        return int(result.stdout.strip())
    except Exception:
        return 1000


def get_user_gid(username: str) -> int:
    """Get the GID of a user."""
    try:
        result = run_command(["id", "-g", username], check=True)
        return int(result.stdout.strip())
    except Exception:
        return 1000


def detect_timezone() -> str:
    """Detect the system timezone."""
    try:
        result = run_command(
            ["timedatectl", "show", "--property=Timezone", "--value"],
            check=True,
        )
        return result.stdout.strip() or "America/Chicago"
    except Exception:
        return "America/Chicago"


# ============================================================================
# Permission Checks
# ============================================================================


def require_not_root() -> None:
    """Ensure the script is not run as root."""
    if os.geteuid() == 0:
        log_error("This script should NOT be run as root")
        log_info("Please run as a regular user. Sudo will be used when needed.")
        sys.exit(1)


def require_sudo() -> None:
    """Ensure sudo access is available."""
    try:
        # Check if we can run sudo without password
        result = run_command(["sudo", "-n", "true"], check=False, capture_output=True)
        if result.returncode != 0:
            # Try to get sudo access
            log_info("This script requires sudo privileges.")
            run_command(["sudo", "-v"], check=True)
    except Exception:
        log_error("Failed to obtain sudo privileges")
        sys.exit(1)


# ============================================================================
# Progress Display
# ============================================================================


def create_progress() -> Progress:
    """Create a Rich progress bar."""
    return Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        BarColumn(),
        TextColumn("[progress.percentage]{task.percentage:>3.0f}%"),
        console=console,
    )
