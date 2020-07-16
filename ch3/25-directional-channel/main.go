package main

func main() {
	var receiveChan <-chan interface{}
	var sendChan chan<- interface{}
	dataStream := make(chan interface{})

	// 유효한 구문
	receiveChan = dataStream
	sendChan = dataStream
}
