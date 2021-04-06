package test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/authorization/service"
	"github.com/plgd-dev/cloud/pkg/log"
)

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, config service.Config) func() {
	ctx := context.Background()
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)

	auth, err := service.New(ctx, config, logger)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = auth.Serve()
		require.NoError(t, err)
	}()

	return func() {
		auth.Shutdown()
		wg.Wait()
	}
}
