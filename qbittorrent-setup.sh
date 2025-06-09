#!/bin/bash

set -e

CONFIG_DIR="/root/.config/qBittorrent"
mkdir -p "$CONFIG_DIR"

# Create default config if it doesn't exist
if [ ! -f "$CONFIG_DIR/qBittorrent.conf" ]; then
    cat > "$CONFIG_DIR/qBittorrent.conf" << EOL
[LegalNotice]
Accepted=true

[Preferences]
WebUI\Port=8085
WebUI\Username=admin
WebUI\Password_PBKDF2="@ByteArray(DSnAaJR4f0qJ6yo90ZwVkQ==:R6TSNtYH+wYNOc0s7S6YGFGYYm2YVAAi+6yX7P5PzptM+iYukqPbPigqz3QEyLrspz6eOmA98Ld0YEZwJhoSBA==)"  # Default password: adminadmin
WebUI\Address=0.0.0.0
WebUI\CSRFProtection=false
Downloads\SavePath=/media/dl_anime
Connection\PortRangeMin=6881
# Performance optimizations for VPN
Connection\GlobalMaxRatio=2
Connection\GlobalMaxConnections=200
Connection\GlobalMaxUploads=20
BitTorrent\Session\DefaultSavePath=/media/dl_anime
BitTorrent\Session\TempPath=/media/dl_anime/temp
BitTorrent\Session\MaxConnectionsPerTorrent=100
BitTorrent\Session\MaxUploadsPerTorrent=10
BitTorrent\Session\SlowTorrentsDownloadRate=50
BitTorrent\Session\SlowTorrentsUploadRate=50
BitTorrent\Session\SlowTorrentsInactivityTimer=60
BitTorrent\Session\OutgoingPortsMin=6881
BitTorrent\Session\OutgoingPortsMax=6999
BitTorrent\Session\UTPRateLimited=true
BitTorrent\Session\IncludeOverheadInLimits=false
BitTorrent\Session\IgnoreLimitsOnLAN=true
EOL
fi

# Ensure proper permissions
chown -R root:root "$CONFIG_DIR"
