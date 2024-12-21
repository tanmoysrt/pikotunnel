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
	prepareServer()
	queuePendingTasks()

	globalWaitGroup.Add(1)
	go startServer()
	globalWaitGroup.Add(1)
	go runWorkers()

	globalWaitGroup.Wait()
}
