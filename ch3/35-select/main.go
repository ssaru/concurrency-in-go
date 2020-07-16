package main

func main() {
	var c1, c2 <-chan interface{}
	var c3 chan<- interface{}

	select {
	case <-c1:
		// 작업 수행

	case <-c2:
		// 작업 수행

	case c3 <- struct{}{}:
		// 작업 수행
	}
}
