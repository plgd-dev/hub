package test

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
)

type MockEventStore struct {
	events map[string]map[string][]eventstore.EventUnmarshaler
}

var errNotSupported = errors.New("not supported")

func (s *MockEventStore) GetEvents(ctx context.Context, queries []eventstore.GetEventsQuery, timestamp int64, eventHandler eventstore.Handler) error {
	return errNotSupported
}

func (s *MockEventStore) Save(ctx context.Context, events ...eventstore.Event) (eventstore.SaveStatus, error) {
	return eventstore.Fail, errNotSupported
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
	return errNotSupported
}

func makeModelId(groupID, aggregateID string) string {
	return groupID + "." + aggregateID
}

func (s *MockEventStore) addAllModels(queriesInt map[string]eventstore.VersionQuery) map[string]eventstore.VersionQuery {
	for groupId, group := range s.events {
		for aggrId, events := range group {
			queriesInt[makeModelId(groupId, aggrId)] = eventstore.VersionQuery{AggregateID: aggrId, Version: events[0].Version()}
		}
	}
	return queriesInt
}

func (s *MockEventStore) addGroupModels(queriesInt map[string]eventstore.VersionQuery, groupID string) map[string]eventstore.VersionQuery {
	aggregates, ok := s.events[groupID]
	if !ok {
		return queriesInt
	}
	for aggrID, events := range aggregates {
		queriesInt[makeModelId(groupID, aggrID)] = eventstore.VersionQuery{AggregateID: aggrID, Version: events[0].Version()}
	}
	return queriesInt
}

func (s *MockEventStore) addAggregateModels(queriesInt map[string]eventstore.VersionQuery, aggregateID string) map[string]eventstore.VersionQuery {
	for groupID, aggregates := range s.events {
		if events, ok := aggregates[aggregateID]; ok {
			queriesInt[makeModelId(groupID, aggregateID)] = eventstore.VersionQuery{AggregateID: aggregateID, Version: events[0].Version()}
		}
	}
	return queriesInt
}

func (s *MockEventStore) addModel(queriesInt map[string]eventstore.VersionQuery, groupID, aggregateID string) map[string]eventstore.VersionQuery {
	aggregates, ok := s.events[groupID]
	if !ok {
		return queriesInt
	}
	events, ok := aggregates[aggregateID]
	if !ok {
		return queriesInt
	}

	queriesInt[makeModelId(groupID, aggregateID)] = eventstore.VersionQuery{AggregateID: aggregateID, Version: events[0].Version()}
	return queriesInt
}

func (s *MockEventStore) uniqueQueries(queries []eventstore.SnapshotQuery) map[string]eventstore.VersionQuery {
	queriesInt := make(map[string]eventstore.VersionQuery)
	if len(queries) == 0 {
		return s.addAllModels(queriesInt)
	}
	for _, query := range queries {
		if query.GroupID == "" && query.AggregateID == "" {
			return s.addAllModels(queriesInt)
		}

		if query.GroupID != "" && query.AggregateID != "" {
			queriesInt = s.addModel(queriesInt, query.GroupID, query.AggregateID)
			continue
		}

		if query.GroupID != "" {
			queriesInt = s.addGroupModels(queriesInt, query.GroupID)
			continue
		}

		if query.AggregateID != "" {
			queriesInt = s.addAggregateModels(queriesInt, query.AggregateID)
			continue
		}
	}
	return queriesInt
}

func (s *MockEventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	queriesInt := s.uniqueQueries(queries)
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
	return errNotSupported
}

// Delete aggregates events for specific group ids.
func (s *MockEventStore) Delete(ctx context.Context, queries []eventstore.DeleteQuery) error {
	return errNotSupported
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

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{make(map[string]map[string][]eventstore.EventUnmarshaler)}
}

func (s *MockEventStore) Append(groupID, aggregateID string, ev eventstore.EventUnmarshaler) {
	var m map[string][]eventstore.EventUnmarshaler
	var ok bool
	if m, ok = s.events[groupID]; !ok {
		m = make(map[string][]eventstore.EventUnmarshaler)
		s.events[groupID] = m
	}
	var r []eventstore.EventUnmarshaler
	if r, ok = m[aggregateID]; !ok {
		r = make([]eventstore.EventUnmarshaler, 0, 10)
	}
	m[aggregateID] = append(r, ev)
}
