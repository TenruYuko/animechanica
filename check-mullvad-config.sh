#!/bin/bash

# Script to check Mullvad configuration files for Gluetun compatibility

MULLVAD_DIR="/aeternae/configurations/animechanica/mullvad"

echo "Checking Mullvad configuration files..."

# Check if directory exists
if [ ! -d "$MULLVAD_DIR" ]; then
    echo "❌ Error: Mullvad directory not found at $MULLVAD_DIR"
    echo "Please create the directory and add your Mullvad configuration files."
    exit 1
fi

# Check for required files
REQUIRED_FILES=(
    "mullvad_ca.crt"
    "mullvad_userpass.txt"
)

# Find any .conf file
CONF_FILES=($(find "$MULLVAD_DIR" -name "*.conf" -type f))

ALL_PRESENT=true

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$MULLVAD_DIR/$file" ]; then
        echo "✅ Found: $file"
    else
        echo "❌ Missing: $file"
        ALL_PRESENT=false
    fi
done

# Check for OpenVPN config files
if [ ${#CONF_FILES[@]} -gt 0 ]; then
    echo "✅ Found OpenVPN config file(s):"
    for conf_file in "${CONF_FILES[@]}"; do
        echo "   - $(basename "$conf_file")"
    done
    
    # Update compose file to use the first found config
    FIRST_CONF=$(basename "${CONF_FILES[0]}")
    echo "📝 Using config file: $FIRST_CONF"
    
    # Update the compose file if needed
    if [ -f "podman-compose.yml" ]; then
        sed -i "s|OPENVPN_CUSTOM_CONFIG=/gluetun/mullvad/.*\.conf|OPENVPN_CUSTOM_CONFIG=/gluetun/mullvad/$FIRST_CONF|g" podman-compose.yml
        echo "✅ Updated podman-compose.yml to use $FIRST_CONF"
    fi
else
    echo "❌ Missing: OpenVPN configuration file (*.conf)"
    ALL_PRESENT=false
fi

if [ "$ALL_PRESENT" = false ]; then
    echo ""
    echo "Missing required files. Please ensure you have:"
    echo "1. *.conf - OpenVPN configuration file from Mullvad (any server location)"
    echo "2. mullvad_ca.crt - Certificate authority file"
    echo "3. mullvad_userpass.txt - Username and password (one per line)"
    echo ""
    echo "You can download these from your Mullvad account page."
    exit 1
fi

# Check userpass format
if [ -f "$MULLVAD_DIR/mullvad_userpass.txt" ]; then
    LINES=$(wc -l < "$MULLVAD_DIR/mullvad_userpass.txt")
    if [ "$LINES" -eq 2 ]; then
        echo "✅ mullvad_userpass.txt format looks correct (2 lines)"
    else
        echo "⚠️  Warning: mullvad_userpass.txt should contain exactly 2 lines:"
        echo "   Line 1: Username"
        echo "   Line 2: Password"
    fi
fi

# Check OpenVPN config content
if [ ${#CONF_FILES[@]} -gt 0 ]; then
    if grep -q "remote " "${CONF_FILES[0]}"; then
        echo "✅ OpenVPN config contains remote server information"
    else
        echo "⚠️  Warning: OpenVPN config may be missing remote server information"
    fi
fi

echo ""
echo "✅ Configuration check complete!"
echo "You can now run: podman-compose up -d"
