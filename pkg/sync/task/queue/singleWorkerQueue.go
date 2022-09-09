package queue

import (
	"container/list"
	"sync"

	"go.uber.org/atomic"
)

type oneWorkerQueue struct {
	inUse        atomic.Bool
	addedToQueue atomic.Bool

	mutex           sync.Mutex
	queue           *list.List
	deleteFromQueue func()
	deleted         bool
}

func (q *oneWorkerQueue) append(tasks []func()) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.deleted {
		return false
	}
	for _, t := range tasks {
		q.queue.PushBack(t)
	}
	return true
}

func (q *oneWorkerQueue) freeLocked() {
	q.queue.Init()
	q.deleteFromQueue()
	q.deleted = true
}

func (q *oneWorkerQueue) pop() func() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queue.Len() > 0 {
		return q.queue.Remove(q.queue.Front()).(func())
	}
	q.freeLocked()
	return nil
}

func (q *oneWorkerQueue) free() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.freeLocked()
}

func (q *Queue) makeDeleteFromQueue(key interface{}) func() {
	return func() {
		q.oneworkerQueue.Delete(key)
	}
}

func (q *Queue) appendOneWorker(key interface{}, tasks []func()) (w *oneWorkerQueue) {
	for {
		val, _ := q.oneworkerQueue.LoadOrStore(key, &oneWorkerQueue{
			queue:           list.New(),
			deleteFromQueue: q.makeDeleteFromQueue(key),
		})
		v := val.(*oneWorkerQueue)
		if v.append(tasks) {
			return v
		}
	}
}

// SubmitForOneWorker function adds tasks to the Queue by key. The tasks in the Queue are executed in the order they are submitted by the Goroutine.
func (q *Queue) SubmitForOneWorker(key interface{}, tasks ...func()) error {
	w := q.appendOneWorker(key, tasks)
	if w.addedToQueue.Load() {
		return nil
	}
	err := q.Submit(func() {
		val, ok := q.oneworkerQueue.Load(key)
		if !ok {
			return
		}
		v := val.(*oneWorkerQueue)
		if !v.inUse.CompareAndSwap(false, true) {
			return
		}
		for {
			task := v.pop()
			if task == nil {
				return
			}
			task()
		}
	})
	if err == nil {
		w.addedToQueue.Store(true)
		return nil
	}
	if !w.inUse.CompareAndSwap(false, true) {
		w.addedToQueue.Store(true)
		// one goroutine processes task from one worker queue
		return nil
	}
	w.free()
	return err
}
