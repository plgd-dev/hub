package test

import (
	"context"
	"sync"
	"testing"

	"github.com/plgd-dev/hub/v2/identity-store/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, config service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(config.Log)

	idServer, err := service.New(ctx, config, logger)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = idServer.Serve()
	}()

	return func() {
		idServer.Shutdown()
		wg.Wait()
	}
}
