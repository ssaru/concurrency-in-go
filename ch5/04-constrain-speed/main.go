package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limit은 어떤 이벤트의 최대 빈도를 정의한다.
// Limit은 초당 이벤트의 수를 나타낸다. 0의 Limit는 아무런 이벤트도 허용하지 않는다.
// type Limit float64

// // NewLimiter은 새로운 r의 속도를 가지며 최대 b개의 토큰을 가지는
// // 새로운 Limiter를 리턴한다.
// func NewLimiter(r Lim it, b int) *Limiter

// // Every는 Limit에 대한 이벤트 사이의 최소 시간 간격을 변환한다.
// func Every(interval time.Duration) Limit

func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}

// func (limt *Limiter) Wait(ctx context.Context)

// // WaitN함수는 lim가 n개의 이벤트 발생을 허용할 때까지 대기한다.
// // n이 Limiter의 버퍼 사이즈를 초과하면 error를 리턴하며, Context는 취소된다.
// // 그렇지 않은 경우에는 Context의 Deadlineㅇ이 지날 때까지 대기한다.
// func (lim *Limiter) WaitN(ctx context.Context, n int)(err error)

func Open() *APIConnection {
	return &APIConnection{
		rateLimiter: rate.NewLimiter(rate.Limit(1), 1),
	}
}

type APIConnection struct {
	rateLimiter *rate.Limiter
}

func (a *APIConnection) ReadFile(ctx context.Context) error {
	if err := a.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	// 여기서는 작업하는 척한다.
	return nil
}

func (a *APIConnection) ResolveAddress(ctx context.Context) error {
	if err := a.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	// 여기서 작업하는 척한다.
	return nil
}

func main() {
	defer log.Printf("Done")

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)

	apiConnection := Open()
	var wg sync.WaitGroup
	wg.Add(20)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			err := apiConnection.ReadFile(context.Background())
			if err != nil {
				log.Printf("cannot ReadFile: %v", err)
			}
			log.Printf("ReadFile")
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			err := apiConnection.ResolveAddress(context.Background())
			if err != nil {
				log.Printf("cannot ResolveAddress: %v", err)
			}
			log.Printf("ResolveAddress")
		}()
	}

	wg.Wait()
}
