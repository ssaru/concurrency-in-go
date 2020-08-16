package main

import "time"

func main() {

	done := make(chan interface{})
	defer close(done)

	take := func(
		done <-chan interface{},
		num int,
		valueStream <-chan interface{},
	) <-chan interface{} {
		takeStream := make(chan interface{})
		go func() {
			defer close(takeStream)

			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	repeat := func(
		done <-chan interface{},
		values ...interface{},
	) <-chan interface{} {
		valueStream := make(chan interface{})

		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}

	zeros := take(done, 3, repeat(done, 0))
	short := sleep(done, 1*time.Second, zeros)
	long := sleep(done, 4*time.Second, short)

	pipeline := long
}
