// Copyright 2015. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package common

import (
	"sync"
)

// Just a BufferdQueue simply without thread local memory
type ObjectPool struct {
	// use array instead of list for being avoid of new/delete element frequently
	// and array will prealloc enough memory
	queue []interface{}
	lock  sync.Mutex
	New   func() interface{}
}

func NewObjectPool(new func() interface{}) *ObjectPool {
	pool := &ObjectPool{New: new}
	return pool
}

func (p *ObjectPool) Get() interface{} {
	p.lock.Lock()
	defer p.lock.Unlock()

	last := len(p.queue) - 1
	if last < 0 {
		if p.New != nil {
			return p.New
		}
		return nil
	}
	rs := p.queue[last]
	p.queue = p.queue[:last]
	return rs
}

func (p *ObjectPool) Put(x interface{}) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, x)
}

func (p *ObjectPool) Free(count int) {
	p.lock.Lock()
	defer p.lock.Unlock()
	l := len(p.queue)
	if count < l {
		count = l
	}

	p.queue = p.queue[:l-count]
}
