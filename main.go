package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"seanime/internal/server"
	"strings"
	"syscall"
	"time"
)

//go:embed web/*
var WebFS embed.FS

//go:embed internal/icon/logo.png
var embeddedLogo []byte

// VPNConfig holds the OpenVPN configuration
type VPNConfig struct {
	ConfigPath string
	LogPath    string
	AuthPath   string
}

// setupVPN creates necessary configuration files for OpenVPN
func setupVPN() (*VPNConfig, error) {
	// Create a temporary directory for VPN files
	tmpDir, err := os.MkdirTemp("", "mullvad-vpn")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Create OpenVPN configuration file
	configPath := filepath.Join(tmpDir, "mullvad.conf")
	configContent := `client
dev tun
resolv-retry infinite
nobind
persist-key
persist-tun
verb 3
remote-cert-tls server
ping 10
ping-restart 60
sndbuf 524288
rcvbuf 524288
cipher AES-256-GCM
ncp-ciphers AES-256-GCM
auth SHA512
pull-filter ignore "route-ipv6"
pull-filter ignore "ifconfig-ipv6"
script-security 2
route-noexec
route-up "/bin/sh -c 'ip route add 0.0.0.0/1 via $route_vpn_gateway && ip route add 128.0.0.0/1 via $route_vpn_gateway'"
route 43211.0.0.0 255.255.255.0

# Expose port 43211 (Seanime)
route 0.0.0.0 0.0.0.0 net_gateway
route-nopull
route-up "/bin/sh -c 'ip route add default via $route_vpn_gateway metric 1000'"

# Mullvad servers
remote se-sto-001.mullvad.net 1194 udp
remote se-sto-002.mullvad.net 1194 udp
remote se-sto-003.mullvad.net 1194 udp

# Authentication
auth-user-pass mullvad-auth.txt

# Certificates
<ca>
-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
</ca>
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return nil, fmt.Errorf("failed to write config file: %v", err)
	}

	// Create auth file (you'll need to replace with your actual Mullvad credentials)
	authPath := filepath.Join(tmpDir, "mullvad-auth.txt")
	authContent := "mullvad\n5254230649909803" // Replace with your actual username and password
	if err := os.WriteFile(authPath, []byte(authContent), 0600); err != nil {
		return nil, fmt.Errorf("failed to write auth file: %v", err)
	}

	// Create log file
	logPath := filepath.Join(tmpDir, "openvpn.log")

	return &VPNConfig{
		ConfigPath: configPath,
		LogPath:    logPath,
		AuthPath:   authPath,
	}, nil
}

// startVPN starts the OpenVPN client
func startVPN(config *VPNConfig) (*exec.Cmd, error) {
	// Check if OpenVPN is installed
	if _, err := exec.LookPath("openvpn"); err != nil {
		return nil, fmt.Errorf("OpenVPN is not installed: %v", err)
	}

	// Start OpenVPN
	cmd := exec.Command("openvpn", "--config", config.ConfigPath, "--log", config.LogPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start OpenVPN: %v", err)
	}

	// Wait for VPN connection to establish
	log.Println("Waiting for VPN connection to establish...")
	time.Sleep(5 * time.Second)

	// Check if the VPN connection is established by looking for tun interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %v", err)
	}

	hasTun := false
	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "tun") {
			hasTun = true
			log.Printf("VPN connection established on interface %s", iface.Name)
			break
		}
	}

	if !hasTun {
		// Kill the OpenVPN process if no tun interface is found
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to establish VPN connection, no tun interface found")
	}

	return cmd, nil
}

// cleanupVPN cleans up VPN resources
func cleanupVPN(vpnCmd *exec.Cmd, config *VPNConfig) {
	if vpnCmd != nil && vpnCmd.Process != nil {
		log.Println("Stopping VPN connection...")
		vpnCmd.Process.Signal(syscall.SIGTERM)
		vpnCmd.Wait()
	}

	// Remove temporary files
	if config != nil {
		os.Remove(config.ConfigPath)
		os.Remove(config.AuthPath)
		os.Remove(config.LogPath)
		os.Remove(filepath.Dir(config.ConfigPath))
	}
}

// exposePort adds an iptables rule to expose port 43211
func exposePort() error {
	// Check if iptables is installed
	if _, err := exec.LookPath("iptables"); err != nil {
		return fmt.Errorf("iptables is not installed: %v", err)
	}

	// Add iptables rule to expose port 43211
	cmd := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "43211", "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add iptables rule: %v", err)
	}

	return nil
}

func main() {
	// Setup VPN configuration
	vpnConfig, err := setupVPN()
	if err != nil {
		log.Fatalf("Failed to setup VPN: %v", err)
	}

	// Start VPN connection
	vpnCmd, err := startVPN(vpnConfig)
	if err != nil {
		log.Fatalf("Failed to start VPN: %v", err)
	}

	// Expose port 43211
	if err := exposePort(); err != nil {
		cleanupVPN(vpnCmd, vpnConfig)
		log.Fatalf("Failed to expose port: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start Seanime server in a goroutine
	go func() {
		server.StartServer(WebFS, embeddedLogo)
	}()

	// Wait for termination signal
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)

	// Cleanup VPN resources
	cleanupVPN(vpnCmd, vpnConfig)
}
