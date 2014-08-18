package rwcutils

import (
	"io"
	"sync"
	"time"
)

// RateLimiter represents a rate-limited proxy to an underlying RWC.
type RateLimiter struct {
	underlying io.ReadWriteCloser
	token_ch   chan bool
	stop_ch    chan bool
	once       sync.Once
}

// RateLimit proxies a given RWC with a rate limit in read/writes per second.
// By giving a nonzero maxburst, you can do token-bucketing.
// Units of the maxburst argument is in read/write counts.
func RateLimit(original io.ReadWriteCloser, limitrate int, maxburst int) *RateLimiter {
	toret := new(RateLimiter)
	toret.underlying = original
	tokench := make(chan bool, maxburst)
	toret.token_ch = tokench
	stop := make(chan bool)
	toret.stop_ch = stop

	go func() {
		defer original.Close()
		for {
			time.Sleep(time.Second * time.Duration(100/limitrate))
			for i := 0; i < 100; i++ {
				select {
				case <-tokench:
				case <-stop:
					return
				}
			}
		}
	}()
	return toret
}

func (rl *RateLimiter) Read(p []byte) (int, error) {
	select {
	case rl.token_ch <- true:
	case <-rl.stop_ch:
		return 0, io.ErrClosedPipe
	}
	return rl.underlying.Read(p)
}

func (rl *RateLimiter) Write(p []byte) (int, error) {
	select {
	case rl.token_ch <- true:
	case <-rl.stop_ch:
		return 0, io.ErrClosedPipe
	}
	return rl.underlying.Write(p)
}

func (rl *RateLimiter) Close() error {
	rl.once.Do(func() {
		close(rl.stop_ch)
	})
	return nil
}
