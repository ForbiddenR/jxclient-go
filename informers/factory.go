package informers

import (
	"reflect"
	"sync"

	esam "github.com/ForbiddenR/jxclient-go/informers/esam"
	internalinterfaces "github.com/ForbiddenR/jxclient-go/informers/internalinterfaces"
	services "github.com/ForbiddenR/jxclient-go/informers/services"
	jxclient "github.com/ForbiddenR/jxclient-go/jxclient"
	cache "github.com/ForbiddenR/jxclient-go/tools/cache"
)

type sharedInformerFactory struct {
	client    jxclient.Interface
	lock      sync.Mutex
	informers map[reflect.Type]cache.SharedInformer
	// staredInformers is used for tracking which informers have been started.
	// This allows Start() to be called multiple times safely.
	staredInformers map[reflect.Type]bool
	// wg tracks how many goroutines were started
	wg sync.WaitGroup
	// shuttingDown is true when Shutdown has been called. It may still be running
	// because it needs to wait for goroutines.
	shuttingDown bool
}

func (f *sharedInformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.shuttingDown {
		return
	}

	for informerType, informer := range f.informers {
		if !f.staredInformers[informerType] {
			f.wg.Add(1)
			// We need a new variable in each loop iteration.
			// otherwise the goroutine would use the loop variable
			// and that keeps changing.
			informer := informer
			go func() {
				defer f.wg.Done()
				informer.Run(stopCh)
			}()
			f.staredInformers[informerType] = true
		}
	}
}

func (f *sharedInformerFactory) Shutdown() {
	f.lock.Lock()
	f.shuttingDown = true
	f.lock.Unlock()

	// Will return immediately if there is nothing to wait for.
	f.wg.Wait()
}

func (f *sharedInformerFactory) InformerFor(obj interface{}) cache.SharedInformer {
	return nil
}

type SharedInformerFactory interface {
	internalinterfaces.SharedInformerFactory

	// Start initializes all requested informers. They are handled in goroutines
	// which run until the stop channel gets closed.
	Start(stopCh <-chan struct{})

	// Shutdown marks a factory as shutting down. At that point no new
	// informers can be started anymore and Start will return without
	// doing anything.
	Shutdown()

	// InformerFor returns the SharedInformer
	InformerFor(obj interface{}) cache.SharedInformer

	Esam() esam.Interface
	Services() services.Interface
}

func (f *sharedInformerFactory) Esam() esam.Interface {
	return esam.New(f)
}

func (f *sharedInformerFactory) Services() services.Interface {
	return services.New(f)
}