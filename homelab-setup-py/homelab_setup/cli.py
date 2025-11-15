"""
Main CLI entry point for homelab setup scripts.

Provides a unified command-line interface for all setup operations.
"""

import sys
import click
from rich.console import Console

from . import __version__
from .utils import log_error

console = Console()


@click.group()
@click.version_option(version=__version__)
@click.pass_context
def cli(ctx):
    """UBlue uCore Homelab Setup Scripts (Python Edition)."""
    ctx.ensure_object(dict)


@cli.command()
def preflight():
    """Run pre-flight system checks."""
    from .preflight import run_preflight

    sys.exit(run_preflight())


@cli.command()
def user():
    """Configure user account for container management."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
def directories():
    """Create directory structure for containers and data."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
def wireguard():
    """Configure WireGuard VPN."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
def nfs():
    """Configure NFS mounts."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
def containers():
    """Configure container services."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
def deploy():
    """Deploy and start all services."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
@click.option("--all", "-a", is_flag=True, help="Run all diagnostics")
@click.option("--services", "-s", is_flag=True, help="Check services only")
@click.option("--network", "-n", is_flag=True, help="Check network only")
@click.option("--storage", "-d", is_flag=True, help="Check storage only")
@click.option("--logs", "-l", is_flag=True, help="Collect diagnostic logs")
def troubleshoot(all, services, network, storage, logs):
    """Run system diagnostics and troubleshooting."""
    log_error("Not yet implemented - use bash scripts for now")
    sys.exit(1)


@cli.command()
def run_all():
    """Run all setup steps interactively."""
    from .preflight import run_preflight

    console.print("\n[bold cyan]Starting full homelab setup...[/bold cyan]\n")

    # Run preflight checks
    console.print("[bold]Step 1/7:[/bold] Preflight checks")
    if run_preflight() != 0:
        console.print("\n[bold red]Preflight checks failed. Aborting setup.[/bold red]\n")
        sys.exit(1)

    console.print("\n[bold green]✓ Preflight checks passed[/bold green]\n")

    # TODO: Add remaining setup steps
    log_error("Remaining setup steps not yet implemented - use bash scripts")
    sys.exit(1)


def main():
    """Main entry point."""
    try:
        cli(obj={})
    except KeyboardInterrupt:
        console.print("\n\n[yellow]Setup interrupted by user[/yellow]\n")
        sys.exit(130)
    except Exception as e:
        console.print(f"\n[bold red]Fatal error:[/bold red] {e}\n")
        sys.exit(1)


if __name__ == "__main__":
    main()
