package service

import (
	"sync"
	"testing"

	"github.com/go-ocf/cloud/resource-aggregate/refImpl"
	"github.com/stretchr/testify/require"
)

func NewResourceAggregate(t *testing.T, cfg refImpl.Config) func() {
	t.Log("NewResourceAggregate")
	defer t.Log("NewResourceAggregate done")
	s, err := refImpl.Init(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.Serve()
		require.NoError(t, err)
	}()

	return func() {
		s.Shutdown()
		wg.Wait()
	}
}
