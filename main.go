package main

import (
	"log"
	"os"
	"sync"
)

var globalWaitGroup = sync.WaitGroup{}

func main() {
	// check for root
	if os.Geteuid() != 0 {
		log.Fatal("Please run as root")
	}

	checkForToolInEnvironment("wg")
	checkForToolInEnvironment("iptables")
	checkForToolInEnvironment("ip")

	loadConfig()
	initialSetup()
	queuePendingTasks()

	go startServer()
	go runWorkers()

	globalWaitGroup.Wait()
}
