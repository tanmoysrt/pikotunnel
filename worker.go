package main

func runWorkers() {
	globalWaitGroup.Add(1)
	process()
	globalWaitGroup.Done()
}

func process() {

}
