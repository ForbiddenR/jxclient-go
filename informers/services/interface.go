package services

import (
	internalinterfaces "github.com/ForbiddenR/jxclient-go/informers/internalinterfaces"
	v1 "github.com/ForbiddenR/jxclient-go/informers/services/v1"
)

type Interface interface {
	V1() v1.Interface
}

type group struct {
	factory internalinterfaces.SharedInformerFactory
}

func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &group{factory: f}
}

func (g *group) V1() v1.Interface {
	return v1.New(g.factory)
}
