package v1

import (
	internalinterfaces "github.com/ForbiddenR/jxclient-go/informers/internalinterfaces"
	cache "github.com/ForbiddenR/jxclient-go/tools/cache"
)

type AccessVerifyInformer interface {
	Informer() cache.SharedInformer
}

type accessVerifyInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func (a *accessVerifyInformer) Informer() cache.SharedInformer {
	return a.factory.InformerFor(struct{}{})
}