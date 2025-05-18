package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDeviceMetadataByServiceIDs(t *testing.T) {
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
	assert.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	t.Log("event store with default namespace")
	addEventsForGetEventsToDB(ctx, t, store)

	start := time.Now()
	limit := int64(1)
	for i := range getEventsServiceIDsCount {
		docs, err := store.LoadDeviceMetadataByServiceIDs(ctx, []string{getServiceID(i)}, limit)
		require.NoError(t, err)
		require.Equal(t, limit, int64(len(docs)))
		for _, doc := range docs {
			assert.Equal(t, getServiceID(i), doc.ServiceID)
		}
	}
	fmt.Printf("event store - TestLoadDeviceMetadataByServiceIDs(limit=%v) %v\n", limit, time.Since(start))
	start = time.Now()
	limit = 1024 * 1024 * 1024
	for i := range getEventsServiceIDsCount {
		docs, err := store.LoadDeviceMetadataByServiceIDs(ctx, []string{getServiceID(i)}, limit)
		require.NoError(t, err)
		require.Equal(t, int64(getEventsResourceCount/getEventsDeviceCount), int64(len(docs)))
		for _, doc := range docs {
			assert.Equal(t, getServiceID(i), doc.ServiceID)
		}
	}

	fmt.Printf("event store - TestLoadDeviceMetadataByServiceIDs(limit=%v) %v\n", limit, time.Since(start))
}
