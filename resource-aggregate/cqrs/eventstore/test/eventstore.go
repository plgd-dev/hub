package test

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

type MockEventStore struct {
	events map[string]map[string][]eventstore.EventUnmarshaler
}

func (s *MockEventStore) GetEvents(ctx context.Context, queries []eventstore.GetEventsQuery, timestamp int64, eventHandler eventstore.Handler) error {
	return errors.New("not supported")
}

func (s *MockEventStore) Save(ctx context.Context, events ...eventstore.Event) (eventstore.SaveStatus, error) {
	return eventstore.Fail, errors.New("not supported")
}

func (s *MockEventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eventHandler eventstore.Handler) error {
	aggregates := make(map[string][]eventstore.EventUnmarshaler)
	for _, device := range s.events {
		for aggrId, events := range device {
			aggregates[aggrId] = events
		}
	}

	var events []eventstore.EventUnmarshaler
	for _, q := range queries {
		var ok bool
		var r []eventstore.EventUnmarshaler

		if r, ok = aggregates[q.AggregateID]; !ok {
			continue
		}

		events = append(events, r...)
	}

	return eventHandler.Handle(ctx, &iter{events: events})
}

// LoadUpToVersion loads aggregate events up to a specific version.
func (s *MockEventStore) LoadUpToVersion(ctx context.Context, queries []eventstore.VersionQuery, eventHandler eventstore.Handler) error {
	return errors.New("not supported")
}

func makeModelId(groupId, aggregateId string) string {
	return groupId + "." + aggregateId
}

func (s *MockEventStore) allModels(queriesInt map[string]eventstore.VersionQuery) map[string]eventstore.VersionQuery {
	for groupId, group := range s.events {
		for aggrId, events := range group {
			queriesInt[makeModelId(groupId, aggrId)] = eventstore.VersionQuery{AggregateID: aggrId, Version: events[0].Version()}
		}
	}
	return queriesInt
}

func (s *MockEventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	queriesInt := make(map[string]eventstore.VersionQuery)
	if len(queries) == 0 {
		queriesInt = s.allModels(queriesInt)
	} else {
		for _, query := range queries {
			stop := false
			switch {
			case query.GroupID == "" && query.AggregateID == "":
				queriesInt = s.allModels(queriesInt)
				stop = true
			case query.GroupID != "" && query.AggregateID == "":
				if aggregates, ok := s.events[query.GroupID]; ok {
					for aggrId, events := range aggregates {
						queriesInt[makeModelId(query.GroupID, aggrId)] = eventstore.VersionQuery{AggregateID: aggrId, Version: events[0].Version()}
					}
				}
			case query.GroupID == "" && query.AggregateID != "":
				for groupId, aggregates := range s.events {
					if events, ok := aggregates[query.AggregateID]; ok {
						queriesInt[makeModelId(groupId, query.AggregateID)] = eventstore.VersionQuery{AggregateID: query.AggregateID, Version: events[0].Version()}
					}
				}
			default:
				if aggregates, ok := s.events[query.GroupID]; ok {
					if events, ok := aggregates[query.AggregateID]; ok {
						queriesInt[makeModelId(query.GroupID, query.AggregateID)] = eventstore.VersionQuery{AggregateID: query.AggregateID, Version: events[0].Version()}
					}
				}
			}

			if stop {
				break
			}
		}
	}

	ret := make([]eventstore.VersionQuery, 0, len(queriesInt))
	for _, q := range queriesInt {
		ret = append(ret, q)
	}
	if len(ret) == 0 {
		return fmt.Errorf("cannot load events: not found")
	}

	return s.LoadFromVersion(ctx, ret, eventHandler)
}

// RemoveUpToVersion deletes the aggregates events up to a specific version.
func (s *MockEventStore) RemoveUpToVersion(ctx context.Context, queries []eventstore.VersionQuery) error {
	return errors.New("not supported")
}

type iter struct {
	idx    int
	events []eventstore.EventUnmarshaler
}

func (i *iter) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	if i.idx >= len(i.events) {
		return nil, false
	}
	eu := i.events[i.idx]
	i.idx++
	return eu, true
}

func (i *iter) Err() error {
	return nil
}

func (s *MockEventStore) GetInstanceId(ctx context.Context, groupId, aggregateId string) (int64, error) {
	return -1, errors.New("not supported")
}
func (s *MockEventStore) RemoveInstanceId(ctx context.Context, instanceId int64) error {
	return errors.New("not supported")
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{make(map[string]map[string][]eventstore.EventUnmarshaler)}
}

func (e *MockEventStore) Append(groupId, aggregateId string, ev eventstore.EventUnmarshaler) {
	var m map[string][]eventstore.EventUnmarshaler
	var ok bool
	if m, ok = e.events[groupId]; !ok {
		m = make(map[string][]eventstore.EventUnmarshaler)
		e.events[groupId] = m
	}
	var r []eventstore.EventUnmarshaler
	if r, ok = m[aggregateId]; !ok {
		r = make([]eventstore.EventUnmarshaler, 0, 10)
	}
	m[aggregateId] = append(r, ev)
}
