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
		onEventHandlers []*func(event Event)
	}
	w        *fsnotify.Watcher
	done     chan struct{}
	closed   atomic.Bool
	finished sync.WaitGroup
	logger   log.Logger
}

// Event represents a file system notification.
type Event struct {
	// Path to the file or directory.
	//
	// Paths are relative to the input; for example with Add("dir") the Name
	// will be set to "dir/file" if you create that file, but if you use
	// Add("/path/to/dir") it will be "/path/to/dir/file".
	Name string

	// File operation that triggered the event.
	//
	// This is a bitmask and some systems may send multiple operations at once.
	// Use the Event.Has() method instead of comparing with ==.
	Op Op
}

// String returns a string representation of the event with their path.
func (e Event) String() string {
	return fmt.Sprintf("%-13s %q", e.Op.String(), e.Name)
}

// Op describes a set of file operations.
type Op fsnotify.Op

func (op Op) String() string {
	if fsnotify.Op(op).Has(fsnotify.Op(WatchingResumed)) {
		return "WATCHING_RESUMED"
	}
	return fsnotify.Op(op).String()
}

const (
	Create          = Op(fsnotify.Create)
	Remove          = Op(fsnotify.Remove)
	Rename          = Op(fsnotify.Rename)
	Chmod           = Op(fsnotify.Chmod)
	Write           = Op(fsnotify.Write)
	WatchingResumed = Op(1 << 31)
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
	watcher.w = w
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
	if err := w.w.Add(name); err != nil {
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
	err := w.w.Remove(name)
	if errors.Is(err, fsnotify.ErrNonExistentWatch) {
		return nil
	}
	return err
}

func (w *Watcher) AddOnEventHandler(onEventHandler *func(event Event)) {
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

func (w *Watcher) RemoveOnEventHandler(onEventHandler *func(event Event)) {
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
	err := w.w.Close()
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
		case event := <-w.w.Events:
			w.handleEvent(event, &deletedPaths, &ticksRunning, ticks)
		case err := <-w.w.Errors:
			if err != nil {
				w.logger.Errorf("fsnotify error: %w", err)
			}
		case <-ticks.C:
			ticksRunning = w.handleDeletedPaths(&deletedPaths, ticks)
		}
	}
}

func (w *Watcher) triggerHandlersLocked(event Event) {
	for _, handler := range w.private.onEventHandlers {
		(*handler)(event)
	}
}

func (w *Watcher) triggerHandlers(event Event) {
	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()

	w.triggerHandlersLocked(event)
}

func (w *Watcher) handleEvent(event fsnotify.Event, deletedPaths *[]string, ticksRunning *bool, ticks *time.Ticker) {
	startTimer := false

	if event.Op&fsnotify.Remove == fsnotify.Remove {
		*deletedPaths = append(*deletedPaths, event.Name)
		startTimer = true
	}
	if startTimer && !*ticksRunning {
		ticks.Reset(time.Second)
		*ticksRunning = true
	}
	ev := Event{
		Name: event.Name,
		Op:   Op(event.Op),
	}
	w.triggerHandlers(ev)
}

func (w *Watcher) handleDeletedPaths(deletedPaths *[]string, ticks *time.Ticker) bool {
	reDeletedPaths := make([]string, 0, len(*deletedPaths))

	w.private.mutex.Lock()
	defer w.private.mutex.Unlock()

	for _, path := range *deletedPaths {
		if w.private.paths[path] == 0 {
			continue
		}
		err := w.w.Add(path)
		if err != nil {
			w.logger.Errorf("cannot add path %v to fsnotify: %w", path, err)
			reDeletedPaths = append(reDeletedPaths, path)
		} else {
			w.triggerHandlersLocked(Event{
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
