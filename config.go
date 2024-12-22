package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
)

type Config struct {
	APIToken                     string `json:"api_token"`
	WireguardSubnet              string `json:"wireguard_subnet"`
	WireguardRelayServerPublicIP string `json:"wireguard_relay_server_public_ip"`
	WireguardListenPort          int    `json:"wireguard_listen_port"`
	WireguardPrivateKey          string `json:"wireguard_private_key"`
	WireguardPublicKey           string `json:"wireguard_public_key"`
}

var config *Config

func loadConfig() {
	// load from config.json
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Println("Failed to open config.json:", err)
	}
	defer jsonFile.Close()

	json.NewDecoder(jsonFile).Decode(&config)

	if config == nil {
		config = &Config{
			WireguardListenPort: 51820,
		}
	}

	// write to config.json
	jsonFile, err = os.Create("config.json")
	if err != nil {
		log.Println("Failed to create config.json:", err)
	}
	defer jsonFile.Close()
	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "    ")
	encoder.Encode(config)

	// check if config.json is empty
	if config.APIToken == "" || config.WireguardSubnet == "" || config.WireguardRelayServerPublicIP == "" || config.WireguardListenPort == 0 || config.WireguardPrivateKey == "" || config.WireguardPublicKey == "" {
		log.Println("Config.json is empty, please fill in the required fields")
		os.Exit(1)
	}
}

func (c *Config) GetRelayWireguardAddress() string {
	return strings.Split(config.WireguardSubnet, "/")[0]
}

func (c *Config) GetWireguardClientSubnet() string {
	_, ipnet, err := net.ParseCIDR(c.WireguardSubnet)
	if err != nil {
		return ""
	}
	return ipnet.String()
}
