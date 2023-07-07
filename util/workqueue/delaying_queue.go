package workqueue

import (
	"container/heap"
	"sync"
	"time"

	"github.com/ForbiddenR/jxutils/clock"
)

// DelayingInterface is an Interface that can Add an item at a later time. This makes it easier to
// requeue items after failures without ending up in a hot-loop.
type DelayingInterface[T comparable] interface {
	Interface[T]
	// AddAfter adds an item to the workqueue after the indicated duration has passed
	AddAfter(item T, duration time.Duration)
}

type DelayingQueueConfig[T comparable] struct {
	// Name for the queue. If unnamed, the metrics will not be registered.
	Name string

	// Clock optionally allows injecting a real or fake clock for testing purposes.
	Clock clock.WithTicker

	// Queue optionally allows injecting custom queue Interface instead of the default one.
	Queue Interface[T]
}

func NewDelayingQueue[T comparable]() DelayingInterface[T] {
	return NewDelayingQueueWithConfig(DelayingQueueConfig[T]{})
}

// NewDelayingQueueWithConfig constructs a new workqueue with options to
// customize different properties.
func NewDelayingQueueWithConfig[T comparable](config DelayingQueueConfig[T]) DelayingInterface[T] {
	if config.Clock == nil {
		config.Clock = clock.RealClock{}
	}

	if config.Queue == nil {
		config.Queue = NewWithConfig[T](QueueConfig{
			Name:  config.Name,
			Clock: config.Clock,
		})
	}

	return newDelayingQueue(config.Clock, config.Queue, config.Name)
}

// NewDelayingQueueWithCustomClock constructs a new named workqueue
// with ability to inject real or fake clock for testing purposes.
// Deprecated: UseDelayingQueueWithConfig instead.
func NewDelayingQueueWithCustomClock[T comparable](clock clock.WithTicker, name string) DelayingInterface[T] {
	return NewDelayingQueueWithConfig[T](DelayingQueueConfig[T]{
		Name: name,
		Clock: clock,
	})
}

func newDelayingQueue[T comparable](clock clock.WithTicker, q Interface[T], jjname string) *delayingType[T] {
	ret := &delayingType[T]{
		Interface:       q,
		clock:           clock,
		heartbeat:       clock.NewTicker(maxWait),
		stopCh:          make(chan struct{}),
		waitingForAddCh: make(chan *waitFor[T], 1000),
	}

	go ret.waitingLoop()
	return ret
}

// delayingType wraps an Interface and provides delayed re-enquing
type delayingType[T comparable] struct {
	Interface[T]

	// clock tracks time for delayed firing
	clock clock.Clock

	// stopCh lets us signal a shutdown to the waiting loop
	stopCh chan struct{}
	// stopOnce gurantees we only signal shutdown a single time
	stopOnce sync.Once

	// heartbeat ensure we wait no more that maxWait before firing
	heartbeat clock.Ticker

	// waitingForAddCh is a buffered channel that feeds waitingForAdd
	waitingForAddCh chan *waitFor[T]
}

// waitFor holds the data to add and the time it should be added
type waitFor[T comparable] struct {
	data    T
	readyAt time.Time
	// index in the priority queue (heap)
	index int
}

// waitForPriorityQueue implements a priority queue for waitFor items.
//
// waitForPriorityQueue implements heap.Interface. The item occurring next in
// time (i.e., the item with the smallest readyAt) is at the root (index 0).
// Peek returns this minimum item at index 0. Pop returns the minimum item after
// it has been removed from the queue and placed at index Len()-1 by
// container/heap. Push adds an item at index Len(), and container/heap
// percolates it into the correct location.
type waitForPriorityQueue[T comparable] []*waitFor[T]

func (pq waitForPriorityQueue[T]) Len() int {
	return len(pq)
}

func (pq waitForPriorityQueue[T]) Less(i, j int) bool {
	return pq[i].readyAt.Before(pq[j].readyAt)
}

func (pq waitForPriorityQueue[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = j
	pq[j].index = i
}

func (pq *waitForPriorityQueue[T]) Push(x any) {
	n := len(*pq)
	item := x.(*waitFor[T])
	item.index = n
	*pq = append(*pq, item)
}

func (pq *waitForPriorityQueue[T]) Pop() any {
	n := len(*pq)
	item := (*pq)[n-1]
	item.index = -1
	*pq = (*pq)[:(n - 1)]
	return item
}

func (pq waitForPriorityQueue[T]) Peek() any {
	return pq[0]
}

func (q *delayingType[T]) ShutDown() {
	q.stopOnce.Do(func() {
		q.Interface.ShutDown()
		close(q.stopCh)
		q.heartbeat.Stop()
	})
}

// AddAfter adds the given item to the work queue after the given delay
func (q *delayingType[T]) AddAfter(item T, duration time.Duration) {
	// don't add if we're already shutting down
	if q.ShuttingDown() {
		return
	}

	// immediately add things with no delay
	if duration <= 0 {
		q.Add(item)
		return
	}

	select {
	case <-q.stopCh:
		// unblock if ShutDown() is called
	case q.waitingForAddCh <- &waitFor[T]{data: item, readyAt: q.clock.Now().Add(duration)}:
	}
}

const maxWait = 10 * time.Second

// waitingLoop runs until the workqueue is shutdown and keeps a check on the list of items to be added.
func (q *delayingType[T]) waitingLoop() {

	// Make a placeholder channel to use when there are no items in our list
	never := make(<-chan time.Time)

	// Make a timer that expires when the item at the head of the waiting queue is ready
	var nextReadyAtTimer clock.Timer

	waitingForQueue := &waitForPriorityQueue[T]{}
	heap.Init(waitingForQueue)

	waitingEntryByData := map[T]*waitFor[T]{}

	for {
		if q.Interface.ShuttingDown() {
			return
		}

		now := q.clock.Now()

		// Add ready entries
		for waitingForQueue.Len() > 0 {
			entry := waitingForQueue.Peek().(*waitFor[T])
			if entry.readyAt.After(now) {
				break
			}

			entry = heap.Pop(waitingForQueue).(*waitFor[T])
			q.Add(entry.data)
			delete(waitingEntryByData, entry.data)
		}

		// Set up a wait for the first item's readyAt (if one exists)
		nextReadyAt := never
		if waitingForQueue.Len() > 0 {
			if nextReadyAtTimer != nil {
				nextReadyAtTimer.Stop()
			}
			entry := waitingForQueue.Peek().(*waitFor[T])
			nextReadyAtTimer = q.clock.NewTimer(entry.readyAt.Sub(now))
			nextReadyAt = nextReadyAtTimer.C()
		}

		select {
		case <-q.stopCh:
			return

		case <-q.heartbeat.C():
			// continue the loop, which will add ready items

		case <-nextReadyAt:
			// continue the loop, which will add ready items

		case waitEntry := <-q.waitingForAddCh:
			if waitEntry.readyAt.After(q.clock.Now()) {
				insert(waitingForQueue, waitingEntryByData, waitEntry)
			} else {
				q.Add(waitEntry.data)
			}

			drained := false
			for !drained {
				select {
				case waitEntry := <-q.waitingForAddCh:
					if waitEntry.readyAt.After(q.clock.Now()) {
						insert(waitingForQueue, waitingEntryByData, waitEntry)
					} else {
						q.Add(waitEntry.data)
					}
				default:
					drained = true
				}
			}
		}
	}
}

// insert adds the entry to the priority queue, or updates the readyAt if it already exists in the queue
func insert[T comparable](q *waitForPriorityQueue[T], knownEntries map[T]*waitFor[T], entry *waitFor[T]) {
	// if the entry already exists, update the time only if it would cause the item to be queue sooner
	existing, exists := knownEntries[entry.data]
	if exists {
		if existing.readyAt.After(entry.readyAt) {
			existing.readyAt = entry.readyAt
			heap.Fix(q, existing.index)
		}

		return
	}

	heap.Push(q, entry)
	knownEntries[entry.data] = entry
}
