package service

import (
	"sync"
	"testing"

	"github.com/go-ocf/cloud/coap-gateway/refImpl"
	"github.com/stretchr/testify/require"
)

// NewCoapGateway creates test coap-gateway.
func NewCoapGateway(t *testing.T, cfg refImpl.Config) func() {
	t.Log("newCoapGateway")
	defer t.Log("newCoapGateway done")
	c, err := refImpl.Init(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		c.Serve()
	}()

	return func() {
		c.Shutdown()
		wg.Wait()
	}
}
