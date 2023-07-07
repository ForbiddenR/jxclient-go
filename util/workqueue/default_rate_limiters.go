package workqueue

import (
	"math"
	"sync"
	"time"
)

type RateLimiter[T comparable] interface {
	// When gets an item and gets to decide how long that item should wait.
	When(item T) time.Duration
	// Forget indicates that an item is finished being retried. Doesn't matter whether it'sfor failing
	// or for success, we'll stop tracking it.
	Forget(item T)
	// NumRequeues returns back how many failures the item has had.
	NumRequeues(item T) int
}

type ItemExponentialFailureRateLimiter[T comparable] struct {
	failuresLock sync.Mutex
	failures     map[T]int

	baseDelay time.Duration
	maxDelay  time.Duration
}

func (r *ItemExponentialFailureRateLimiter[T]) When(item T) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	exp := r.failures[item]
	r.failures[item] = r.failures[item] + 1

	// The backoff is capped such that 'calculated' value never overflows.
	backoff := float64(r.baseDelay.Nanoseconds()) * math.Pow(2, float64(exp))
	if backoff > math.MaxInt64 {
		return r.maxDelay
	}

	calculated := time.Duration(backoff)
	if calculated > r.maxDelay {
		return r.maxDelay
	}

	return calculated
}