package workqueue

import (
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
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

func DefaultContrllerRateLimiter[T comparable]() RateLimiter[T] {
	return NewMaxOfRateLimiter[T](
		NewItemExponentialFailureRateLimiter[T](5*time.Millisecond, 1000*time.Second),
		// 10 qps, 100bucket size. This is only for retry speed and its only the overall factor(not per item)
		&BucketRateLimiter[T]{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}

// BucketRateLimiter adapts a standard bucket to the workqueue ratelimiter API
type BucketRateLimiter[T comparable] struct {
	*rate.Limiter
}

var _ RateLimiter[any] = &BucketRateLimiter[any]{}

func (r *BucketRateLimiter[T]) When(item T) time.Duration {
	return r.Limiter.Reserve().Delay()
}

func (r *BucketRateLimiter[T]) NumRequeues(item T) int {
	return 0
}

func (r *BucketRateLimiter[T]) Forget(item T) {
}

type ItemExponentialFailureRateLimiter[T comparable] struct {
	failuresLock sync.Mutex
	failures     map[T]int

	baseDelay time.Duration
	maxDelay  time.Duration
}

var _ RateLimiter[any] = &ItemExponentialFailureRateLimiter[any]{}

func NewItemExponentialFailureRateLimiter[T comparable](baseDelay time.Duration, maxDelay time.Duration) RateLimiter[T] {
	return &ItemExponentialFailureRateLimiter[T]{
		failures:  map[T]int{},
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
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

func (r *ItemExponentialFailureRateLimiter[T]) NumRequeues(item T) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item]
}

func (r *ItemExponentialFailureRateLimiter[T]) Forget(item T) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	delete(r.failures, item)
}

// ItemFastSlowRateLimiter does a quick retry for a certain number of attempts, then a slow retry after that
type ItemFastSlowRateLimiter[T comparable] struct {
	failuresLock sync.Mutex
	failures     map[T]int

	maxFastAttempts int
	fastDelay       time.Duration
	slowDelay       time.Duration
}

var _ RateLimiter[any] = &ItemFastSlowRateLimiter[any]{}

func NewItemFastSlowRateLimiter[T comparable](fastDelay, slowDelay time.Duration, maxFastAttempts int) RateLimiter[T] {
	return &ItemFastSlowRateLimiter[T]{
		failures:        map[T]int{},
		fastDelay:       fastDelay,
		slowDelay:       slowDelay,
		maxFastAttempts: maxFastAttempts,
	}
}

func (r *ItemFastSlowRateLimiter[T]) When(item T) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	r.failures[item] = r.failures[item] + 1

	if r.failures[item] <= r.maxFastAttempts {
		return r.fastDelay
	}

	return r.slowDelay
}

func (r *ItemFastSlowRateLimiter[T]) NumRequeues(item T) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item]
}

func (r *ItemFastSlowRateLimiter[T]) Forget(item T) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	delete(r.failures, item)
}

// NewMaxOfRateLimiter calls every RateLimiter and returns the worst case respose
// When used with a token bucket limiter, the burst could be apparently exceeded in cases where particular items
// were separately delayed a longer time.
type MaxOfRateLimiter[T comparable] struct {
	limiters []RateLimiter[T]
}

func (r *MaxOfRateLimiter[T]) When(item T) time.Duration {
	ret := time.Duration(0)
	for _, limiter := range r.limiters {
		curr := limiter.When(item)
		if curr > ret {
			ret = curr
		}
	}

	return ret
}

func NewMaxOfRateLimiter[T comparable](limiters ...RateLimiter[T]) RateLimiter[T] {
	return &MaxOfRateLimiter[T]{limiters: limiters}
}

func (r *MaxOfRateLimiter[T]) NumRequeues(item T) int {
	ret := 0
	for _, limiter := range r.limiters {
		curr := limiter.NumRequeues(item)
		if curr > ret {
			ret = curr
		}
	}

	return ret
}

func (r *MaxOfRateLimiter[T]) Forget(item T) {
	for _, limiter := range r.limiters {
		limiter.Forget(item)
	}
}

// WithMaxWaitRateLimiter have maxDelay which avoids waiting too long
type WithMaxWaitRateLimiter[T comparable] struct {
	limiter  RateLimiter[T]
	maxDelay time.Duration
}

func NewWithMaxWaitRateLimiter[T comparable](limiter RateLimiter[T], maxDelay time.Duration) RateLimiter[T] {
	return &WithMaxWaitRateLimiter[T]{limiter: limiter, maxDelay: maxDelay}
}

func (w WithMaxWaitRateLimiter[T]) When(item T) time.Duration {
	delay := w.limiter.When(item)
	if delay > w.maxDelay {
		return w.maxDelay
	}
	return delay
}

func (w WithMaxWaitRateLimiter[T]) Forget(item T) {
	w.limiter.Forget(item)
}

func (w WithMaxWaitRateLimiter[T]) NumRequeues(item T) int {
	return w.limiter.NumRequeues(item)
}
