package cache

import "time"

type SharedInformer interface {
	// AddEventHandler adds event handler to the shared informer using
	// the shared informer's resync period. Events to a single handler are
	// delivered sequentially, but there is no coordination between
	// different handlers.
	AddEventHandler(handler ResourceEventHandler) (ResourceEventHandlerRegisteration, error)
	// AddEventHandlerWithResyncPeriod adds an event handler to the
	// shared informer with the requested resync period; zero means
	// this handler does not care about resyncs. The resync operation
	// consists of delivering to the handler an update notification
	// for every object in the informer's local cache; it does not add
	// informers do no resyncs at all, not even for handlers added
	// with a non-zero resyncPeriod. For an informer that does
	// resyncs, and for each handler that requests resyncs, that
	// informer develops a nominal resync period that is no shorter
	// between any two resyncs may be longer that the nominal period
	// because the implementation takes time to do work and there may
	// be competing load and scheduling noise.
	// It returns a registration handle for the handler that can be used to remove
	// the handler again and an error if the handler cannot be added.
	AddEventHandlerWithResyncPeriod(handler ResourceEventHandler, resyncPeriod time.Duration) (ResourceEventHandlerRegisteration, error)
	// RemoveEventHandler removes a formerly added event handler given by
	// its registration handle.
	// This function is guaranteed to be idempotent, and thread-safe.
	RemoveEventHandler(handle ResourceEventHandlerRegisteration) error
	// Run starts and runs the shared informer, returning after it stops.
	// The informer will be stopped when stopCh is closed.
	Run(stopCh <-chan struct{})

	// IsStopped reports whether the informer has already been stopped.
	// Adding event handlers to already stopped informers is no possible.
	// An informer already stopped will never be started again.
	IsStopped() bool
}


type ResourceEventHandlerRegisteration interface {
	// HasSynced reports if both the parent has synced and all pre-sync
	// events have been dellivered.
	HasSynced() bool
}

