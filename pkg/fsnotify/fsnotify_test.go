package fsnotify

import (
	"os"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestWatcher(t *testing.T) {
	w, err := NewWatcher(log.Get())
	require.NoError(t, err)
	err = w.Add("/tmp")
	require.NoError(t, err)
	err = w.Add("/tmp")
	require.NoError(t, err)
	err = w.Remove("/tmp")
	require.NoError(t, err)
	err = w.Remove("/tmp")
	require.NoError(t, err)
	err = w.Remove("/tmp")
	require.Error(t, err)
	err = w.Add("/tmp")
	require.NoError(t, err)

	onEventHandler := func(Event) {}
	w.AddOnEventHandler(&onEventHandler)
	w.RemoveOnEventHandler(&onEventHandler)

	err = w.Close()
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)
}

type testOnEvent struct {
	ch chan Event
}

func (o *testOnEvent) onEvent(event Event) {
	o.ch <- event
}

func (o *testOnEvent) waitEvent(timeout time.Duration, op Op) (Event, bool) {
	for {
		select {
		case event := <-o.ch:
			if event.Op&op == 0 {
				continue
			}
			return event, true
		case <-time.After(timeout):
			return Event{}, false
		}
	}
}

func newTestOnEvent() *testOnEvent {
	return &testOnEvent{
		ch: make(chan Event, 32),
	}
}

func TestWatcherFile(t *testing.T) {
	w, err := NewWatcher(log.Get())
	require.NoError(t, err)

	h := newTestOnEvent()
	onEvent := h.onEvent
	w.AddOnEventHandler(&onEvent)

	// test not exist
	err = w.Add("/tmp/TestWatcherFile.notexist")
	require.Error(t, err)

	// test create tmp file
	file, err := os.CreateTemp("", "WatcherFile")
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)
	defer func() {
		err = os.Remove(file.Name())
		require.NoError(t, err)
	}()
	err = w.Add(file.Name())
	require.NoError(t, err)

	// test remove file
	err = os.Remove(file.Name())
	require.NoError(t, err)

	event, ok := h.waitEvent(time.Second, Remove)
	require.True(t, ok)
	require.Equal(t, file.Name(), event.Name)

	// test watching resumed
	file, err = os.Create(file.Name())
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	event, ok = h.waitEvent(time.Second+time.Second/2, WatchingResumed)
	require.True(t, ok)
	require.Equal(t, file.Name(), event.Name)

	// test remove watcher
	err = os.Remove(file.Name())
	require.NoError(t, err)
	event, ok = h.waitEvent(time.Second, Remove)
	require.True(t, ok)
	require.Equal(t, file.Name(), event.Name)

	err = w.Remove(file.Name())
	require.NoError(t, err)

	file, err = os.Create(file.Name())
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	_, ok = h.waitEvent(time.Second+time.Second/2, WatchingResumed)
	require.False(t, ok)

	err = w.Close()
	require.NoError(t, err)
}
