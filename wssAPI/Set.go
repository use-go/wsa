package wssAPI

import (
	"sync"
)

type Set struct {
	m map[interface{}]bool
	sync.RWMutex
}

func NewSet() *Set {
	return &Set{
		m: make(map[interface{}]bool),
	}
}

func (set *Set) Add(item interface{}) {
	set.Lock()
	defer set.Unlock()
	set.m[item] = true
}

func (set *Set) Del(item interface{}) {
	set.Lock()
	defer set.Unlock()
	delete(set.m, item)
}

func (set *Set) Has(item interface{}) bool {
	set.RLock()
	defer set.RUnlock()
	return set.m[item]
}
