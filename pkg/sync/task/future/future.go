package future

import (
	"context"

	"go.uber.org/atomic"
)

type Value interface{}

type SetFunc func(Value, error)

// Thread-safe facility to obtain/wait for a value from a single producer in
// multiple goroutines
type Future struct {
	value Value
	err   error

	ready    chan struct{}
	blockSet atomic.Bool // block repeated calls of set
}

// New constructs new Future.
//
// Use return setter to set value of the future and unblock all routines waiting
// on the future. The setter will succeed only once, repeated calls of set do nothing.
func New() (*Future, SetFunc) {
	f := &Future{ready: make(chan struct{})}
	return f, f.set
}

// Get returns value when it is set.
func (f *Future) Get(ctx context.Context) (Value, error) {
	if f.Ready() {
		return f.value, f.err
	}

	select {
	case <-f.ready:
		return f.value, f.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Set value or error of the future and change state to ready.
func (f *Future) set(value Value, err error) {
	if !f.blockSet.CAS(false, true) {
		return
	}

	f.value = value
	f.err = err
	close(f.ready)
}

// Ready checks whether value is set
func (f *Future) Ready() bool {
	select {
	case <-f.ready:
		return true
	default:
		return false
	}
}
