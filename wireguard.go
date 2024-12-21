package main

import (
	"bytes"
	"fmt"
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
