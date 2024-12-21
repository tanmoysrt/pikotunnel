package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	exec.Command("iptables", "-F", "WG_RULES")
	log.Println("[DONE] Flushed iptables")

	// down and delete wg0 interface
	exec.Command("ip", "link", "set", "down", "wg0")
	exec.Command("ip", "link", "delete", "wg0")
	log.Println("[DONE] Deleted wg0 interface")

	// setup ip forwarding
	exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	exec.Command("sysctl", "-w", "net.ipv4.conf.all.proxy_arp=1")
	log.Println("[DONE] Setup ip forwarding")

	// setup
	exec.Command("ip", "link", "add", "wg0", "type", "wireguard")
	exec.Command("ip", "addr", "add", config.WireguardSubnet, "dev", "wg0")
	exec.Command("ip", "link", "set", "up", "wg0")
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
	exec.Command("wg", "set", "wg0", "private-key", "/tmp/"+tmpFile.Name())
	log.Println("[DONE] Added private key to wg0 interface")

	// setup chain
	exec.Command("iptables", "-N", "WG_RULES")
	exec.Command("iptables", "-I", "FORWARD", "-i", "wg0", "-o", "wg0", "-j", "WG_RULES")
	exec.Command("iptables", "-A", "WG_RULES", "-i", "wg0", "-o", "wg0", "-j", "DROP")
	log.Println("[DONE] Setup iptables chain")

	log.Println("[DONE] Initial setup")
}
