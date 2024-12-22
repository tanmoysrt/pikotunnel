package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

var globalWaitGroup = sync.WaitGroup{}

func main() {
	// check for root
	if os.Geteuid() != 0 {
		log.Fatal("Please run as root")
	}

	checkForToolInEnvironment("tar")
	checkForToolInEnvironment("wg")
	checkForToolInEnvironment("iptables")
	checkForToolInEnvironment("ip")
	checkForToolInEnvironment("sqlite3")

	loadConfig()

	if len(os.Args) < 2 {
		fmt.Println("Please provide a command : <backup|flush|server>")
		os.Exit(1)
	}
	cmd := os.Args[1]
	if cmd == "backup" {
		backup()
	} else if cmd == "flush" {
		// initial setup will do the job
		initialSetup()
	} else if cmd == "server" {
		initialSetup()
		prepareServer()
		queuePendingTasks()
		globalWaitGroup.Add(1)
		go startServer()
		globalWaitGroup.Add(1)
		go runWorkers()
		globalWaitGroup.Wait()
	}
}

func backup() {
	cmd := exec.Command("sqlite3", dbPath, ".dump")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	os.WriteFile("backup.sql", out, 0644)
	defer os.Remove("backup.sql")

	// tar backup.sql and config.json
	backupFileName := fmt.Sprintf("backup_%s.tar.gz", time.Now().Format("2006-01-02_15-04-05"))
	cmd = exec.Command("tar", "-czf", backupFileName, "backup.sql", "config.json")
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Printf("Backup saved to %s\n", backupFileName)
	}
}
