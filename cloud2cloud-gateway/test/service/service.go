package service

import (
	"sync"
	"testing"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/refImpl"
	"github.com/stretchr/testify/require"
)

func NewCloud2cloudGateway(t *testing.T, cfg refImpl.Config) func() {
	t.Log("NewCloud2cloudGateway")
	defer t.Log("NewCloud2cloudGateway done")
	s, err := refImpl.Init(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Serve()
	}()

	return func() {
		s.Close()
		wg.Wait()
	}
}
