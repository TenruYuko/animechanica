#!/bin/bash

# Network optimization script for better VPN performance
# Run this on the host system before starting containers

echo "🚀 Optimizing network settings for VPN performance..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script needs to be run as root (use sudo)"
    echo "Usage: sudo ./optimize-network.sh"
    exit 1
fi

# Backup current settings
echo "📋 Creating backup of current network settings..."
sysctl -a | grep -E "(net\.core|net\.ipv4)" > /tmp/network_settings_backup_$(date +%Y%m%d_%H%M%S).txt
echo "   Backup saved to /tmp/network_settings_backup_*.txt"

echo ""
echo "⚡ Applying network optimizations..."

# Increase network buffer sizes
echo "   Increasing network buffer sizes..."
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728
sysctl -w net.core.rmem_default=65536
sysctl -w net.core.wmem_default=65536

# Optimize network device queue
echo "   Optimizing network device queue..."
sysctl -w net.core.netdev_max_backlog=5000
sysctl -w net.core.netdev_budget=600

# TCP optimizations
echo "   Applying TCP optimizations..."
sysctl -w net.ipv4.tcp_window_scaling=1
sysctl -w net.ipv4.tcp_timestamps=1
sysctl -w net.ipv4.tcp_sack=1
sysctl -w net.ipv4.tcp_fastopen=3
sysctl -w net.ipv4.tcp_low_latency=1

# Set BBR congestion control if available
if sysctl net.ipv4.tcp_available_congestion_control | grep -q bbr; then
    echo "   Enabling BBR congestion control..."
    sysctl -w net.ipv4.tcp_congestion_control=bbr
else
    echo "   BBR not available, using cubic..."
    sysctl -w net.ipv4.tcp_congestion_control=cubic
fi

# Reduce TCP retransmission timeouts
echo "   Optimizing TCP retransmission settings..."
sysctl -w net.ipv4.tcp_syn_retries=3
sysctl -w net.ipv4.tcp_synack_retries=3
sysctl -w net.ipv4.tcp_retries2=8

# Optimize for VPN traffic
echo "   Optimizing for VPN traffic..."
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv4.conf.all.rp_filter=0
sysctl -w net.ipv4.conf.default.rp_filter=0

# Disable IPv6 if not needed (reduces overhead)
echo "   Disabling IPv6 (reduces overhead)..."
sysctl -w net.ipv6.conf.all.disable_ipv6=1
sysctl -w net.ipv6.conf.default.disable_ipv6=1

echo ""
echo "✅ Network optimizations applied!"
echo ""
echo "📝 To make these settings permanent, add them to /etc/sysctl.conf:"
echo "   sudo tee -a /etc/sysctl.conf << EOF"
echo "# VPN performance optimizations"
echo "net.core.rmem_max=134217728"
echo "net.core.wmem_max=134217728"
echo "net.core.netdev_max_backlog=5000"
echo "net.ipv4.tcp_window_scaling=1"
echo "net.ipv4.tcp_timestamps=1"
echo "net.ipv4.tcp_sack=1"
echo "net.ipv4.tcp_fastopen=3"
echo "net.ipv4.tcp_congestion_control=bbr"
echo "net.ipv6.conf.all.disable_ipv6=1"
echo "EOF"
echo ""
echo "🔄 To apply these settings automatically at boot:"
echo "   sudo sysctl -p"
echo ""
echo "⚠️  Note: These settings are temporary until reboot unless added to /etc/sysctl.conf"
