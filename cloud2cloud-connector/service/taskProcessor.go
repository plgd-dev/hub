package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	kitSync "github.com/plgd-dev/kit/v2/sync"
	"go.opentelemetry.io/otel/trace"
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

type OnTaskTrigger func(Task)

type TaskProcessor struct {
	tasksChan      chan Task
	wg             *sync.WaitGroup
	poolGets       *semaphore.Weighted
	timeout        time.Duration
	raClient       raService.ResourceAggregateClient
	pulledDevices  *kitSync.Map // [userid+deviceID]
	delay          time.Duration
	tracerProvider trace.TracerProvider
}

func NewTaskProcessor(raClient raService.ResourceAggregateClient, tracerProvider trace.TracerProvider, maxParallelGets, cacheSize int, timeout, delay time.Duration) *TaskProcessor {
	return &TaskProcessor{
		pulledDevices:  kitSync.NewMap(),
		tasksChan:      make(chan Task, cacheSize),
		wg:             &sync.WaitGroup{},
		poolGets:       semaphore.NewWeighted(int64(maxParallelGets)),
		timeout:        timeout,
		raClient:       raClient,
		delay:          delay,
		tracerProvider: tracerProvider,
	}
}

func (h *TaskProcessor) Trigger(task Task) {
	h.tasksChan <- task
}

func (h *TaskProcessor) pullDevice(ctx context.Context, e Task) error {
	key := getKey(e.linkedAccount.UserID, e.deviceID)
	_, loaded := h.pulledDevices.LoadOrStore(key, e)
	if loaded {
		return nil
	}
	var device RetrieveDeviceWithLinksResponse
	err := Get(ctx, h.tracerProvider, e.linkedCloud.Endpoint.URL+prefixDevicesPath+e.deviceID, e.linkedAccount, e.linkedCloud, &device)
	if err != nil {
		h.pulledDevices.Delete(key)
		return fmt.Errorf("cannot pull device %v for linked linkedAccount(%v): %w", e.deviceID, e.linkedAccount, err)
	}
	err = publishDeviceResources(ctx, h.raClient, e.deviceID, e.linkedAccount, e.linkedCloud, device, h.Trigger)
	if err != nil {
		h.pulledDevices.Delete(key)
		return fmt.Errorf("cannot publish device %v resources for linkedAccount(%v): %w", e.deviceID, e.linkedAccount, err)
	}
	return nil
}

func (h *TaskProcessor) handleTask(e Task, subscriptionManager *SubscriptionManager) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	switch e.taskType {
	case TaskType_PullDevice:
		err := h.pullDevice(ctx, e)
		if err != nil {
			log.Errorf("cannot run task pull device: %v", err)
		}
	case TaskType_SubscribeToResource:
		err := subscriptionManager.SubscribeToResource(ctx, e.deviceID, e.href, e.linkedAccount, e.linkedCloud)
		if err != nil {
			log.Errorf("cannot run task subscribe to resource: %v", err)
		}
	case TaskType_SubscribeToDevice:
		err := subscriptionManager.SubscribeToDevice(ctx, e.deviceID, e.linkedAccount, e.linkedCloud)
		if err != nil {
			log.Errorf("cannot run task subscribe to device: %v", err)
		}
	case TaskType_SubscribeToDevices:
		err := subscriptionManager.SubscribeToDevices(ctx, e.linkedAccount, e.linkedCloud)
		if err != nil {
			log.Errorf("cannot run task subscribe to devices: %v", err)
		}
	}
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
		h.handleTask(e, subscriptionManager)
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
			log.Errorf("cannot process task %+v: %w", e, err)
		}
		return true, nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
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
					log.Errorf("cannot process task %+v: %w", e, err)
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
