// Copyright 2015.  All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package common

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type RoutineWorker struct {
	routineId int32
	isStop    int32 // flag indicate to stop worker
}

func (this *RoutineWorker) Stop() {
	atomic.StoreInt32(&this.isStop, 1)
}

var (
	routineOnce sync.Once
	routinePool *RoutinePool
)

type RoutinePool struct {
	tasks         chan func()
	workers       []*RoutineWorker
	lock          sync.Mutex
	join          sync.WaitGroup
	workerCnt     int32
	isRunning     int32
	taskCnt       int32
	idleWorkerCnt int32
}

func NewRoutinePool(init_work_cnt int, pending_size int) *RoutinePool {
	pool := &RoutinePool{
		workerCnt: int32(init_work_cnt),
		isRunning: 0,
	}
	if pending_size <= 0 {
		pool.tasks = make(chan func())
	} else {
		pool.tasks = make(chan func(), pending_size)
	}
	return pool
}

func NewGlobalRoutinePool(init_work_cnt int, pending_size int) *RoutinePool {
	routineOnce.Do(func() {
		routinePool = &RoutinePool{
			workerCnt: int32(init_work_cnt),
			isRunning: 0,
		}
		if pending_size <= 0 {
			routinePool.tasks = make(chan func())
		} else {
			routinePool.tasks = make(chan func(), pending_size)
		}
	})
	return routinePool
}

func (this *RoutinePool) Start() bool {
	if atomic.LoadInt32(&this.workerCnt) <= 0 {
		return false
	}
	if atomic.LoadInt32(&this.isRunning) == 1 {
		return true
	}

	this.lock.Lock()
	defer this.lock.Unlock()
	for i := int32(0); i < this.workerCnt; i++ {
		worker := &RoutineWorker{routineId: int32(i), isStop: 1}
		this.workers = append(this.workers, worker)
		this.join.Add(1)
		go this.RoutineWrapper(worker)
	}
	atomic.StoreInt32(&this.isRunning, 1)
	return true
}

func (this *RoutinePool) Join() {
	this.join.Wait()
}

func (this *RoutinePool) Stop(wait bool) {
	ticker := time.NewTicker(100 * time.Microsecond)
	defer ticker.Stop()
	if wait {
		for wait {
			select {
			case <-ticker.C:
				if len(this.tasks) == 0 {
					wait = false
				}
			}
		}
	}
	atomic.StoreInt32(&this.isRunning, 0)
	this.Join()
}

func (this *RoutinePool) RoutineWrapper(worker *RoutineWorker) {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	defer this.join.Done()

	atomic.StoreInt32(
		&worker.isStop, 0)
	atomic.AddInt32(&this.idleWorkerCnt, 1)
	for atomic.LoadInt32(&this.isRunning) == 1 && atomic.LoadInt32(&worker.isStop) != 1 {
		select {
		case <-ticker.C:
			break
		case task := <-this.tasks:
			atomic.AddInt32(&this.idleWorkerCnt, -1)
			task()
			atomic.AddInt32(&this.idleWorkerCnt, 1)
			atomic.AddInt32(&this.taskCnt, -1)
		}
	}
	atomic.AddInt32(&this.idleWorkerCnt, -1)
}

func (this *RoutinePool) AddWorker() {
	this.lock.Lock()
	defer this.lock.Unlock()
	worker := &RoutineWorker{
		routineId: atomic.AddInt32(&this.workerCnt, 1),
		isStop:    1}
	this.workers = append(this.workers, worker)
	this.join.Add(1)
	go this.RoutineWrapper(worker)
}

func (this *RoutinePool) DelWorker() {
	this.lock.Lock()
	defer this.lock.Unlock()
	last := len(this.workers) - 1
	if last < 0 {
		return
	}
	worker := this.workers[last]
	worker.Stop()
	this.workers = this.workers[:last]
	atomic.AddInt32(&this.workerCnt, -1)
}

func (this *RoutinePool) String() string {
	o := &struct {
		taskCnt       int32 `json:"pending_task"`
		workerCnt     int32 `json:"worker_cnt"`
		idleWorkerCnt int32 `json:"idle_worker_cnt"`
	}{
		atomic.LoadInt32(&this.taskCnt),
		atomic.LoadInt32(&this.workerCnt),
		atomic.LoadInt32(&this.idleWorkerCnt),
	}
	b, _ := json.Marshal(o)
	return string(b)
}

func (this *RoutinePool) Routine(task func()) {
	for {
		select {
		case this.tasks <- task:
			atomic.AddInt32(&this.taskCnt, 1)
			return
		default:
			if cap(this.tasks) == 0 {
				if atomic.LoadInt32(&this.taskCnt) > atomic.LoadInt32(&this.idleWorkerCnt) {
					fmt.Println("no enough worker")
					this.AddWorker()
				}
			} else {
				this.tasks <- task
			}
		}
	}
}
