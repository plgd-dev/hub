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

func TestGetLatestDeviceETAG(t *testing.T) {
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
	limit := 1
	for i := range getEventsDeviceCount {
		etags, err := store.GetLatestDeviceETags(ctx, getDeviceID(i), uint32(limit))
		require.NoError(t, err)
		assert.Equal(t, getNLatestETag(i, limit), etags)
	}
	fmt.Printf("event store - GetLatestDeviceETAGs(limit=%v) %v\n", limit, time.Since(start))
	start = time.Now()
	limit = 0
	for i := range getEventsDeviceCount {
		etags, err := store.GetLatestDeviceETags(ctx, getDeviceID(i), uint32(limit))
		require.NoError(t, err)
		assert.Equal(t, getNLatestETag(i, limit), etags)
	}

	fmt.Printf("event store - GetLatestDeviceETAGs(limit=%v) %v\n", limit, time.Since(start))
}
