package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/rand"
)

func checkForToolInEnvironment(tool string) {
	_, err := exec.LookPath(tool)
	if err != nil {
		log.Fatalf("%s not found in environment", tool)
	}
}

func generateRandomIP() string {
	_, ipNet, _ := net.ParseCIDR(config.WireguardSubnet) // Assume valid subnet input

	// Get the base IP as a byte slice
	ip := ipNet.IP.To4()

	// Determine the subnet mask size
	ones, bits := ipNet.Mask.Size()

	// Calculate the number of available addresses in the subnet
	numIPs := 1 << (bits - ones)
	// Create a new random source and initialize it
	source := rand.NewSource(uint64(time.Now().UnixNano()))
	rng := rand.New(source)

	// Generate a random offset within the subnet range, excluding the network and broadcast addresses
	randomOffset := rng.Intn(numIPs-2) + 1 // Exclude 0 and last address
	for i := 3; i >= 0; i-- {
		ip[i] += byte(randomOffset & 0xFF)
		randomOffset >>= 8
	}

	return ip.String()
}

func getUsedIPAddresses() []string {
	db := GetDB()
	var ips []string
	err := db.Model(&Peer{}).Select("ip").Find(&ips).Error
	if err != nil {
		log.Println("Failed to get used IP addresses:", err)
	}
	return ips
}

func getUniqueIPInSubnet() string {
	ips := getUsedIPAddresses()
	ips = append(ips, config.GetRelayWireguardAddress())
	for {
		ip := generateRandomIP()
		if !slices.Contains(ips, ip) {
			return ip
		}
	}
}

const wireguardScriptTemplate = `#!/bin/bash

# Function to check root privileges
check_root() {
    if [ "$EUID" -ne 0 ]; then 
        echo "Please run as root"
        exit 1
    fi
}

# Function to setup temporary private key
setup_private_key() {
    local private_key="$1"
    TEMP_KEY_FILE=$(mktemp)
    echo "$private_key" > "$TEMP_KEY_FILE"
    echo "$TEMP_KEY_FILE"
}

# Function to cleanup temporary files
cleanup() {
    local temp_file="$1"
    rm -f "$temp_file"
    echo "Cleaned up temporary files"
}

# Function to setup WireGuard interface
setup_wireguard() {
    local temp_key_file="$1"
    local interface_name="$2"
    
    # Create WireGuard interface
    ip link add "$interface_name" type wireguard
    
    # Configure WireGuard with private key
    wg set "$interface_name" private-key "$temp_key_file" listen-port 0
    
    # Configure peer
    wg set "$interface_name" peer "$PEER_PUBLIC_KEY" \
        allowed-ips "$ALLOWED_IPS" \
        endpoint "$ENDPOINT" \
        persistent-keepalive 25
    
    # Set IP address
    ip addr add "$INTERFACE_IP" dev "$interface_name"


    # Bring interface up
    ip link set "$interface_name" up
    
    # Add route
    ip route add "$ALLOWED_IPS" dev "$interface_name"
    
    echo "WireGuard interface $interface_name has been set up"
}

# Function to remove WireGuard interface and rules
remove_wireguard() {
    local interface_name="$1"
    
    # Check if interface exists
    if ip link show "$interface_name" >/dev/null 2>&1; then
        # Remove route
        ip route del "$ALLOWED_IPS" dev "$interface_name" 2>/dev/null || true
        
        # Bring interface down
        ip link set "$interface_name" down 2>/dev/null || true
        
        # Delete interface
        ip link del "$interface_name" 2>/dev/null || true
        
        echo "WireGuard interface $interface_name has been removed"
    else
        echo "WireGuard interface $interface_name does not exist"
    fi
}

# Main function
main() {
    # Check if command is provided
    if [ $# -lt 1 ]; then
        echo "Usage: $0 <up|down>"
        exit 1
    fi

    # Configuration variables
    local INTERFACE_NAME="wg0"
    local PRIVATE_KEY="{{.PrivateKey}}"
    local ALLOWED_IPS="{{.AllowedIPs}}"
    local PEER_PUBLIC_KEY="{{.PublicKey}}"
    local ENDPOINT="{{.WireguardRelayServerPublicIP}}:{{.WireguardListenPort}}"
    local INTERFACE_IP="{{.IP}}/32"

    # Get command
    local COMMAND="$1"

    # Check root privileges
    check_root

    case "$COMMAND" in
        "up")
            # Setup private key and get temp file path
            local temp_key_file=$(setup_private_key "$PRIVATE_KEY")
            
            # Setup WireGuard
            setup_wireguard "$temp_key_file" "$INTERFACE_NAME"
            
            # Cleanup
            cleanup "$temp_key_file"
            ;;
            
        "down")
            remove_wireguard "$INTERFACE_NAME"
            ;;
            
        *)
            echo "Error: Unknown command '$COMMAND'. Use 'up' or 'down'"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
`

func (peer *Peer) GenerateWireguardScript() string {
	wireguardScript := strings.Replace(wireguardScriptTemplate, "{{.PrivateKey}}", peer.PrivateKey, 1)
	wireguardScript = strings.Replace(wireguardScript, "{{.AllowedIPs}}", config.GetWireguardClientSubnet(), 1)
	wireguardScript = strings.Replace(wireguardScript, "{{.PublicKey}}", config.WireguardPublicKey, 1)
	wireguardScript = strings.Replace(wireguardScript, "{{.WireguardRelayServerPublicIP}}", config.WireguardRelayServerPublicIP, 1)
	wireguardScript = strings.Replace(wireguardScript, "{{.WireguardListenPort}}", strconv.Itoa(int(config.WireguardListenPort)), 1)
	wireguardScript = strings.Replace(wireguardScript, "{{.IP}}", peer.IP, 1)
	return wireguardScript
}

func (peer *Peer) GetWireguardConfig() map[string]string {
	return map[string]string{
		"private_key":      peer.PrivateKey,
		"public_key":       peer.PublicKey,
		"ip":               peer.IP,
		"ip_with_mask":     fmt.Sprintf("%s/32", peer.IP),
		"allowed_ips":      config.GetWireguardClientSubnet(),
		"relay_public_key": config.WireguardPublicKey,
		"endpoint":         fmt.Sprintf("%s:%d", config.WireguardRelayServerPublicIP, config.WireguardListenPort),
	}
}
