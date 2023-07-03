package v1

import internalinterfaces "github.com/ForbiddenR/jxclient-go/informers/internalinterfaces"

type Interface interface {
	SendQRCode() SendQRCodeInformer
}

type version struct {
	factory internalinterfaces.SharedInformerFactory
}

func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &version{factory: f}
}

func (v *version) SendQRCode() SendQRCodeInformer {
	return &sendQRCodeInformer{factory: v.factory}
}