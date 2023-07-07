package workqueue_test

import "testing"


type Type[T comparable] struct {
	keep map[T]struct{}
}

type Data struct {

}

func TestGeneric(t *testing.T) {
	my := &Type[*Data]{
		keep: make(map[*Data]struct{}),
	}
	d1 := &Data{}
	my.keep[d1] = struct{}{} 
}