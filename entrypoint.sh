#!/bin/bash

# Function to log messages with timestamps
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if a process is running
is_running() {
    pgrep -f "$1" > /dev/null
}

# Start VPN connection
log "Starting VPN connection..."
bash /usr/local/bin/vpn-setup.sh || {
    log "Failed to establish VPN connection"
    exit 1
}

# Function to wait for a service to be ready
wait_for_service() {
    local service=$1
    local port=$2
    local max_attempts=30
    local attempt=1

    while ! nc -z localhost "$port"; do
        if [ $attempt -ge $max_attempts ]; then
            log "Error: $service failed to start after $max_attempts attempts"
            return 1
        fi
        log "Waiting for $service to start (attempt $attempt/$max_attempts)..."
        sleep 2
        ((attempt++))
    done
    return 0
}

# Initialize services
log "Starting initialization process"

# Apply TLS certificate patch
log "Applying TLS certificate patch"
/usr/local/bin/tls-patch.sh || log "Warning: TLS certificate patch may have failed"

# Set up DNS (using VPN DNS to prevent leaks)
log "Setting up DNS..."
if [ -f "/etc/resolv.conf" ]; then
    # Backup original resolv.conf
    cp /etc/resolv.conf /etc/resolv.conf.backup
fi

# Force DNS through VPN tunnel
echo "nameserver 193.138.218.74" > /etc/resolv.conf  # Mullvad DNS server
echo "nameserver 193.138.218.77" >> /etc/resolv.conf  # Mullvad backup DNS

# Prevent DNS leaks by blocking other DNS requests
iptables -A OUTPUT -p udp --dport 53 -j DROP
iptables -A OUTPUT -p tcp --dport 53 -j DROP
iptables -A OUTPUT -o tun0 -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -o tun0 -p tcp --dport 53 -j ACCEPT

# Configure and start qBittorrent
log "Setting up qBittorrent..."
/usr/local/bin/qbittorrent-setup.sh
qbittorrent-nox --daemon --webui-port=8085
wait_for_service "qBittorrent" 8085 || log "Warning: qBittorrent may not have started properly"

# Set up VPN if configured
if [ -f "/mullvad/mullvad_se_mma.conf" ]; then
    log "Setting up Mullvad VPN..."
    
    # Configure OpenVPN
    mkdir -p /etc/openvpn
    cp /mullvad/mullvad_se_mma.conf /etc/openvpn/mullvad.conf
    cp /mullvad/mullvad_ca.crt /etc/openvpn/
    cp /mullvad/mullvad_userpass.txt /etc/openvpn/
    
    # Update OpenVPN config
    sed -i 's|mullvad_userpass.txt|/etc/openvpn/mullvad_userpass.txt|g' /etc/openvpn/mullvad.conf
    sed -i 's|mullvad_ca.crt|/etc/openvpn/mullvad_ca.crt|g' /etc/openvpn/mullvad.conf
    sed -i '/script-security/d' /etc/openvpn/mullvad.conf
    sed -i '/update-resolv-conf/d' /etc/openvpn/mullvad.conf
    
    # Start OpenVPN
    openvpn --config /etc/openvpn/mullvad.conf --daemon
    sleep 10
    
    # Verify VPN connection
    if ! ip link show tun0 >/dev/null 2>&1; then
        log "Warning: VPN tunnel (tun0) not detected"
    else
        log "VPN connection established"
    fi
fi

# Start Seanime
log "Starting Seanime..."
/usr/local/bin/seanime &

# Monitor services
while true; do
    if ! is_running "qbittorrent-nox"; then
        log "qBittorrent crashed, restarting..."
        qbittorrent-nox --daemon --webui-port=8085
    fi
    
    if ! is_running "seanime"; then
        log "Seanime crashed, restarting..."
        /usr/local/bin/seanime &
    fi
    
    sleep 30
done
