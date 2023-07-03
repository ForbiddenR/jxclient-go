package internalinterfaces

import (
	"time"

	jxclient "github.com/ForbiddenR/jxclient-go/jxclient"
	cache "github.com/ForbiddenR/jxclient-go/tools/cache"
)

// NewInformerFunc takes jxclient.Interface and time.Duration to return a SharedInformer.
type NewInformerFunc func(jxclient.Interface, time.Duration) cache.SharedInformer

// SharedInformerFactory a small interface to allow for adding an informer without an import cycle.
type SharedInformerFactory interface {
	Start(stopCh <-chan struct{})
	InformerFor(obj interface{}) cache.SharedInformer
}