package v1

import (
	internalinterfaces "github.com/ForbiddenR/jxclient-go/informers/internalinterfaces"
	cache "github.com/ForbiddenR/jxclient-go/tools/cache"
)

type SendQRCodeInformer interface {
	Informer() cache.SharedInformer
}

type sendQRCodeInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func (s *sendQRCodeInformer) Informer() cache.SharedInformer {
	return s.factory.InformerFor(struct{}{})
}
