package aggregate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/internal/math"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
)

// Command user defined command that will handled in AggregateModel.HandleCommand
type Command = interface{}

// AggregateModel user model for aggregate need to satisfy this interface.
type AggregateModel = interface {
	eventstore.Model

	HandleCommand(ctx context.Context, cmd Command, newVersion uint64) ([]eventstore.Event, error)
	TakeSnapshot(version uint64) (snapshotEvent eventstore.Event, ok bool)
	GroupID() string // defines group where model belows
}

// RetryFunc defines policy to repeat HandleCommand on concurrency exception.
type RetryFunc func() (when time.Time, err error)

// NewDefaultRetryFunc default retry function
func NewDefaultRetryFunc(limit int) RetryFunc {
	counter := new(int)
	return func() (time.Time, error) {
		if *counter >= limit {
			return time.Time{}, errors.New("retry reach limit")
		}
		*counter++
		return time.Now().Add(time.Millisecond * 10), nil
	}
}

// FactoryModelFunc creates model for aggregate
type FactoryModelFunc = func(ctx context.Context, groupID, aggregateID string) (AggregateModel, error)

// Aggregate holds data for Handle command
type Aggregate struct {
	groupID          string
	aggregateID      string
	store            eventstore.EventStore
	retryFunc        RetryFunc
	factoryModel     FactoryModelFunc
	LogDebugfFunc    eventstore.LogDebugfFunc
	additionalModels []AdditionalModel
}

type AdditionalModel struct {
	GroupID     string
	AggregateID string
}

// NewAggregate creates aggregate. it load and store events created from commands
func NewAggregate(groupID, aggregateID string, retryFunc RetryFunc, store eventstore.EventStore, factoryModel FactoryModelFunc, logDebugfFunc eventstore.LogDebugfFunc, additionalModels ...AdditionalModel) (*Aggregate, error) {
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

	return &Aggregate{
		groupID:          groupID,
		aggregateID:      aggregateID,
		store:            store,
		factoryModel:     factoryModel,
		retryFunc:        retryFunc,
		LogDebugfFunc:    logDebugfFunc,
		additionalModels: additionalModels,
	}, nil
}

type aggrIterator struct {
	iter        eventstore.Iter
	lastVersion uint64
	loaded      bool
}

func (i *aggrIterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	event, ok := i.iter.Next(ctx)
	if !ok {
		return nil, false
	}
	i.lastVersion = event.Version()
	i.loaded = true
	return event, true
}

func (i *aggrIterator) Err() error {
	return i.iter.Err()
}

type AggregateModelWrapper struct {
	model       AggregateModel
	lastVersion uint64
	loaded      bool
}

func (ah *AggregateModelWrapper) TakeSnapshot(newVersion uint64) (eventstore.Event, bool) {
	return ah.model.TakeSnapshot(newVersion)
}

func (ah *AggregateModelWrapper) GroupID() string {
	return ah.model.GroupID()
}

func (ah *AggregateModelWrapper) HandleCommand(ctx context.Context, cmd Command, newVersion uint64) ([]eventstore.Event, error) {
	return ah.model.HandleCommand(ctx, cmd, newVersion)
}

func (ah *AggregateModelWrapper) Handle(ctx context.Context, iter eventstore.Iter) error {
	i := aggrIterator{
		iter: iter,
	}
	err := ah.model.Handle(ctx, &i)
	ah.lastVersion = i.lastVersion
	ah.loaded = i.loaded
	return err
}

func HandleRetry(ctx context.Context, retryFunc RetryFunc) error {
	when, err := retryFunc()
	if err != nil {
		return fmt.Errorf("cannot retry: %w", err)
	}
	select {
	case <-time.After(time.Until(when)):
	case <-ctx.Done():
		return errors.New("retry canceled")
	}
	return nil
}

func NewAggregateModel(ctx context.Context, groupID, aggregateID string, store eventstore.EventStore, logDebugfFunc eventstore.LogDebugfFunc, factoryModel FactoryModelFunc, additionalModels ...AdditionalModel) (*AggregateModelWrapper, error) {
	models := make(map[string]AggregateModel, 1+len(additionalModels))
	model, err := factoryModel(ctx, groupID, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate model: %w", err)
	}
	models[aggregateID] = model
	amodel := &AggregateModelWrapper{model: model}
	for _, r := range additionalModels {
		model, err = factoryModel(ctx, r.GroupID, r.AggregateID)
		if err != nil {
			return nil, fmt.Errorf("cannot create aggregate model: %w", err)
		}
		models[r.AggregateID] = model
	}
	ep := eventstore.NewProjection(store, func(_ context.Context, _, projectionAggregateID string) (eventstore.Model, error) {
		if projectionAggregateID == aggregateID {
			return amodel, nil
		}
		if model, ok := models[projectionAggregateID]; ok {
			return model, nil
		}
		return nil, fmt.Errorf("cannot create aggregate model for %v %v : not found", groupID, aggregateID)
	}, logDebugfFunc)
	q := make([]eventstore.SnapshotQuery, 0, 1+len(additionalModels))
	q = append(q, eventstore.SnapshotQuery{
		GroupID:     groupID,
		AggregateID: aggregateID,
	})
	for _, r := range additionalModels {
		q = append(q, eventstore.SnapshotQuery{
			GroupID:     r.GroupID,
			AggregateID: r.AggregateID,
		})
	}
	err = ep.Project(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("cannot load aggregate model: %w", err)
	}
	return amodel, nil
}

func (a *Aggregate) FactoryModel(ctx context.Context, groupID, aggregateID string) (AggregateModel, error) {
	return a.factoryModel(ctx, groupID, aggregateID)
}

func (a *Aggregate) HandleCommandWithAggregateModelWrapper(ctx context.Context, cmd Command, amodel *AggregateModelWrapper) (events []eventstore.Event, concurrencyExcpetion bool, err error) {
	newVersion := amodel.lastVersion
	if amodel.loaded {
		// increase version for event only when some events has been processed
		newVersion++
	}
	newEvents, err := amodel.HandleCommand(ctx, cmd, newVersion)
	if err != nil {
		return nil, false, fmt.Errorf("error occurred during command handling: %w", err)
	}
	if len(newEvents) == 0 {
		return nil, false, nil
	}
	snapshot, ok := amodel.TakeSnapshot(newVersion + math.CastTo[uint64](len(newEvents)-1))
	if !ok {
		return nil, false, errors.New("cannot take snapshot")
	}
	// save all events except last one, because last one will be replaced by snapshot
	saveEvents := make([]eventstore.Event, 0, len(newEvents))
	saveEvents = append(saveEvents, newEvents[:len(newEvents)-1]...)
	saveEvents = append(saveEvents, snapshot)
	status, err := a.store.Save(ctx, saveEvents...)
	if err != nil {
		return nil, false, fmt.Errorf("cannot save events: %w", err)
	}
	switch status {
	case eventstore.Ok:
		return newEvents, false, nil
	case eventstore.SnapshotRequired:
		return nil, false, fmt.Errorf("cannot create snapshot during save events[%v] that contains last snapshot event", newEvents)
	case eventstore.ConcurrencyException:
		return nil, true, nil
	}
	return nil, false, fmt.Errorf("cannot save events[%v]: %w", newEvents, err)
}

// HandleCommand transforms command to a event, store and publish eventstore.
func (a *Aggregate) HandleCommand(ctx context.Context, cmd Command) ([]eventstore.Event, error) {
	errHandleCommand := func(err error) error {
		return fmt.Errorf("aggregate model cannot handle command: %w", err)
	}

	firstIteration := true
	for {
		if !firstIteration {
			err := HandleRetry(ctx, a.retryFunc)
			if err != nil {
				return nil, errHandleCommand(err)
			}
		}

		firstIteration = false
		amodel, err := NewAggregateModel(ctx, a.groupID, a.aggregateID, a.store, a.LogDebugfFunc, a.factoryModel, a.additionalModels...)
		if err != nil {
			return nil, errHandleCommand(err)
		}

		events, concurrencyException, err := a.HandleCommandWithAggregateModelWrapper(ctx, cmd, amodel)
		if err != nil {
			return nil, errHandleCommand(err)
		}
		if concurrencyException {
			continue
		}
		return events, nil
	}
}

func (a *Aggregate) GroupID() string {
	return a.groupID
}

func (a *Aggregate) AggregateID() string {
	return a.aggregateID
}
