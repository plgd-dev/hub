package queue

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/panjf2000/ants/v2"
)

// Queue representation of task queue.
type Queue struct {
	goPool *ants.Pool
	limit  int

	mutex sync.Mutex
	queue *list.List
}

// New creates task queue which is processed by goroutines.
func New(cfg Config) (*Queue, error) {
	if cfg.Size <= 0 {
		return nil, fmt.Errorf("invalid value of Size")
	}
	p, err := ants.NewPool(cfg.GoPoolSize, ants.WithPreAlloc(true), ants.WithExpiryDuration(cfg.MaxIdleTime), ants.WithNonblocking(true))
	if err != nil {
		return nil, err
	}
	return &Queue{
		queue:  list.New(),
		goPool: p,
		limit:  cfg.Size,
	}, nil
}

func (q *Queue) appendQueue(tasks []func()) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queue.Len()+len(tasks) > q.limit {
		return fmt.Errorf("reached limit of max processed jobs")
	}
	for _, t := range tasks {
		q.queue.PushBack(t)
	}
	return nil
}

func (q *Queue) popQueue() func() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queue.Len() == 0 {
		return nil
	}
	return q.queue.Remove(q.queue.Front()).(func())
}

// Submit appends and execute task by Queue.
func (q *Queue) Submit(tasks ...func()) error {
	err := q.appendQueue(tasks)
	if err != nil {
		return err
	}
	q.goPool.Submit(func() {
		for {
			task := q.popQueue()
			if task == nil {
				return
			}
			task()
		}
	})
	return nil
}

// Release closes queue and release it.
func (q *Queue) Release() {
	q.goPool.Release()
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.queue.Init()
}
