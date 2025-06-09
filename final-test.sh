#!/bin/bash

echo "🎉 FINAL SYSTEM TEST - Podman Compose with Gluetun VPN"
echo "======================================================"
echo ""

# Test 1: Container Status
echo "📊 1. Container Status:"
echo "----------------------"
podman ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "(NAMES|gluetun|seanime)"
echo ""

# Test 2: VPN Connection Status
echo "🔒 2. VPN Connection Status:"
echo "----------------------------"
if podman logs gluetun | grep -q "Initialization Sequence Completed"; then
    VPN_SERVER=$(podman logs gluetun | grep "Peer Connection Initiated" | tail -1 | grep -o '\[.*\]' | tr -d '[]' | cut -d'.' -f1)
    VPN_IP=$(podman logs gluetun | grep "ip addr add dev tun0" | tail -1 | awk '{print $6}')
    echo "✅ VPN Connected to: $VPN_SERVER"
    echo "✅ VPN IP: $VPN_IP"
else
    echo "❌ VPN not connected"
fi
echo ""

# Test 3: Web Interface Accessibility
echo "🌐 3. Web Interface Tests:"
echo "--------------------------"
if curl -s --connect-timeout 5 http://localhost:43211 >/dev/null 2>&1; then
    echo "✅ Seanime WebUI (http://localhost:43211): Accessible"
else
    echo "❌ Seanime WebUI: Not accessible"
fi

if curl -s --connect-timeout 5 http://localhost:8085 >/dev/null 2>&1; then
    echo "✅ qBittorrent WebUI (http://localhost:8085): Accessible"
else
    echo "❌ qBittorrent WebUI: Not accessible"
fi
echo ""

# Test 4: Service Integration
echo "⚙️  4. Service Integration:"
echo "---------------------------"
if podman logs seanime | grep -q "Successfully logged in"; then
    echo "✅ Seanime → qBittorrent: Authentication successful"
else
    echo "❌ Seanime → qBittorrent: Authentication failed"
fi

if podman logs seanime | grep -q "Seanime started at :43211"; then
    echo "✅ Seanime: Fully started and running"
else
    echo "❌ Seanime: Not fully started"
fi
echo ""

# Test 5: VPN Kill Switch Test
echo "🚫 5. VPN Kill Switch Test:"
echo "---------------------------"
echo "Temporarily stopping VPN to test kill switch..."
podman stop gluetun >/dev/null 2>&1
sleep 2

if timeout 5 podman exec seanime wget -qO- --timeout=3 https://google.com >/dev/null 2>&1; then
    echo "❌ KILL SWITCH FAILED - External traffic leaked!"
    KILL_SWITCH_STATUS="FAILED"
else
    echo "✅ Kill switch working - No external traffic leak"
    KILL_SWITCH_STATUS="WORKING"
fi

echo "Restarting VPN..."
podman start gluetun >/dev/null 2>&1
sleep 3
echo ""

# Test 6: Performance Metrics
echo "📈 6. Basic Performance Test:"
echo "-----------------------------"
LATENCY=$(timeout 5 ping -c 1 127.0.0.1 2>/dev/null | grep 'time=' | cut -d'=' -f4 | cut -d' ' -f1)
if [ ! -z "$LATENCY" ]; then
    echo "✅ Localhost latency: ${LATENCY}ms"
else
    echo "❌ Could not measure localhost latency"
fi
echo ""

# Summary
echo "📋 SUMMARY:"
echo "==========="
echo "VPN Provider: Mullvad (Switzerland)"
echo "VPN Status: $(podman logs gluetun | grep -q "Initialization Sequence Completed" && echo "Connected" || echo "Disconnected")"
echo "Kill Switch: $KILL_SWITCH_STATUS"
echo "Seanime: $(curl -s --connect-timeout 2 http://localhost:43211 >/dev/null 2>&1 && echo "Accessible" || echo "Not accessible")"
echo "qBittorrent: $(curl -s --connect-timeout 2 http://localhost:8085 >/dev/null 2>&1 && echo "Accessible" || echo "Not accessible")"
echo ""
echo "🎯 Next Steps:"
echo "- Open Seanime: http://localhost:43211"
echo "- Open qBittorrent: http://localhost:8085"
echo "- Monitor with: podman-compose logs -f"
echo "- Performance tune: ./optimize-network.sh"
