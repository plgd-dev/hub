package cqldb_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/stretchr/testify/require"
)

type dummyEventHandler struct{}

func (eh *dummyEventHandler) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
	}
	return nil
}

const (
	getEventsServiceIDsCount = 10
	getEventsDeviceCount     = 10
	getEventsResourceCount   = 200
)

func getDeviceID(deviceIndex int) string {
	// return uuid
	return fmt.Sprintf("d0000000-0000-0000-0000-%012v", deviceIndex)
}

func getServiceID(serviceIndex int) string {
	return fmt.Sprintf("e0000000-0000-0000-0000-%012v", serviceIndex)
}

func getResourceID(deviceIndex, resourceIdx int) string {
	return fmt.Sprintf("a%07v-0000-0000-0000-%012v", deviceIndex, resourceIdx)
}

func getETag(deviceIndex int, resourceIndex int) []byte {
	return []byte("device" + strconv.Itoa(deviceIndex) + ".resource" + strconv.Itoa(resourceIndex))
}

func getNLatestETag(deviceIndex int, limit int) [][]byte {
	if limit == 0 {
		limit = getEventsResourceCount / getEventsDeviceCount
	}
	etags := make([][]byte, 0, limit)
	for i := 1; i <= limit; i++ {
		etags = append(etags, getETag(deviceIndex, getEventsResourceCount-(i*getEventsDeviceCount)+deviceIndex))
	}
	return etags
}

func addEventsForGetEventsToDB(ctx context.Context, t *testing.T, store *cqldb.EventStore) int {
	const eventCount = 10000
	var resourceVersion [getEventsResourceCount]uint64
	var resourceTimestamp [getEventsResourceCount]int64
	var resourceEvents [getEventsResourceCount][]eventstore.Event
	for i := range eventCount {
		deviceIndex := i % getEventsDeviceCount
		resourceIndex := i % getEventsResourceCount
		serviceIndex := i % getEventsServiceIDsCount
		if i < getEventsResourceCount {
			resourceTimestamp[i] = int64((eventCount / getEventsResourceCount) * i)
		}

		resourceEvents[resourceIndex] = append(resourceEvents[resourceIndex], test.MockEvent{
			VersionI:     resourceVersion[resourceIndex],
			EventTypeI:   "testType",
			IsSnapshotI:  true,
			AggregateIDI: getResourceID(deviceIndex, resourceIndex),
			GroupIDI:     getDeviceID(deviceIndex),
			TimestampI:   1 + resourceTimestamp[resourceIndex],
			ETagI:        getETag(deviceIndex, resourceIndex),
			ServiceIDI:   getServiceID(serviceIndex),
		})

		resourceVersion[resourceIndex]++
		resourceTimestamp[resourceIndex]++
	}

	for _, v := range resourceEvents {
		saveStatus, err := store.Save(ctx, v...)
		require.NoError(t, err)
		require.Equal(t, eventstore.Ok, saveStatus)
	}

	return eventCount
}

func getEventsByTimestamp(ctx context.Context, t *testing.T, store *cqldb.EventStore, queries []eventstore.GetEventsQuery, timestamp int64) {
	err := store.GetEvents(ctx, queries, timestamp, &dummyEventHandler{})
	require.NoError(t, err)
}

type getEventsQueryGenerator func() []eventstore.GetEventsQuery

type runGetEventsConfig struct {
	iterations int
	queries    []eventstore.GetEventsQuery
	generator  getEventsQueryGenerator
}

func runGetEvents(t *testing.T, cfg runGetEventsConfig) {
	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	ctx := context.Background()
	store, err := NewTestEventStore(ctx, fileWatcher, logger)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		err = store.Clear(ctx)
		require.NoError(t, err)
		err := store.Close(ctx)
		require.NoError(t, err)
	}()

	eventCount := addEventsForGetEventsToDB(ctx, t, store)

	weakRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := time.Now()
	for range cfg.iterations {
		if cfg.queries != nil {
			getEventsByTimestamp(ctx, t, store, cfg.queries, int64(weakRng.Intn(eventCount+1)))
		} else {
			getEventsByTimestamp(ctx, t, store, cfg.generator(), int64(weakRng.Intn(eventCount+1)))
		}
	}
	end := time.Now()
	elapsed := end.Sub(start)
	t.Logf("elapsed: %v", elapsed)
}

func TestGetEventsByTimestamp(t *testing.T) {
	runGetEvents(t, runGetEventsConfig{
		iterations: 200,
		queries: []eventstore.GetEventsQuery{
			{
				GroupID: getDeviceID(1),
			},
		},
	})
}

func TestGetDeviceEventsByTimestamp(t *testing.T) {
	runGetEvents(t, runGetEventsConfig{
		iterations: 200,
		queries: []eventstore.GetEventsQuery{
			{
				GroupID: getDeviceID(0),
			}, {
				GroupID: getDeviceID(2),
			}, {
				GroupID: getDeviceID(4),
			}, {
				GroupID: getDeviceID(6),
			}, {
				GroupID: getDeviceID(8),
			},
		},
	})
}

func TestGetResourceEventsByTimestamp(t *testing.T) {
	weakRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	runGetEvents(t, runGetEventsConfig{
		iterations: 500,
		generator: func() []eventstore.GetEventsQuery {
			resourceIndex := weakRng.Intn(getEventsResourceCount + 1)
			deviceIndex := resourceIndex % getEventsDeviceCount
			return []eventstore.GetEventsQuery{
				{
					GroupID:     getDeviceID(deviceIndex),
					AggregateID: getResourceID(deviceIndex, resourceIndex),
				},
			}
		},
	})
}

func TestGetResourcesEventsByTimestamp(t *testing.T) {
	weakRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	runGetEvents(t, runGetEventsConfig{
		iterations: 500,
		generator: func() []eventstore.GetEventsQuery {
			queries := make([]eventstore.GetEventsQuery, 5)
			for i := range queries {
				resourceIndex := weakRng.Intn(getEventsResourceCount + 1)
				deviceIndex := resourceIndex % getEventsDeviceCount
				queries[i].GroupID = getDeviceID(deviceIndex)
				queries[i].AggregateID = getResourceID(deviceIndex, resourceIndex)
			}
			return queries
		},
	})
}
