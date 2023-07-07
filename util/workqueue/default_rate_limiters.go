package workqueue

import (
	"sync"
	"time"
)

type RateLimiter[T any] interface {
	// When gets an item and gets to decide how long that item should wait.
	When(item T) time.Duration
	// Forget indicates that an item is finished being retried. Doesn't matter whether it'sfor failing
	// or for success, we'll stop tracking it.
	Forget(item T)
	// NumRequeues returns back how many failures the item has had.
	NumRequeues(item T) int
}

type ItemExponentialFailureRateLimiter[T any] struct {
	failuresLock sync.Mutex
	failures     map[T]int

	baseDelay time.Duration
	maxDelay  time.Duration
}

func (r *ItemExponentialFailureRateLimiter[T]) When(item t) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	exp := r.failures[item]
	r.failures[item] = r.failures[item] + 1

	// The backoff 
}