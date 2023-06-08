package fsnotify

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.uber.org/atomic"
)

type Watcher struct {
	private struct {
		mutex           sync.Mutex
		paths           map[string]uint32
		w               *fsnotify.Watcher
		onEventHandlers []*func(event fsnotify.Event)
	}
	done     chan struct{}
	closed   atomic.Bool
	finished sync.WaitGroup
	logger   log.Logger
}

type Event = fsnotify.Event

const (
	Create          = fsnotify.Create
	Remove          = fsnotify.Remove
	Rename          = fsnotify.Rename
	Chmod           = fsnotify.Chmod
	Write           = fsnotify.Write
	WatchingResumed = fsnotify.Op(1 << 31)
)

// NewWatcher creates a new Watcher. It's allows to watch a single file or a directory multiple times.
func NewWatcher(logger log.Logger) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := Watcher{
		done:   make(chan struct{}),
		logger: logger,
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
	err := w.private.w.Remove(name)
	if errors.Is(err, fsnotify.ErrNonExistentWatch) {
		return nil
	}
	return err
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
	if !w.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(w.done)
	w.finished.Wait()
	err := w.private.w.Close()
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()
	w.private.paths = make(map[string]uint32)
	return err
}

func (w *Watcher) run() {
	defer w.finished.Done()
	ticks := time.NewTicker(time.Second)
	ticksRunning := true
	deletedPaths := make([]string, 0, 16)

	for {
		select {
		case <-w.done:
			return
		case event := <-w.private.w.Events:
			w.handleEvent(event, &deletedPaths, &ticksRunning, ticks)
		case err := <-w.private.w.Errors:
			if err != nil {
				w.logger.Errorf("fsnotify error: %w", err)
			}
		case <-ticks.C:
			ticksRunning = w.handleDeletedPaths(&deletedPaths, ticks)
		}
	}
}

func (w *Watcher) triggerHandlersLocked(event fsnotify.Event) {
	for _, handler := range w.private.onEventHandlers {
		(*handler)(event)
	}
}

func (w *Watcher) triggerHandlers(event fsnotify.Event) {
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()

	w.triggerHandlersLocked(event)
}

func (w *Watcher) handleEvent(event fsnotify.Event, deletedPaths *[]string, ticksRunning *bool, ticks *time.Ticker) {
	startTimer := false
	if event.Op&Remove == Remove {
		*deletedPaths = append(*deletedPaths, event.Name)
		startTimer = true
	}
	if startTimer && !*ticksRunning {
		ticks.Reset(time.Second)
		*ticksRunning = true
	}

	w.triggerHandlers(event)
}

func (w *Watcher) handleDeletedPaths(deletedPaths *[]string, ticks *time.Ticker) bool {
	reDeletedPaths := make([]string, 0, len(*deletedPaths))

	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()

	for _, path := range *deletedPaths {
		if w.private.paths[path] == 0 {
			continue
		}
		err := w.private.w.Add(path)
		if err != nil {
			w.logger.Errorf("cannot add path %v to fsnotify: %w", path, err)
			reDeletedPaths = append(reDeletedPaths, path)
		} else {
			w.triggerHandlersLocked(fsnotify.Event{
				Name: path,
				Op:   WatchingResumed,
			})
		}
	}

	if len(reDeletedPaths) == 0 {
		ticks.Stop()
		*deletedPaths = (*deletedPaths)[:0]
		return false
	}
	*deletedPaths = reDeletedPaths
	return true
}
