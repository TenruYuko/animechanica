#!/bin/bash
# Remove set -e to prevent the script from exiting on errors

echo "Starting Seanime container services..."

# Function to log messages with timestamps
log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log "Starting initialization process"

# Apply TLS certificate patch
log "Applying TLS certificate patch"
/usr/local/bin/tls-patch.sh || log "Warning: TLS certificate patch may have failed, continuing anyway"

# Set up DNS manually
log "Setting up DNS..."
echo "nameserver 1.1.1.1" > /etc/resolv.conf
echo "nameserver 8.8.8.8" >> /etc/resolv.conf
log "DNS configuration complete"

# Check if Mullvad configuration files exist
if [ -f "/mullvad/mullvad_se_mma.conf" ] && [ -f "/mullvad/mullvad_ca.crt" ] && [ -f "/mullvad/mullvad_userpass.txt" ]; then
  log "Setting up Mullvad VPN..."
  
  # Create necessary directories
  mkdir -p /etc/openvpn
  
  # Copy Mullvad configuration files
  cp /mullvad/mullvad_se_mma.conf /etc/openvpn/mullvad.conf || log "Warning: Failed to copy mullvad_se_mma.conf"
  cp /mullvad/mullvad_ca.crt /etc/openvpn/ || log "Warning: Failed to copy mullvad_ca.crt"
  cp /mullvad/mullvad_userpass.txt /etc/openvpn/ || log "Warning: Failed to copy mullvad_userpass.txt"
  
  # Update paths in the OpenVPN config
  sed -i 's|mullvad_userpass.txt|/etc/openvpn/mullvad_userpass.txt|g' /etc/openvpn/mullvad.conf || log "Warning: Failed to update userpass path"
  sed -i 's|mullvad_ca.crt|/etc/openvpn/mullvad_ca.crt|g' /etc/openvpn/mullvad.conf || log "Warning: Failed to update ca path"
  
  # Remove references to update-resolv-conf script
  sed -i '/script-security/d' /etc/openvpn/mullvad.conf || log "Warning: Failed to remove script-security"
  sed -i '/up \/etc\/openvpn\/update-resolv-conf/d' /etc/openvpn/mullvad.conf || log "Warning: Failed to remove up script"
  sed -i '/down \/etc\/openvpn\/update-resolv-conf/d' /etc/openvpn/mullvad.conf || log "Warning: Failed to remove down script"
  
  # Start OpenVPN in the background
  log "Starting Mullvad VPN connection..."
  openvpn --config /etc/openvpn/mullvad.conf --daemon || {
    log "Error: Failed to start OpenVPN. Continuing without VPN."
  }
  
  # Wait for VPN connection
  log "Waiting for VPN to connect..."
  sleep 10
  
  # Verify VPN connection
  if ip addr show tun0 > /dev/null 2>&1; then
    log "VPN connected successfully"
  else
    log "Warning: VPN connection may not be established. Continuing anyway..."
  fi
else
  log "Mullvad VPN configuration files not found. Skipping VPN setup."
fi

# Configure qBittorrent before starting it
log "Configuring qBittorrent..."
/usr/local/bin/qbittorrent-setup.sh || log "Warning: Failed to configure qBittorrent, continuing anyway"

# Start qBittorrent in the background
log "Starting qBittorrent..."
qbittorrent-nox --webui-port=8085 --profile=/root/.config/qBittorrent &
QBITTORRENT_PID=$!

# Wait for qBittorrent to start
log "Waiting for qBittorrent to start..."
sleep 5

# Check if qBittorrent is running
if kill -0 $QBITTORRENT_PID 2>/dev/null; then
  log "qBittorrent started successfully on port 8085"
  
  # Wait a bit more for qBittorrent to fully initialize
  sleep 2
  
  # Verify qBittorrent is responding
  if curl -s -f http://127.0.0.1:8085 > /dev/null; then
    log "qBittorrent WebUI is accessible"
  else
    log "Warning: qBittorrent WebUI may not be accessible, but continuing anyway"
  fi
else
  log "Warning: qBittorrent may have failed to start. Continuing anyway..."
fi

# Start Seanime
log "Starting Seanime..."
cd /usr/local/bin

# Start Seanime with proper error handling
./seanime -datadir /data || {
  log "Error: Seanime failed to start or crashed. Attempting to restart..."
  sleep 5
  ./seanime -datadir /data || log "Error: Seanime failed to start after retry."
}

# Keep container running if Seanime exits
log "Seanime process exited. Keeping container alive..."
tail -f /dev/null
