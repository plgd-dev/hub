package service

import (
	"sync"
	"testing"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/refImpl"
	"github.com/stretchr/testify/require"
)

func NewOpenApiGateway(t *testing.T, cfg refImpl.Config) func() {
	t.Log("NewOpenApiGateway")
	defer t.Log("NewOpenApiGateway done")
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
