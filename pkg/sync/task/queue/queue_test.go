package queue_test

import (
	"testing"

	"github.com/plgd-dev/hub/pkg/sync/task/queue"
	"github.com/stretchr/testify/require"
)

func TestTaskQueue_Submit(t *testing.T) {
	_, err := queue.New(queue.Config{
		GoPoolSize: -1,
		Size:       10,
	})
	require.Error(t, err)
	q, err := queue.New(queue.Config{
		GoPoolSize: 1,
		Size:       2,
	})
	require.NoError(t, err)
	err = q.Submit(func() {}, func() {}, func() {})
	require.Error(t, err)
	v := make(chan int)
	err = q.Submit(func() { v <- 1 }, func() { v <- 2 })
	require.NoError(t, err)
	d := <-v
	require.Equal(t, 1, d)
	d = <-v
	require.Equal(t, 2, d)
	err = q.Submit(func() { v <- 3; v <- 4 })
	require.NoError(t, err)
	d = <-v
	require.Equal(t, 3, d)
	q.Release()
	d = <-v
	require.Equal(t, 4, d)
}
