#!/bin/bash

# VPN Performance Test Script
# Tests latency and speed through the VPN connection

echo "🔍 VPN Performance Test"
echo "======================="

# Check if containers are running
if ! podman-compose ps | grep -q "Up"; then
    echo "❌ Containers are not running. Start them with: ./start-services.sh"
    exit 1
fi

echo ""
echo "📍 Current VPN Location:"
echo "------------------------"
location_info=$(podman-compose exec gluetun wget -qO- https://ipinfo.io/json 2>/dev/null || echo "{}")
if echo "$location_info" | grep -q "city"; then
    ip=$(echo "$location_info" | grep -o '"ip":"[^"]*"' | cut -d'"' -f4)
    city=$(echo "$location_info" | grep -o '"city":"[^"]*"' | cut -d'"' -f4)
    country=$(echo "$location_info" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
    org=$(echo "$location_info" | grep -o '"org":"[^"]*"' | cut -d'"' -f4)
    echo "IP: $ip"
    echo "Location: $city, $country"
    echo "Provider: $org"
else
    echo "❌ Could not get location info"
fi

echo ""
echo "⚡ Latency Tests:"
echo "----------------"

# Test latency to various servers
test_servers=(
    "google.com"
    "cloudflare.com"
    "1.1.1.1"
    "8.8.8.8"
    "github.com"
    "netflix.com"
)

for server in "${test_servers[@]}"; do
    echo -n "Testing $server... "
    ping_result=$(podman-compose exec gluetun ping -c 3 "$server" 2>/dev/null | grep "avg" | cut -d'/' -f5 2>/dev/null || echo "failed")
    if [ "$ping_result" != "failed" ]; then
        echo "${ping_result}ms"
    else
        echo "❌ Failed"
    fi
done

echo ""
echo "🌐 DNS Resolution Test:"
echo "-----------------------"
dns_start=$(date +%s%3N)
dns_result=$(podman-compose exec gluetun nslookup google.com 2>/dev/null | grep "Address" | tail -1 | awk '{print $2}' || echo "failed")
dns_end=$(date +%s%3N)
dns_time=$((dns_end - dns_start))

if [ "$dns_result" != "failed" ]; then
    echo "✅ DNS Resolution: ${dns_time}ms (resolved to $dns_result)"
else
    echo "❌ DNS Resolution failed"
fi

echo ""
echo "📊 Speed Test (Simple):"
echo "-----------------------"
echo "Testing download speed..."
speed_start=$(date +%s)
speed_result=$(podman-compose exec gluetun wget -O /dev/null --progress=dot:mega http://speedtest.wdc01.softlayer.com/downloads/test10.zip 2>&1 | grep -o '[0-9.]*[KM]B/s' | tail -1 || echo "failed")
speed_end=$(date +%s)
speed_time=$((speed_end - speed_start))

if [ "$speed_result" != "failed" ]; then
    echo "✅ Download Speed: $speed_result (test took ${speed_time}s)"
else
    echo "❌ Speed test failed"
fi

echo ""
echo "🔧 Performance Recommendations:"
echo "-------------------------------"

# Check if network optimizations are applied
rmem_max=$(sysctl -n net.core.rmem_max 2>/dev/null || echo "0")
if [ "$rmem_max" -lt 134217728 ]; then
    echo "⚠️  Run: sudo ./optimize-network.sh (for better host network performance)"
fi

# Check for high latency
if echo "$location_info" | grep -q '"country":"HU"'; then
    echo "ℹ️  Currently connected to Hungary server"
    echo "💡 If latency is high, consider using a closer Mullvad server"
fi

echo ""
echo "🔍 To monitor real-time performance:"
echo "   watch -n 2 'podman-compose exec gluetun ping -c 1 8.8.8.8'"
echo ""
echo "📋 To view VPN logs:"
echo "   podman-compose logs -f gluetun"
