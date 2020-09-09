package main

import (
	"context"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter interface {
	Wait(context.Context) error
	Limit() rate.Limit
}

func MultiLimiter(limiters ...RateLimiter) *multiLimiter {
	byLimit := func(i, j int) bool {
		return limiters[i].Limit() < limiters[j].Limit()
	}
	sort.Slice(limiters, byLimit)
	return &multiLimiter{limiters: limiters}
}

type multiLimiter struct {
	limiters []RateLimiter
}

func (l *multiLimiter) Wait(ctx context.Context) error {
	for _, l := range l.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *multiLimiter) Limit() rate.Limit {
	return l.limiters[0].Limit()
}

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
		apiLimit: MultiLimiter(
			rate.NewLimiter(Per(2, time.Second), 2),
			rate.NewLimiter(Per(10, time.Minute), 10),
		),

		diskLimit: MultiLimiter(
			rate.NewLimiter(rate.Limit(1), 1),
		),

		networkLimit: MultiLimiter(
			rate.NewLimiter(Per(3, time.Second), 3),
		),
	}
}

type APIConnection struct {
	networkLimit,
	diskLimit,
	apiLimit RateLimiter
}

func (a *APIConnection) ReadFile(ctx context.Context) error {
	if err := MultiLimiter(a.apiLimit, a.diskLimit).Wait(ctx); err != nil {
		return err
	}
	// 여기서는 작업하는 척한다.
	return nil
}

func (a *APIConnection) ResolveAddress(ctx context.Context) error {
	if err := MultiLimiter(a.apiLimit, a.diskLimit).Wait(ctx); err != nil {
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
