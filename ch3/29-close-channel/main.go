package main

import "fmt"

func main() {
	intStream := make(chan int)
	close(intStream)
	integer, ok := <-intStream
	fmt.Printf("(%v): %v", ok, integer)
}
