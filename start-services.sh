#!/bin/bash

# Startup script for Seanime with Gluetun VPN

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting Seanime with Gluetun VPN..."

# Check for network optimization
echo "🔧 Network Performance Check..."
current_rmem=$(sysctl -n net.core.rmem_max 2>/dev/null || echo "0")
if [ "$current_rmem" -lt 134217728 ]; then
    echo "⚠️  Network buffers not optimized for VPN performance"
    echo "💡 Run: sudo ./optimize-network.sh (for better performance)"
else
    echo "✅ Network settings appear optimized"
fi

# Check Mullvad configuration
echo "📋 Checking Mullvad configuration..."
./check-mullvad-config.sh || {
    echo "❌ Configuration check failed. Please fix the issues above."
    exit 1
}

echo ""
echo "🔨 Building and starting services..."

# Build and start services
podman-compose down 2>/dev/null || true
bash build-web.sh
bash build-pylons.sh
podman-compose build --no-cache seanime
podman-compose up -d

echo ""
echo "⏳ Waiting for services to start..."

# Wait for Gluetun to establish VPN connection
echo "🔒 Waiting for VPN connection..."
timeout=150
counter=0
vpn_connected=false

while [ $counter -lt $timeout ]; do
    # Check multiple indicators of successful VPN connection
    logs=$(podman-compose logs gluetun 2>/dev/null || echo "")
    
    if echo "$logs" | grep -q "Initialization Sequence Completed"; then
        echo "✅ OpenVPN connection established!"
        vpn_connected=true
    elif echo "$logs" | grep -q "healthy!"; then
        if [ "$vpn_connected" = "true" ]; then
            # Get the public IP to confirm VPN is working
            sleep 2
            public_ip=$(podman-compose exec gluetun wget -qO- https://ipinfo.io/ip 2>/dev/null || echo "unknown")
            if [ "$public_ip" != "unknown" ] && [ -n "$public_ip" ]; then
                echo "🌍 VPN connected successfully! Public IP: $public_ip"
                
                # Get location info
                location_info=$(podman-compose exec gluetun wget -qO- https://ipinfo.io/json 2>/dev/null || echo "{}")
                if echo "$location_info" | grep -q "city"; then
                    city=$(echo "$location_info" | grep -o '"city":"[^"]*"' | cut -d'"' -f4)
                    country=$(echo "$location_info" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
                    org=$(echo "$location_info" | grep -o '"org":"[^"]*"' | cut -d'"' -f4)
                    echo "📍 Location: $city, $country ($org)"
                fi
                break
            fi
        fi
    fi
    
    # Show different status messages based on what we see in logs
    if [ $((counter % 15)) -eq 0 ]; then
        if echo "$logs" | grep -q "starting"; then
            echo "   🔄 VPN service starting... ($counter/$timeout seconds)"
        elif echo "$logs" | grep -q "OpenVPN.*starting"; then
            echo "   🔗 OpenVPN connecting... ($counter/$timeout seconds)"
        elif echo "$logs" | grep -q "Peer Connection Initiated"; then
            echo "   🤝 Establishing connection to VPN server... ($counter/$timeout seconds)"
        elif echo "$logs" | grep -q "AUTH_FAILED"; then
            echo "   ❌ Authentication failed. Check your Mullvad credentials."
            break
        else
            echo "   ⏳ Waiting for VPN connection... ($counter/$timeout seconds)"
        fi
    fi
    
    sleep 1
    counter=$((counter + 1))
done

if [ $counter -ge $timeout ] || echo "$logs" | grep -q "AUTH_FAILED"; then
    echo "❌ VPN connection failed or timed out. Recent Gluetun logs:"
    echo "----------------------------------------"
    podman-compose logs --tail 20 gluetun
    echo "----------------------------------------"
    exit 1
fi

# Wait for services to be ready
echo "🎯 Waiting for services to be ready..."
sleep 10

# Check if services are running
echo ""
echo "📊 Service status:"

if curl -s -f http://localhost:43211 > /dev/null 2>&1; then
    echo "✅ Seanime is running on http://localhost:43211"
else
    echo "⚠️  Seanime may still be starting..."
fi

if curl -s -f http://localhost:8085 > /dev/null 2>&1; then
    echo "✅ qBittorrent is running on http://localhost:8085"
else
    echo "⚠️  qBittorrent may still be starting..."
fi

echo ""
echo "🔐 VPN Status Verification:"
echo "----------------------------------------"
# Test VPN connection from Seanime container
seanime_ip=$(podman-compose exec seanime curl -s --max-time 5 https://ipinfo.io/ip 2>/dev/null || echo "failed")
if [ "$seanime_ip" != "failed" ] && [ -n "$seanime_ip" ]; then
    seanime_location=$(podman-compose exec seanime curl -s --max-time 5 https://ipinfo.io/json 2>/dev/null || echo "{}")
    if echo "$seanime_location" | grep -q "city"; then
        city=$(echo "$seanime_location" | grep -o '"city":"[^"]*"' | cut -d'"' -f4)
        country=$(echo "$seanime_location" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
        echo "✅ Seanime container VPN IP: $seanime_ip ($city, $country)"
    else
        echo "✅ Seanime container VPN IP: $seanime_ip"
    fi
else
    echo "❌ Failed to get Seanime container IP - VPN may not be working properly"
fi

# Test kill switch by checking if local IP is different
local_ip=$(curl -s --max-time 3 https://ipinfo.io/ip 2>/dev/null || echo "unknown")
if [ "$local_ip" != "unknown" ] && [ "$local_ip" != "$seanime_ip" ]; then
    echo "✅ Kill switch working: Local IP ($local_ip) differs from container IP"
elif [ "$local_ip" = "$seanime_ip" ]; then
    echo "⚠️  Warning: Local IP matches container IP - kill switch may not be working"
else
    echo "ℹ️  Could not verify kill switch (local IP check failed)"
fi
echo "----------------------------------------"

echo ""
echo "🔍 Useful commands:"
echo "Check VPN status and IP:"
echo "  podman-compose exec gluetun wget -qO- https://ipinfo.io"
echo "Check Seanime container IP:"
echo "  podman-compose exec seanime curl -s https://ipinfo.io"
echo "Test VPN performance and latency:"
echo "  ./test-vpn-performance.sh"
echo "Optimize host network settings:"
echo "  sudo ./optimize-network.sh"
echo "Monitor VPN connection:"
echo "  podman-compose logs -f gluetun | grep -E '(INFO|ERROR)'"
echo "Real-time latency monitoring:"
echo "  watch -n 2 'podman-compose exec gluetun ping -c 1 8.8.8.8'"
echo "Test kill switch (stop VPN):"
echo "  podman-compose exec gluetun pkill -f openvpn"
echo ""
echo "📋 View all logs:"
echo "  podman-compose logs -f"
echo ""
echo "🛑 Stop all services:"
echo "  podman-compose down"
echo ""
echo "🎉 Setup complete!"
