// Package ratelimit provides rate limiting helpers that are not tied to HTTP.
package ratelimit

import (
	"context"
	"time"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// FailureCounter blocks a key after too many failures within a time window.
// Only failures are counted, so successful attempts never use up the quota.
//
// It is meant for keys the client cannot freely change, such as the email of an
// account being logged into, to stop brute force spread over many IP addresses:
//
//	if retryAfter, blocked, _ := counter.Blocked(ctx, email); blocked { ... }
//	if wrongPassword { counter.Fail(ctx, email) } else { counter.Reset(ctx, email) }
//
// Counts are kept in memory, per process. Use a shared store (e.g. Redis) when
// running several instances.
type FailureCounter struct {
	limiter *limiter.Limiter
}

// NewFailureCounter blocks a key once it has maxFailures failures within window.
// The window starts at the first failure.
func NewFailureCounter(maxFailures int64, window time.Duration) *FailureCounter {
	return &FailureCounter{
		limiter: limiter.New(memory.NewStore(), limiter.Rate{Period: window, Limit: maxFailures}),
	}
}

// Blocked reports whether key has reached the failure limit and, if so, how long
// until it is unblocked
func (f *FailureCounter) Blocked(ctx context.Context, key string) (time.Duration, bool, error) {
	c, err := f.limiter.Peek(ctx, key)
	if err != nil {
		return 0, false, err
	}
	// Remaining is 0 once the limit is used up (Reached only turns true past it)
	if c.Remaining > 0 {
		return 0, false, nil
	}
	return time.Until(time.Unix(c.Reset, 0)), true, nil
}

// Fail records a failure for key
func (f *FailureCounter) Fail(ctx context.Context, key string) error {
	_, err := f.limiter.Get(ctx, key)
	return err
}

// Reset clears the failures of key, e.g. after a successful attempt
func (f *FailureCounter) Reset(ctx context.Context, key string) error {
	_, err := f.limiter.Reset(ctx, key)
	return err
}
