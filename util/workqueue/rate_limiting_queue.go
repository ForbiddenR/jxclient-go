package workqueue

import "github.com/ForbiddenR/jxutils/clock"

type RateLimitingInterface[T comparable] interface {
	DelayingInterface[T]

	// AddRateLimited adds an item to the workqueue after the rate limiter says it's ok
	AddRateLimited(item T)

	// Forget indicates that an item is finished being retried. Doesn't matter whether it's for perm failing
	// or for success, we'll stop the rate limiter from tracking it. This only clears the `rateLimiter`, you
	// still have to call `Done` on the queue.
	Forget(item T)

	// NumRequeues returns back how many times the item was requeued
	NumRequeues(item T) int
}

type RateLimitingQueueConfig[T comparable] struct {
	// Name for the queue. If unnamed, the metrics will not be registered.
	Name string

	// Clock optionally allows injecting a read or fake clock for testing purposes.
	Clock clock.WithTicker

	// DelayingQueue optionally allows injecting custom delaying queue DelayingInterface instead of the default one.
	DelayingQueue DelayingInterface[T]
}

// NewRateLimitingQueue constructs a new workqueue with rateLimited queuing ability
// Remember to call Forget! If you don't, you may end up tracking failures forever.
// NewRateLimitingQueue does not emit metrics.
func NewRateLimitingQueue[T comparable](rateLimiter RateLimiter[T]) RateLimitingInterface[T] {
	return NewRateLimitingQueueWithConfig[T](rateLimiter, RateLimitingQueueConfig[T]{})
} 

// NewRateLimitingQueueWithConfig constructs a new workqueue with rateLimited queuing ability
// with options to customize different properties.
// Remember to call Forget! If you don't, you may end up tracking faiures forever.
func NewRateLimitingQueueWithConfig[T comparable](rateLimiter RateLimiter[T], config RateLimitingQueueConfig[T]) RateLimitingInterface[T] {
	if config.Clock == nil {
		config.Clock = clock.RealClock{}
	}

	if config.DelayingQueue == nil {
		config.DelayingQueue = NewDelayingQueueWithConfig[T](DelayingQueueConfig[T]{
			Name: config.Name,
			Clock: config.Clock,
		})
	}

	return &rateLimitingType[T] {
		DelayingInterface: config.DelayingQueue,
		rateLimiter: rateLimiter,
	}
}

// rateLimitingType wraps an Interface and provides rateLimited re-enquing
type rateLimitingType[T comparable] struct {
	DelayingInterface[T]

	rateLimiter RateLimiter[T]
}

func (q *rateLimitingType[T]) AddRateLimited(item T) {
	q.DelayingInterface.AddAfter(item, q.rateLimiter.When(item))
}

func (q *rateLimitingType[T]) NumRequeues(item T) int {
	return q.rateLimiter.NumRequeues(item)
}

func (q *rateLimitingType[T]) Forget(item T) {
	q.rateLimiter.Forget(item)
}