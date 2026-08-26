package kafkax

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type RetryPolicy struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	MaxAttempts int
	Jitter      float64 // fraction of the delay, e.g. 0.2 == ±20%
}

var DefaultPolicy = RetryPolicy{
	BaseDelay:   250 * time.Millisecond,
	MaxDelay:    30 * time.Second,
	Multiplier:  2,
	MaxAttempts: 5,
	Jitter:      0.2,
}

func Do(ctx context.Context, p RetryPolicy, fn func() error) error {
	var err error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == p.MaxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(p.backoff(attempt)):
		}
	}
	return err
}

func (p RetryPolicy) backoff(attempt int) time.Duration {
	d := float64(p.BaseDelay) * math.Pow(p.Multiplier, float64(attempt-1))
	if cap := float64(p.MaxDelay); d > cap {
		d = cap
	}
	if p.Jitter > 0 {
		delta := d * p.Jitter
		d = d - delta + rand.Float64()*2*delta
	}
	return time.Duration(d)
}
