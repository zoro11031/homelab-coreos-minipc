#!/bin/bash

# Create setup directory for templates
mkdir -p /home/core/setup
cp -r /usr/share/compose-setup /home/core/setup
chown -R core:core /home/core/setup

# Create the production compose directory
mkdir -p /srv/containers

# Copy compose files to production location
# Use a loop to safely handle potential missing files
if ls /usr/share/compose-setup/*.yml >/dev/null 2>&1; then
    cp /usr/share/compose-setup/*.yml /srv/containers/
fi

if [ -f /usr/share/compose-setup/.env.example ]; then
    cp /usr/share/compose-setup/.env.example /srv/containers/.env.example
fi

# Create appdata directory structure using brace expansion
mkdir -p /var/lib/containers/appdata/{plex,jellyfin,tautulli,overseerr,wizarr,organizr,homepage,nextcloud,immich,postgres,redis}

# Set appropriate ownership (dockeruser:dockeruser)
# Note: dockeruser may not exist yet at first boot, so we defer this to post-install
# For now, set to core:core or root:root for initial setup
chown -R core:core /srv/containers
chown -R core:core /var/lib/containers/appdata
