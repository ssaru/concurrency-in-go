package main

import "fmt"

func main() {
	sayHello := func() {
		fmt.Println("Hello")
	}

	go sayHello()
}
