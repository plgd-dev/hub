// Copyright (c) 2015 - The Event Horizon authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cqrs/eventstore/maintenance"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	cqrsUtils "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	"github.com/plgd-dev/cqrs/event"
	"github.com/plgd-dev/cqrs/eventstore"
	cqrsMongodb "github.com/plgd-dev/cqrs/eventstore/mongodb"
)

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	es     *cqrsMongodb.EventStore
	client *mongo.Client
	config Config
}

//NewEventStore create a event store from configuration
func NewEventStore(config Config, goroutinePoolGo eventstore.GoroutinePoolGoFunc, opts ...Option) (*EventStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	config.marshalerFunc = cqrsUtils.Marshal
	config.unmarshalerFunc = cqrsUtils.Unmarshal
	for _, o := range opts {
		config = o(config)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI).SetMaxPoolSize(config.MaxPoolSize).SetMaxConnIdleTime(config.MaxConnIdleTime).SetTLSConfig(config.tlsCfg))
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}

	es, err := cqrsMongodb.NewEventStoreWithClient(ctx, client, config.DatabaseName, "events", config.BatchSize, goroutinePoolGo, config.marshalerFunc, config.unmarshalerFunc, nil)
	if err != nil {
		return nil, err
	}
	return &EventStore{
		es:     es,
		client: client,
		config: config,
	}, nil
}

// Save saves events to a path.
func (s *EventStore) Save(ctx context.Context, groupId, aggregateId string, events []event.Event) (concurrencyException bool, err error) {
	return s.es.Save(ctx, groupId, aggregateId, events)
}

// SaveSnapshot saves snapshots to a path.
func (s *EventStore) SaveSnapshot(ctx context.Context, groupId, aggregateId string, event event.Event) (concurrencyException bool, err error) {
	return s.es.SaveSnapshot(ctx, groupId, aggregateId, event)
}

// LoadFromVersion loads aggragate events from a specific version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eventHandler event.Handler) error {
	return s.es.LoadFromVersion(ctx, queries, eventHandler)
}

// LoadUpToVersion loads aggragate events up to a specific version.
func (s *EventStore) LoadUpToVersion(ctx context.Context, queries []eventstore.VersionQuery, eventHandler event.Handler) error {
	return s.es.LoadUpToVersion(ctx, queries, eventHandler)
}

// LoadFromSnapshot loads events from beginning.
func (s *EventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler event.Handler) error {
	return s.es.LoadFromSnapshot(ctx, queries, eventHandler)
}

// RemoveUpToVersion deletes the aggragates events up to a specific version.
func (s *EventStore) RemoveUpToVersion(ctx context.Context, queries []eventstore.VersionQuery) error {
	return s.es.RemoveUpToVersion(ctx, queries)
}

// Insert stores (or updates) the information about the latest snapshot version per aggregate into the DB
func (s *EventStore) Insert(ctx context.Context, task maintenance.Task) error {
	return s.es.Insert(ctx, task)
}

// Query retrieves the latest snapshot version per aggregate for thw number of aggregates specified by 'limit'
func (s *EventStore) Query(ctx context.Context, limit int, taskHandler maintenance.TaskHandler) error {
	return s.es.Query(ctx, limit, taskHandler)
}

// Remove deletes (the latest snapshot version) database record for a given aggregate ID
func (s *EventStore) Remove(ctx context.Context, task maintenance.Task) error {
	return s.es.Remove(ctx, task)
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	return s.es.Clear(ctx)
}

// Close closes the database session.
func (s *EventStore) Close(ctx context.Context) error {
	return s.es.Close(ctx)
}
