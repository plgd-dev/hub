package eventstore

import (
	"context"
	"fmt"
	"sync"
)

// Model user defined model where events from eventstore will be projected.
type Model interface {
	Handler
}

// FactoryModelFunc creates user model.
type FactoryModelFunc func(ctx context.Context, groupID, aggregateID string) (Model, error)

// LogDebugfFunc log debug messages
type LogDebugfFunc func(fmt string, args ...interface{})

type aggregateModel struct {
	groupID     string
	aggregateID string
	model       Model
	version     uint64
	hasSnapshot bool
	lock        sync.Mutex

	LogDebugfFunc LogDebugfFunc
}

func (am *aggregateModel) Update(e EventUnmarshaler) (ignore bool, reload bool) {
	am.lock.Lock()
	defer am.lock.Unlock()

	am.LogDebugfFunc("projection.aggregateModel.Update: am.GroupId %v: AggregateId %v: Version %v, hasSnapshot %v", am.groupID, am.aggregateID, am.version, am.hasSnapshot)

	switch {
	case e.Version() == 0 || e.IsSnapshot():
		am.LogDebugfFunc("projection.aggregateModel.Update: e.Version == 0 || e.IsSnapshot()")
		am.version = e.Version()
		am.hasSnapshot = true
	case am.version+1 == e.Version() && am.hasSnapshot:
		am.LogDebugfFunc("projection.aggregateModel.Update: am.version+1 == e.Version && am.hasSnapshot")
		am.version = e.Version()
	case am.version >= e.Version() && am.hasSnapshot:
		am.LogDebugfFunc("projection.aggregateModel.Update: am.version >= e.Version && am.hasSnapshot")
		//ignore event - it was already applied
		return true, false
	default:
		am.LogDebugfFunc("projection.aggregateModel.Update: default")
		//need to reload
		return false, true
	}
	return false, false

}

func (am *aggregateModel) Handle(ctx context.Context, iter Iter) error {
	return am.model.Handle(ctx, iter)
}

// Projection projects events from eventstore to user model.
type Projection struct {
	store         EventStore
	LogDebugfFunc LogDebugfFunc

	factoryModel    FactoryModelFunc
	lock            sync.Mutex
	aggregateModels map[string]map[string]*aggregateModel
}

// NewProjection projection over eventstore.
func NewProjection(store EventStore, factoryModel FactoryModelFunc, LogDebugfFunc LogDebugfFunc) *Projection {
	if LogDebugfFunc == nil {
		LogDebugfFunc = func(fmt string, args ...interface{}) {}
	}
	return &Projection{
		store:           store,
		factoryModel:    factoryModel,
		aggregateModels: make(map[string]map[string]*aggregateModel),
		LogDebugfFunc:   LogDebugfFunc,
	}
}

type iterator struct {
	iter       Iter
	firstEvent EventUnmarshaler
	model      *aggregateModel

	nextEventToProcess EventUnmarshaler
	err                error
	reload             *VersionQuery
}

func (i *iterator) RewindToNextAggregateEvent(ctx context.Context) EventUnmarshaler {
	for {
		snapshot, nextAggregateEvent := i.RewindToSnapshot(ctx)
		if nextAggregateEvent != nil {
			return nextAggregateEvent
		}
		if snapshot == nil && nextAggregateEvent == nil {
			return nil
		}
	}
}

func (i *iterator) RewindToSnapshot(ctx context.Context) (snapshot EventUnmarshaler, nextAggregateEvent EventUnmarshaler) {
	for {
		e, ok := i.iter.Next(ctx)
		if !ok {
			return nil, nil
		}
		if e.IsSnapshot() && e.GroupID() == i.model.groupID && e.AggregateID() == i.model.aggregateID {
			return e, nil
		}
		if e.GroupID() != i.model.groupID || e.AggregateID() != i.model.aggregateID {
			return nil, e
		}
	}
}

func (i *iterator) RewindIgnore(ctx context.Context) (EventUnmarshaler, bool) {
	for {
		e, ok := i.iter.Next(ctx)
		if !ok {
			break
		}
		if e.GroupID() != i.model.groupID || e.AggregateID() != i.model.aggregateID {
			i.nextEventToProcess = e
			return nil, false
		}
		ignore, _ := i.model.Update(e)
		if !ignore {
			return e, true
		}
	}
	return nil, false
}

func (i *iterator) Next(ctx context.Context) (EventUnmarshaler, bool) {
	if i.firstEvent != nil {
		tmp := i.firstEvent
		i.firstEvent = nil
		ignore, reload := i.model.Update(tmp)
		i.model.LogDebugfFunc("projection.iterator.next: GroupId %v: AggregateId %v: Version %v, EvenType %v, ignore %v reload %v", tmp.GroupID, tmp.AggregateID, tmp.Version, tmp.EventType, ignore, reload)
		if reload {
			snapshot, nextAggregateEvent := i.RewindToSnapshot(ctx)
			if snapshot == nil {
				i.nextEventToProcess = nextAggregateEvent
				i.reload = &VersionQuery{GroupID: tmp.GroupID(), AggregateID: tmp.AggregateID(), Version: i.model.version}
				return nil, false
			}
			tmp = snapshot
			ignore, reload = i.model.Update(tmp)
			if reload {
				i.nextEventToProcess = i.RewindToNextAggregateEvent(ctx)
				i.reload = &VersionQuery{GroupID: tmp.GroupID(), AggregateID: tmp.AggregateID(), Version: i.model.version}
				return nil, false
			}
		}
		if ignore {
			return i.RewindIgnore(ctx)
		}
		return tmp, true
	}

	e, ok := i.RewindIgnore(ctx)
	if ok {
		i.model.LogDebugfFunc("projection.iterator.next: GroupId %v: AggregateId %v: Version %v, EvenType %v", e.GroupID, e.AggregateID, e.Version, e.EventType)
	}
	return e, ok
}

func (i *iterator) Err() error {
	return i.iter.Err()
}

func (p *Projection) getModel(ctx context.Context, groupID, aggregateID string) (*aggregateModel, error) {
	var ok bool
	var mapApm map[string]*aggregateModel
	var apm *aggregateModel

	p.lock.Lock()
	defer p.lock.Unlock()
	if mapApm, ok = p.aggregateModels[groupID]; !ok {
		mapApm = make(map[string]*aggregateModel)
		p.aggregateModels[groupID] = mapApm
	}
	if apm, ok = mapApm[aggregateID]; !ok {
		model, err := p.factoryModel(ctx, groupID, aggregateID)
		if err != nil {
			return nil, fmt.Errorf("cannot create model: %w", err)
		}
		p.LogDebugfFunc("projection.Projection.getModel: GroupId %v: AggregateId %v: new model", groupID, aggregateID)
		apm = &aggregateModel{groupID: groupID, aggregateID: aggregateID, model: model, LogDebugfFunc: p.LogDebugfFunc}
		mapApm[aggregateID] = apm
	}
	return apm, nil
}

func (p *Projection) handle(ctx context.Context, iter Iter) (reloadQueries []VersionQuery, err error) {
	e, ok := iter.Next(ctx)
	if !ok {
		return nil, iter.Err()
	}
	ie := e
	reloadQueries = make([]VersionQuery, 0, 32)
	for ie != nil {
		p.LogDebugfFunc("projection.iterator.handle: GroupId %v: AggregateId %v: Version %v, EvenType %v", ie.GroupID(), ie.AggregateID(), ie.Version(), ie.EventType())
		am, err := p.getModel(ctx, ie.GroupID(), ie.AggregateID())
		if err != nil {
			return nil, fmt.Errorf("cannot handle projection: %w", err)
		}
		i := iterator{
			iter:               iter,
			firstEvent:         ie,
			model:              am,
			nextEventToProcess: nil,
			err:                nil,
			reload:             nil,
		}
		err = am.Handle(ctx, &i)
		if err != nil {
			return nil, fmt.Errorf("cannot handle projection: %w", err)
		}
		//check if we are on the end
		if i.nextEventToProcess == nil {
			_, ok := i.Next(ctx)
			if ok {
				//iterator need to move to the next event
				i.nextEventToProcess = i.RewindToNextAggregateEvent(ctx)
			}
		}

		ie = i.nextEventToProcess

		if i.reload != nil {
			reloadQueries = append(reloadQueries, *i.reload)
		}
	}

	return reloadQueries, nil
}

// Handle update projection by events.
func (p *Projection) Handle(ctx context.Context, iter Iter) error {
	_, err := p.handle(ctx, iter)
	return err
}

// HandleWithReload update projection by events and reload events if it is needed.
func (p *Projection) HandleWithReload(ctx context.Context, iter Iter) error {
	//reload queries for db because version of events was greater > lastVersionSeen+1
	reloadQueries, err := p.handle(ctx, iter)
	if err != nil {
		return fmt.Errorf("cannot handle events with reload: %w", err)
	}

	if len(reloadQueries) > 0 {
		err := p.store.LoadFromVersion(ctx, reloadQueries, p)
		if err != nil {
			return fmt.Errorf("cannot reload events for db: %w", err)
		}
	}
	return nil
}

// Project update projection from snapshots defined by query. Verson in Query is ignored.
func (p *Projection) Project(ctx context.Context, queries []SnapshotQuery) (err error) {
	return p.store.LoadFromSnapshot(ctx, queries, p)
}

// Forget drop projection by query.Verson in Query is ignored.
func (p *Projection) Forget(queries []SnapshotQuery) (err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, query := range queries {
		if query.AggregateID == "" {
			delete(p.aggregateModels, query.GroupID)
		} else {
			if m, ok := p.aggregateModels[query.GroupID]; ok {
				delete(m, query.AggregateID)
				if len(m) == 0 {
					delete(p.aggregateModels, query.GroupID)
				}
			}
		}
	}

	return nil
}

func makeModelID(groupID, aggregateID string) string {
	return groupID + "." + aggregateID
}

func (p *Projection) allModels(models map[string]Model) map[string]Model {
	for groupID, group := range p.aggregateModels {
		for aggregateID, apm := range group {
			models[makeModelID(groupID, aggregateID)] = apm.model
		}
	}
	return models
}

func (p *Projection) models(queries []SnapshotQuery) map[string]Model {
	models := make(map[string]Model)
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(queries) == 0 {
		return p.allModels(models)
	}
	for _, query := range queries {
		switch {
		case query.GroupID == "" && query.AggregateID == "":
			return p.allModels(models)
		case query.GroupID != "" && query.AggregateID == "":
			if aggregates, ok := p.aggregateModels[query.GroupID]; ok {
				for aggrID, apm := range aggregates {
					models[makeModelID(query.GroupID, aggrID)] = apm.model
				}
			}
		default:
			if aggregates, ok := p.aggregateModels[query.GroupID]; ok {
				if apm, ok := aggregates[query.AggregateID]; ok {
					models[makeModelID(query.GroupID, query.AggregateID)] = apm.model
				}
			}
		}
	}

	return models
}

// Models return models from projection.
func (p *Projection) Models(queries []SnapshotQuery) []Model {
	models := p.models(queries)
	result := make([]Model, 0, len(models))
	for _, m := range models {
		result = append(result, m)
	}
	return result
}
