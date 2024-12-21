package main

func main() {
	checkForToolInEnvironment("wg")
	checkForToolInEnvironment("iptables")
	checkForToolInEnvironment("ip")
	loadConfig()

	startServer()

	// fmt.Println(getUniqueIPInSubnet())
	// fmt.Println("Hello, World!")
	// fmt.Println(generateWireguardPrivateKey())
	// fmt.Print(generateWireguardPublicKey("oFXuPHkynqyw6XPxghrfauPBe1575q7zx7ozdMPa00c="))
}
