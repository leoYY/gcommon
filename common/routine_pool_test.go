// Copyright 2015. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package common

import (
	"fmt"
	"testing"
)

func TestRountineFunc(t *testing.T) {
	pool := NewRoutinePool(10, 0)
	fmt.Println("out func")
	if !pool.Start() {
		t.Fail()
	}
	i := 1
	fn := func() {
		fmt.Println("in func")
		i++
	}
	pool.Routine(fn)
	pool.Stop(true)
	if i <= 1 {
		t.Fail()
	}
}

func TestRountineFuncMulti(t *testing.T) {
	pool := NewRoutinePool(10, 0)
	if !pool.Start() {
		t.Fail()
	}
	i := 1
	fn := func() {
		i++
	}
	for j := 1; j < 10; j++ {
		pool.Routine(fn)
	}
	pool.Stop(true)
	if i < 10 {
		fmt.Println("not correct ", i)
		t.Fail()
	}
}
