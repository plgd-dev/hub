package service

import (
	"sync"
	"testing"

	"github.com/go-ocf/ocf-cloud/grpc-gateway/refImpl"
	"github.com/stretchr/testify/require"
)

func NewGrpcGateway(t *testing.T, cfg refImpl.Config) func() {
	t.Log("NewGrpcGateway")
	defer t.Log("NewGrpcGateway done")
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
		s.Close()
		wg.Wait()
	}
}
