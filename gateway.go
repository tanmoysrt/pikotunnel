package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func runWireguardCommand(input *string, args ...string) (string, error) {
	wg := exec.Command("wg", args...)
	stdoutBuf := bytes.NewBuffer(nil)
	stderrBuf := bytes.NewBuffer(nil)
	if input != nil {
		wg.Stdin = bytes.NewBufferString(*input)
	}
	wg.Stdout = stdoutBuf
	wg.Stderr = stderrBuf
	wg.Run()
	exitCode := wg.ProcessState.ExitCode()
	if exitCode != 0 {
		return "", fmt.Errorf("wireguard command failed with exit code %d: %s", exitCode, stderrBuf.String())
	}
	return strings.TrimSpace(stdoutBuf.String()), nil
}

func generateWireguardPrivateKey() (string, error) {
	result, err := runWireguardCommand(nil, "genkey")
	if err != nil {
		return "", err
	}
	return result, nil
}

func generateWireguardPublicKey(privateKey string) (string, error) {
	result, err := runWireguardCommand(&privateKey, "pubkey")
	if err != nil {
		return "", err
	}
	return result, nil
}

func initialSetup() {
	log.Println("[STARTING] Initial setup")
	// cleanup
	// flush iptables
	exec.Command("iptables", "-F", "WG_RULES").Run()
	log.Println("[DONE] Flushed iptables")

	// down and delete wg0 interface
	exec.Command("ip", "link", "set", "down", "wg0").Run()
	exec.Command("ip", "link", "delete", "wg0").Run()
	log.Println("[DONE] Deleted wg0 interface")

	// setup ip forwarding
	exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	exec.Command("sysctl", "-w", "net.ipv4.conf.all.proxy_arp=1").Run()
	log.Println("[DONE] Setup ip forwarding")

	// setup
	exec.Command("ip", "link", "add", "wg0", "type", "wireguard").Run()
	exec.Command("ip", "addr", "add", config.WireguardSubnet, "dev", "wg0").Run()
	exec.Command("ip", "link", "set", "up", "wg0").Run()
	log.Println("[DONE] Setup wg0 interface")

	// write the private key in a tmp file
	tmpFile, err := os.CreateTemp("", "wg_private_key")
	if err != nil {
		panic(err)
	}
	tmpFile.WriteString(config.WireguardPrivateKey)
	tmpFile.Close()
	log.Println("[DONE] Wrote private key to tmp file")
	// add the private key to the wg0 interface
	exec.Command("wg", "set", "wg0", "private-key", tmpFile.Name(), "listen-port", strconv.Itoa(config.WireguardListenPort)).Run()
	log.Println("[DONE] Added private key to wg0 interface")

	// setup chain
	exec.Command("iptables", "-N", "WG_RULES").Run()
	exec.Command("iptables", "-I", "FORWARD", "-i", "wg0", "-o", "wg0", "-j", "WG_RULES").Run()
	exec.Command("iptables", "-A", "WG_RULES", "-i", "wg0", "-o", "wg0", "-j", "DROP").Run()
	log.Println("[DONE] Setup iptables chain")

	log.Println("[DONE] Initial setup")
}

func prepareServer() {
	// add peers
	var createdPeers []Peer
	err := GetDB().Find(&createdPeers, "status = ?", PeerStatusCreated).Error
	if err != nil {
		panic(err)
	}
	for _, peer := range createdPeers {
		addWireguardPeer(peer.PublicKey, peer.IP)
	}
	log.Println("[DONE] Added wireguard peers")

	// add access rules
	var createdAccessRules []AccessRule
	err = GetDB().Find(&createdAccessRules, "status = ?", AccessRuleStatusCreated).Error
	if err != nil {
		panic(err)
	}
	for _, accessRule := range createdAccessRules {
		addIptablesRuleBetweenPeers(accessRule.PeerAID, accessRule.PeerBID)
	}
	log.Println("[DONE] Added access rules")
}

func addWireguardPeer(peerPublicKey string, peerWireguardIP string) {
	_, err := runWireguardCommand(nil, "set", "wg0", "peer", peerPublicKey, "allowed-ips", peerWireguardIP+"/32")
	if err != nil {
		log.Printf("[ERROR] Failed to add wireguard peer (%s): %s", peerPublicKey, err)
	}
}

func removeWireguardPeer(peerPublicKey string) {
	_, err := runWireguardCommand(nil, "set", "wg0", "peer", peerPublicKey, "remove")
	if err != nil {
		log.Printf("[ERROR] Failed to remove wireguard peer (%s): %s", peerPublicKey, err)
	}
}

func addIptablesRuleBetweenPeers(peerAIP string, peerBIP string) {
	err := exec.Command("iptables", "-I", "WG_RULES", "1", "-s", peerAIP, "-d", peerBIP, "-i", "wg0", "-o", "wg0", "-j", "ACCEPT").Run()
	if err != nil {
		log.Printf("[ERROR] Failed to add iptables rule between peers (%s -> %s): %s", peerAIP, peerBIP, err)
	}
	err = exec.Command("iptables", "-I", "WG_RULES", "1", "-s", peerBIP, "-d", peerAIP, "-i", "wg0", "-o", "wg0", "-j", "ACCEPT").Run()
	if err != nil {
		log.Printf("[ERROR] Failed to add iptables rule between peers (%s -> %s): %s", peerBIP, peerAIP, err)
	}
}

func removeIptablesRuleBetweenPeers(peerAIP string, peerBIP string) {
	err := exec.Command("iptables", "-D", "WG_RULES", "-s", peerAIP, "-d", peerBIP, "-i", "wg0", "-o", "wg0", "-j", "ACCEPT").Run()
	if err != nil {
		log.Printf("[ERROR] Failed to remove iptables rule between peers (%s -> %s): %s", peerAIP, peerBIP, err)
	}
	err = exec.Command("iptables", "-D", "WG_RULES", "-s", peerBIP, "-d", peerAIP, "-i", "wg0", "-o", "wg0", "-j", "ACCEPT").Run()
	if err != nil {
		log.Printf("[ERROR] Failed to remove iptables rule between peers (%s -> %s): %s", peerBIP, peerAIP, err)
	}
}
