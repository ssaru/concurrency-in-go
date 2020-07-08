package main

import "sync"

func main() {
	var wg sync.WaitGroup
	var test bool = false

	wg.Add(1)

	go func(test bool) {
		for {
			ok := false
			if ok {
				break
			}
		}
	}(test)

	test = true

	wg.Wait()
}
