package v1

import internalinterfaces "github.com/ForbiddenR/jxclient-go/informers/internalinterfaces"

type Interface interface {
	AccessVerify() AccessVerifyInformer
}

type version struct {
	factory internalinterfaces.SharedInformerFactory
}

func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &version{factory: f}
}

func (v *version) AccessVerify() AccessVerifyInformer {
	return &accessVerifyInformer{factory: v.factory}
}