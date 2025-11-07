#!/bin/bash
mkdir -p /home/core/setup
cp -r /usr/share/wireguard-setup /home/core/setup
chown -R core:core /home/core/setup