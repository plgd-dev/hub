package test

import (
	"context"
	"sync"
	"testing"

	"github.com/go-ocf/cloud/resource-aggregate/refImpl"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func SetUp(ctx context.Context, t *testing.T) (TearDown func()) {
	var raCfg refImpl.Config
	err = envconfig.Process("", &raCfg)
	require.NoError(t, err)
	//raCfg.Log.Debug = true
	raCfg.Service.Addr = testCfg.RESOURCE_AGGREGATE_HOST
	raCfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	return raService.NewResourceAggregate(t, raCfg)
}

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
