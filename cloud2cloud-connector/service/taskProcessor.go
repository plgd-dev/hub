package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
	kitSync "github.com/plgd-dev/kit/sync"
	"golang.org/x/sync/semaphore"
)

type TaskType string

const (
	TaskType_PullDevice          TaskType = "pulldevice"
	TaskType_SubscribeToDevices  TaskType = "subdevices"
	TaskType_SubscribeToDevice   TaskType = "subdevice"
	TaskType_SubscribeToResource TaskType = "subresource"
)

type Task struct {
	taskType      TaskType
	linkedCloud   store.LinkedCloud
	linkedAccount store.LinkedAccount
	deviceID      string
	href          string
}

type TaskProcessor struct {
	tasksChan     chan Task
	wg            *sync.WaitGroup
	poolGets      *semaphore.Weighted
	timeout       time.Duration
	raClient      pbRA.ResourceAggregateClient
	pulledDevices *kitSync.Map //[userid+deviceID]
	delay         time.Duration
}

func NewTaskProcessor(raClient pbRA.ResourceAggregateClient, maxParallelGets int64, cacheSize int, timeout, delay time.Duration) *TaskProcessor {
	return &TaskProcessor{
		pulledDevices: kitSync.NewMap(),
		tasksChan:     make(chan Task, cacheSize),
		wg:            &sync.WaitGroup{},
		poolGets:      semaphore.NewWeighted(maxParallelGets),
		timeout:       timeout,
		raClient:      raClient,
		delay:         delay,
	}
}

func (h *TaskProcessor) Trigger(task Task) {
	h.tasksChan <- task
}

func (h *TaskProcessor) pullDevice(ctx context.Context, e Task, subscriptionManager *SubscriptionManager) error {
	key := getKey(e.linkedAccount.UserID, e.deviceID)
	_, loaded := h.pulledDevices.LoadOrStore(key, e)
	if loaded {
		return nil
	}
	var device RetrieveDeviceWithLinksResponse
	err := Get(ctx, e.linkedCloud.Endpoint.URL+"/devices/"+e.deviceID, e.linkedAccount, e.linkedCloud, &device)
	if err != nil {
		h.pulledDevices.Delete(key)
		return fmt.Errorf("cannot pull device %v for linked linkedAccount(%v): %w", e.deviceID, e.linkedAccount, err)
	}
	err = publishDeviceResources(ctx, h.raClient, e.deviceID, e.linkedAccount, e.linkedCloud, subscriptionManager, device, h.Trigger)
	if err != nil {
		h.pulledDevices.Delete(key)
		return fmt.Errorf("cannot publish device %v resources for linkedAccount(%v): %w", e.deviceID, e.linkedAccount, err)
	}
	return nil
}

func (h *TaskProcessor) runTask(ctx context.Context, e Task, subscriptionManager *SubscriptionManager) error {
	err := h.poolGets.Acquire(ctx, 1)
	if err != nil {
		return fmt.Errorf("cannot acquire go routine: %w", err)
	}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer h.poolGets.Release(1)
		if h.delay > 0 {
			time.Sleep(h.delay)
		}
		ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
		defer cancel()
		switch e.taskType {
		case TaskType_PullDevice:
			err := h.pullDevice(ctx, e, subscriptionManager)
			if err != nil {
				log.Errorf("cannot run task pull device: %w", err)
			}
		case TaskType_SubscribeToResource:
			err := subscriptionManager.SubscribeToResource(ctx, e.deviceID, e.href, e.linkedAccount, e.linkedCloud)
			if err != nil {
				log.Errorf("cannot run task subscribe to resource: %w", err)
			}
		case TaskType_SubscribeToDevice:
			err := subscriptionManager.SubscribeToDevice(ctx, e.deviceID, e.linkedAccount, e.linkedCloud)
			if err != nil {
				log.Errorf("cannot run task subscribe to device: %w", err)
			}
		case TaskType_SubscribeToDevices:
			err := subscriptionManager.SubscribeToDevices(ctx, e.linkedAccount, e.linkedCloud)
			if err != nil {
				log.Errorf("cannot run task subscribe to devices: %w", err)
			}
		}
	}()
	return nil
}

func (h *TaskProcessor) readTask(ctx context.Context, subscriptionManager *SubscriptionManager) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	select {
	case e := <-h.tasksChan:
		err := h.runTask(ctx, e, subscriptionManager)
		if err != nil {
			log.Errorf("cannot process task %+v: %v", e, err)
		}
		return true, nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return true, nil
		}
		return false, ctx.Err()
	}
}

func (h *TaskProcessor) Run(ctx context.Context, subscriptionManager *SubscriptionManager) error {
	defer func() {
		process := true
		for process {
			select {
			case e := <-h.tasksChan:
				ctx, cancel := context.WithTimeout(ctx, h.timeout)
				defer cancel()
				err := h.runTask(ctx, e, subscriptionManager)
				if err != nil {
					log.Errorf("cannot process task %+v: %v", e, err)
				}
			default:
				process = false
			}
		}
		h.wg.Wait()
	}()
	for {
		ok, err := h.readTask(ctx, subscriptionManager)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}
