package aggregate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

// Command user defined command that will handled in AggregateModel.HandleCommand
type Command = interface{}

// AggregateModel user model for aggregate need to satisfy this interface.
type AggregateModel = interface {
	eventstore.Model

	HandleCommand(ctx context.Context, cmd Command, newVersion uint64) ([]eventstore.Event, error)
	TakeSnapshot(version uint64) (snapshotEvent eventstore.Event, ok bool)
	GroupId() string //defines group where model belows
}

// RetryFunc defines policy to repeat HandleCommand on concurrency exception.
type RetryFunc func() (when time.Time, err error)

// NewDefaultRetryFunc default retry function
func NewDefaultRetryFunc(limit int) RetryFunc {
	counter := new(int)
	return func() (time.Time, error) {
		if *counter >= limit {
			return time.Time{}, fmt.Errorf("retry reach limit")
		}
		*counter++
		return time.Now().Add(time.Millisecond * 10), nil
	}
}

// FactoryModelFunc creates model for aggregate
type FactoryModelFunc = func(ctx context.Context) (AggregateModel, error)

// Aggregate holds data for Handle command
type Aggregate struct {
	groupID             string
	aggregateID         string
	numEventsInSnapshot int
	store               eventstore.EventStore
	retryFunc           RetryFunc
	factoryModel        FactoryModelFunc
	LogDebugfFunc       eventstore.LogDebugfFunc
}

// NewAggregate creates aggregate. it load and store events created from commands
func NewAggregate(groupID, aggregateID string, retryFunc RetryFunc, numEventsInSnapshot int, store eventstore.EventStore, factoryModel FactoryModelFunc, LogDebugfFunc eventstore.LogDebugfFunc) (*Aggregate, error) {
	if groupID == "" {
		return nil, errors.New("cannot create aggregate: invalid groupID")
	}
	if aggregateID == "" {
		return nil, errors.New("cannot create aggregate: invalid aggregateId")
	}
	if retryFunc == nil {
		return nil, errors.New("cannot create aggregate: invalid retryFunc")
	}
	if store == nil {
		return nil, errors.New("cannot create aggregate: invalid eventstore")
	}
	if numEventsInSnapshot < 1 {
		return nil, errors.New("cannot create aggregate: numEventsInSnapshot < 1")
	}

	return &Aggregate{
		groupID:             groupID,
		aggregateID:         aggregateID,
		numEventsInSnapshot: numEventsInSnapshot,
		store:               store,
		factoryModel:        factoryModel,
		retryFunc:           retryFunc,
		LogDebugfFunc:       LogDebugfFunc,
	}, nil
}

type aggrIterator struct {
	iter              eventstore.Iter
	lastVersion       uint64
	numEvents         int
	snapshotEventType string
}

func (i *aggrIterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	event, ok := i.iter.Next(ctx)
	if !ok {
		return nil, false
	}
	i.lastVersion = event.Version()
	if event.EventType() != i.snapshotEventType {
		i.numEvents++
	} else {
		i.numEvents = 0
	}
	return event, true
}

func (i *aggrIterator) Err() error {
	return i.iter.Err()
}

type aggrModel struct {
	model       AggregateModel
	lastVersion uint64
	numEvents   int
}

func (ah *aggrModel) TakeSnapshot(newVersion uint64) (eventstore.Event, bool) {
	return ah.model.TakeSnapshot(newVersion)
}

func (ah *aggrModel) SnapshotEventType() string {
	return ah.model.SnapshotEventType()
}

func (ah *aggrModel) GroupId() string {
	return ah.model.GroupId()
}

func (ah *aggrModel) HandleCommand(ctx context.Context, cmd Command, newVersion uint64) ([]eventstore.Event, error) {
	return ah.model.HandleCommand(ctx, cmd, newVersion)
}

func (ah *aggrModel) Handle(ctx context.Context, iter eventstore.Iter) error {
	i := aggrIterator{
		iter:              iter,
		snapshotEventType: ah.SnapshotEventType(),
	}
	err := ah.model.Handle(ctx, &i)
	ah.lastVersion = i.lastVersion
	ah.numEvents = i.numEvents
	return err
}

func handleRetry(ctx context.Context, retryFunc RetryFunc) error {
	when, err := retryFunc()
	if err != nil {
		return fmt.Errorf("cannot retry: %w", err)
	}
	select {
	case <-time.After(when.Sub(time.Now())):
	case <-ctx.Done():
		return fmt.Errorf("retry canceled")
	}
	return nil
}

func newAggrModel(ctx context.Context, groupID, aggregateID string, store eventstore.EventStore, logDebugfFunc eventstore.LogDebugfFunc, model AggregateModel) (*aggrModel, error) {
	amodel := &aggrModel{model: model}
	ep := eventstore.NewProjection(store, func(ctx context.Context) (eventstore.Model, error) { return amodel, nil }, logDebugfFunc)
	err := ep.Project(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:           groupID,
			AggregateID:       aggregateID,
			SnapshotEventType: model.SnapshotEventType(),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("cannot load aggregate model: %w", err)
	}
	return amodel, nil
}

func (a *Aggregate) handleCommandWithAggrModel(ctx context.Context, cmd Command, amodel *aggrModel) (events []eventstore.Event, concurrencyExcpetion bool, err error) {
	newVersion := amodel.lastVersion
	if amodel.numEvents > 0 || amodel.lastVersion > 0 {
		//increase version for event only when some events has been processed
		newVersion++
	}

	events = make([]eventstore.Event, 0, 32)
	if amodel.numEvents >= a.numEventsInSnapshot {
		snapshotEvent, ok := amodel.TakeSnapshot(newVersion)
		if ok {
			concurrencyException, err := a.store.SaveSnapshot(ctx, a.groupID, a.aggregateID, snapshotEvent)
			if err != nil {
				return nil, false, fmt.Errorf("cannot save snapshot: %w", err)
			}
			if concurrencyException {
				return nil, true, nil
			}
			newVersion++
			events = append(events, snapshotEvent)
		}
	}

	newEvents, err := amodel.HandleCommand(ctx, cmd, newVersion)
	if err != nil {
		return nil, false, fmt.Errorf("cannot handle command by model: %w", err)
	}

	if len(newEvents) > 0 {
		concurrencyException, err := a.store.Save(ctx, a.groupID, a.aggregateID, newEvents)
		if err != nil {
			return nil, false, fmt.Errorf("cannot save events: %w", err)
		}

		if concurrencyException {
			return nil, true, nil
		}
	}

	return append(events, newEvents...), false, nil
}

// HandleCommand transforms command to a event, store and publish eventstore.
func (a *Aggregate) HandleCommand(ctx context.Context, cmd Command) ([]eventstore.Event, error) {
	firstIteration := true
	for {
		if !firstIteration {
			err := handleRetry(ctx, a.retryFunc)
			if err != nil {
				return nil, fmt.Errorf("aggregate model cannot handle command: %w", err)
			}
		}

		firstIteration = false
		model, err := a.factoryModel(ctx)
		if err != nil {
			return nil, fmt.Errorf("aggregate model cannot handle command: %w", err)
		}

		amodel, err := newAggrModel(ctx, a.groupID, a.aggregateID, a.store, a.LogDebugfFunc, model)
		if err != nil {
			return nil, fmt.Errorf("aggregate model cannot handle command: %w", err)
		}

		events, concurrencyException, err := a.handleCommandWithAggrModel(ctx, cmd, amodel)
		if err != nil {
			return nil, fmt.Errorf("aggregate model cannot handle command: %w", err)
		}
		if concurrencyException {
			continue
		}
		return events, nil
	}
}

func (a *Aggregate) DeviceID() string {
	return a.groupID
}
