package wssAPI

import (
	"sync"
)

//command  statue mac

//Set for cmd Status change
type Set struct {
	m map[interface{}]bool
	sync.RWMutex
}

// NewSet to New a Set Object
func NewSet() *Set {
	return &Set{
		m: make(map[interface{}]bool),
	}
}

// Add to  Set item true
func (set *Set) Add(item interface{}) {
	set.Lock()
	defer set.Unlock()
	set.m[item] = true
}

// Del to remove
func (set *Set) Del(item interface{}) {
	set.Lock()
	defer set.Unlock()
	delete(set.m, item)
}

// Has to retrive item
func (set *Set) Has(item interface{}) bool {
	set.RLock()
	defer set.RUnlock()
	return set.m[item]
}
