package future

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TEST_TIMEOUT = time.Second * 10

func TestFutureReady(t *testing.T) {
	fut, set := New()
	require.False(t, fut.Ready())

	const testValue = "test"
	set(testValue, nil)
	require.True(t, fut.Ready())

	val, err := fut.Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, testValue, val.(string))
}

func TestFutureReadyAfterError(t *testing.T) {
	fut, set := New()
	require.False(t, fut.Ready())

	set(nil, errors.New("test error"))
	require.True(t, fut.Ready())

	val, err := fut.Get(context.Background())
	require.Error(t, err)
	require.Nil(t, val)
}

func TestFutureGetTimeout(t *testing.T) {
	fut, _ := New()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := fut.Get(ctx)
	require.Error(t, err)
}

func TestFutureGetMultithreaded(t *testing.T) {
	fut, set := New()

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	const val = "test"

	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := fut.Get(ctx)
			assert.NoError(t, err)
			assert.Equal(t, val, value.(string))
		}()
	}

	set(val, nil)
}

func TestFutureGetAfterSet(t *testing.T) {
	fut, set := New()

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	const val = "test"
	set(val, nil)

	value, err := fut.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, val, value.(string))
}

func TestFutureSetFromWorkerRoutine(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	fut, set := New()
	const val = "test"
	go func() {
		time.Sleep(time.Second)
		set(val, nil)
	}()

	value, err := fut.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, val, value.(string))
}

func TestFutureRepeatedSet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	fut, set := New()
	const val = "test"
	var wg sync.WaitGroup
	const workerCount = 5
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			wg.Wait()
			set(val, nil)
		}()
		wg.Done()
	}

	value, err := fut.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, val, value.(string))
}
