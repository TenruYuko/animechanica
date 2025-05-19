#!/bin/bash
set -e

echo "Starting Seanime container services..."

# Apply TLS certificate patch
/usr/local/bin/tls-patch.sh

# Set up Mullvad VPN
echo "Setting up Mullvad VPN..."

# Create necessary directories
mkdir -p /etc/openvpn

# Copy Mullvad configuration files
cp /mullvad/mullvad_se_mma.conf /etc/openvpn/mullvad.conf
cp /mullvad/mullvad_ca.crt /etc/openvpn/
cp /mullvad/mullvad_userpass.txt /etc/openvpn/

# Set up DNS manually instead of using update-resolv-conf
echo "nameserver 1.1.1.1" > /etc/resolv.conf
echo "nameserver 8.8.8.8" >> /etc/resolv.conf

# Update paths in the OpenVPN config
sed -i 's|mullvad_userpass.txt|/etc/openvpn/mullvad_userpass.txt|g' /etc/openvpn/mullvad.conf
sed -i 's|mullvad_ca.crt|/etc/openvpn/mullvad_ca.crt|g' /etc/openvpn/mullvad.conf

# Remove references to update-resolv-conf script
sed -i '/script-security/d' /etc/openvpn/mullvad.conf
sed -i '/up \/etc\/openvpn\/update-resolv-conf/d' /etc/openvpn/mullvad.conf
sed -i '/down \/etc\/openvpn\/update-resolv-conf/d' /etc/openvpn/mullvad.conf

# Start OpenVPN in the background
echo "Starting Mullvad VPN connection..."
openvpn --config /etc/openvpn/mullvad.conf --daemon

# Wait for VPN connection
echo "Waiting for VPN to connect..."
sleep 10

# Verify VPN connection
if ip addr show tun0 > /dev/null 2>&1; then
  echo "VPN connected successfully"
else
  echo "Warning: VPN connection may not be established. Continuing anyway..."
fi

# Start qBittorrent in the background
echo "Starting qBittorrent..."
qbittorrent-nox --webui-port=8085 --profile=/root/.config/qBittorrent &

# Wait for qBittorrent to start
sleep 5
echo "qBittorrent started on port 8085"

# Start Seanime
echo "Starting Seanime..."
cd /usr/local/bin
./seanime -datadir /data
# Keep container running if Seanime exits
tail -f /dev/null
