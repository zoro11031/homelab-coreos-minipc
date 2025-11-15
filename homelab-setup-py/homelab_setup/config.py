"""
Configuration management for homelab setup.

Handles persistent configuration storage and retrieval.
"""

import os
from pathlib import Path
from typing import Optional, Dict, Any
import configparser


class Config:
    """Configuration manager for homelab setup."""

    def __init__(self, config_file: Optional[Path] = None):
        """
        Initialize configuration manager.

        Args:
            config_file: Path to configuration file (default: ~/.homelab-setup.conf)
        """
        if config_file is None:
            self.config_file = Path.home() / ".homelab-setup.conf"
        else:
            self.config_file = config_file

        self.marker_dir = Path.home() / ".local" / "homelab-setup"
        self.marker_dir.mkdir(parents=True, exist_ok=True)

        # Ensure config file exists with proper permissions
        if not self.config_file.exists():
            self.config_file.touch(mode=0o600)
        else:
            self.config_file.chmod(0o600)

        self._config = configparser.ConfigParser()
        self._load()

    def _load(self) -> None:
        """Load configuration from file."""
        if self.config_file.exists():
            self._config.read(self.config_file)

        # Ensure default section exists
        if not self._config.has_section("homelab"):
            self._config.add_section("homelab")

    def _save(self) -> None:
        """Save configuration to file."""
        with open(self.config_file, "w") as f:
            self._config.write(f)
        self.config_file.chmod(0o600)

    def set(self, key: str, value: str, section: str = "homelab") -> None:
        """
        Set a configuration value.

        Args:
            key: Configuration key
            value: Configuration value
            section: Configuration section (default: homelab)
        """
        if not self._config.has_section(section):
            self._config.add_section(section)

        self._config.set(section, key, str(value))
        self._save()

    def get(self, key: str, default: str = "", section: str = "homelab") -> str:
        """
        Get a configuration value.

        Args:
            key: Configuration key
            default: Default value if key doesn't exist
            section: Configuration section (default: homelab)

        Returns:
            Configuration value or default
        """
        try:
            return self._config.get(section, key)
        except (configparser.NoSectionError, configparser.NoOptionError):
            return default

    def get_int(self, key: str, default: int = 0, section: str = "homelab") -> int:
        """Get a configuration value as an integer."""
        try:
            return self._config.getint(section, key)
        except (configparser.NoSectionError, configparser.NoOptionError, ValueError):
            return default

    def get_bool(self, key: str, default: bool = False, section: str = "homelab") -> bool:
        """Get a configuration value as a boolean."""
        try:
            return self._config.getboolean(section, key)
        except (configparser.NoSectionError, configparser.NoOptionError, ValueError):
            return default

    def exists(self, key: str, section: str = "homelab") -> bool:
        """
        Check if a configuration key exists.

        Args:
            key: Configuration key
            section: Configuration section (default: homelab)

        Returns:
            True if key exists, False otherwise
        """
        return self._config.has_option(section, key)

    def get_all(self, section: str = "homelab") -> Dict[str, str]:
        """
        Get all configuration values in a section.

        Args:
            section: Configuration section (default: homelab)

        Returns:
            Dictionary of configuration values
        """
        if self._config.has_section(section):
            return dict(self._config.items(section))
        return {}

    # ============================================================================
    # Marker File Management
    # ============================================================================

    def create_marker(self, marker: str) -> None:
        """
        Create a completion marker file.

        Args:
            marker: Marker name (e.g., "user-setup-complete")
        """
        marker_file = self.marker_dir / marker
        marker_file.touch()

    def check_marker(self, marker: str) -> bool:
        """
        Check if a completion marker exists.

        Args:
            marker: Marker name

        Returns:
            True if marker exists, False otherwise
        """
        marker_file = self.marker_dir / marker
        return marker_file.exists()

    def remove_marker(self, marker: str) -> None:
        """
        Remove a completion marker.

        Args:
            marker: Marker name
        """
        marker_file = self.marker_dir / marker
        if marker_file.exists():
            marker_file.unlink()

    # ============================================================================
    # Convenience Properties
    # ============================================================================

    @property
    def setup_user(self) -> Optional[str]:
        """Get the configured setup user."""
        return self.get("SETUP_USER") or None

    @property
    def container_runtime(self) -> Optional[str]:
        """Get the configured container runtime."""
        return self.get("CONTAINER_RUNTIME") or None

    @property
    def puid(self) -> int:
        """Get the configured PUID."""
        return self.get_int("PUID", 1000)

    @property
    def pgid(self) -> int:
        """Get the configured PGID."""
        return self.get_int("PGID", 1000)

    @property
    def timezone(self) -> str:
        """Get the configured timezone."""
        return self.get("TZ", "America/Chicago")

    @property
    def nfs_server(self) -> Optional[str]:
        """Get the configured NFS server."""
        return self.get("NFS_SERVER") or None


# Global configuration instance
_config: Optional[Config] = None


def get_config() -> Config:
    """Get the global configuration instance."""
    global _config
    if _config is None:
        _config = Config()
    return _config
