package fsnotify

import (
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

func TestWatcher(t *testing.T) {
	w, err := NewWatcher()
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

	onEventHandler := func(event fsnotify.Event) {}
	w.AddOnEventHandler(&onEventHandler)
	w.RemoveOnEventHandler(&onEventHandler)

	err = w.Close()
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)
}
