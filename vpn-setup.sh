#!/bin/bash

# Exit if there's any error
set -e

# Setup iptables for kill switch
setup_killswitch() {
    # Flush existing rules
    iptables -F
    iptables -X
    iptables -t nat -F

    # Default policies - drop everything
    iptables -P INPUT DROP
    iptables -P FORWARD DROP
    iptables -P OUTPUT DROP

    # Allow loopback
    iptables -A INPUT -i lo -j ACCEPT
    iptables -A OUTPUT -o lo -j ACCEPT

    # Allow established and related connections
    iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
    iptables -A OUTPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

    # Allow local network for initial connection
    iptables -A OUTPUT -d api.mullvad.net -j ACCEPT
    iptables -A INPUT -s api.mullvad.net -j ACCEPT

    # Allow VPN connection
    iptables -A OUTPUT -o eth0 -p udp --dport 1194 -j ACCEPT
    iptables -A INPUT -i eth0 -p udp --sport 1194 -j ACCEPT

    # Once VPN is up, only allow traffic through tun0
    iptables -A INPUT -i tun0 -j ACCEPT
    iptables -A OUTPUT -o tun0 -j ACCEPT
}

# Wait for network to be ready
until ping -c 1 api.mullvad.net &>/dev/null; do
    echo "Waiting for network..."
    sleep 2
done

# Setup kill switch
echo "Setting up VPN kill switch..."
setup_killswitch

# Start OpenVPN with Mullvad config
if [ -f "/mullvad/mullvad.conf" ] && [ -f "/mullvad/mullvad_ca.crt" ]; then
    echo "Starting OpenVPN with Mullvad config..."
    cp /mullvad/auth.txt /tmp/vpn-auth.txt
    openvpn --config /mullvad/mullvad.conf --daemon --auth-user-pass /tmp/vpn-auth.txt

    # Wait for VPN connection
    echo "Waiting for VPN connection..."
    while ! ip a show tun0 up &>/dev/null; do
        sleep 2
    done
    
    echo "VPN Connected and kill switch active!"
else
    echo "Error: Mullvad config file not found!"
    exit 1
fi
