package test

import (
	"context"
	"sync"

	"github.com/plgd-dev/hub/v2/identity-store/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t require.TestingT, config service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(config.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	idServer, err := service.New(ctx, config, fileWatcher, logger)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = idServer.Serve()
	}()

	return func() {
		_ = idServer.Close()
		wg.Wait()
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}
}
