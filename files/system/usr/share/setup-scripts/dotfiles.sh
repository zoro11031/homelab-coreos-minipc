#!/bin/bash
# Copy dotfiles directory to core user's home
cp -a /usr/share/bluebuild/dotfiles/. /home/core/
ECHO "Dotfiles copied to /home/core/"
chown -R core:core /home/core/
ECHO "Dotfiles ownership changed to core user."