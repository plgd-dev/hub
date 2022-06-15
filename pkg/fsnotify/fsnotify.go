package fsnotify

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.uber.org/atomic"
)

type Watcher struct {
	private struct {
		mutex           sync.RWMutex
		paths           map[string]uint32
		w               *fsnotify.Watcher
		onEventHandlers []*func(event fsnotify.Event)
	}
	done     chan struct{}
	closed   atomic.Bool
	finished sync.WaitGroup
}

type Event = fsnotify.Event

const (
	Create = fsnotify.Create
	Remove = fsnotify.Remove
	Rename = fsnotify.Rename
	Chmod  = fsnotify.Chmod
	Write  = fsnotify.Write
)

// NewWatcher creates a new Watcher. It's allows to watch a single file or a directory multiple times.
func NewWatcher() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := Watcher{
		done: make(chan struct{}),
	}
	watcher.private.w = w
	watcher.private.paths = make(map[string]uint32)
	watcher.finished.Add(1)
	go watcher.run()
	return &watcher, nil
}

func (w *Watcher) Add(name string) error {
	name = filepath.Clean(name)
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()
	_, ok := w.private.paths[name]
	if ok {
		w.private.paths[name]++
		return nil
	}
	if err := w.private.w.Add(name); err != nil {
		return err
	}
	w.private.paths[name] = 1
	return nil
}

func (w *Watcher) Remove(name string) error {
	name = filepath.Clean(name)
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()
	_, ok := w.private.paths[name]
	if !ok {
		return fmt.Errorf("%v is not watched", name)
	}
	w.private.paths[name]--
	if w.private.paths[name] > 0 {
		return nil
	}
	delete(w.private.paths, name)
	return w.private.w.Remove(name)
}

func (w *Watcher) AddOnEventHandler(onEventHandler *func(event fsnotify.Event)) {
	if onEventHandler == nil {
		return
	}
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()
	for _, handler := range w.private.onEventHandlers {
		if handler == onEventHandler {
			return
		}
	}
	w.private.onEventHandlers = append(w.private.onEventHandlers, onEventHandler)
}

func (w *Watcher) RemoveOnEventHandler(onEventHandler *func(event fsnotify.Event)) {
	if onEventHandler == nil {
		return
	}
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()
	for i, handler := range w.private.onEventHandlers {
		if handler == onEventHandler {
			w.private.onEventHandlers = append(w.private.onEventHandlers[:i], w.private.onEventHandlers[i+1:]...)
			return
		}
	}
}

func (w *Watcher) Close() error {
	if !w.closed.CAS(false, true) {
		return nil
	}
	err := w.private.w.Close()
	close(w.done)
	defer w.finished.Wait()
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()
	w.private.paths = make(map[string]uint32)
	return err
}

func (w *Watcher) run() {
	defer w.finished.Done()
	for {
		select {
		case <-w.done:
			return
		case event := <-w.private.w.Events:
			w.private.mutex.RLock()
			for _, handler := range w.private.onEventHandlers {
				w.private.mutex.RUnlock()
				(*handler)(event)
				w.private.mutex.RLock()
			}
			w.private.mutex.RUnlock()
		case err := <-w.private.w.Errors:
			if err == nil {
				continue
			}
			log.Errorf("fsnotify error: %w", err)
		}
	}
}
