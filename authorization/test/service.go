package test

import (
	"fmt"
	"github.com/plgd-dev/kit/log"
	"sync"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/authorization/service"
	testCfg "github.com/plgd-dev/cloud/test/config"
)

func newService(config service.Config) (*service.Server, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)

	s, err := service.New(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}

	return s, nil
}

func MakeConfig(t *testing.T) service.Config {
	var authCfg service.Config
	err := envconfig.Process("", &authCfg)
	require.NoError(t, err)
	authCfg.Service.GrpcServer.GrpcAddr = testCfg.AUTH_HOST
	authCfg.Service.HttpServer.HttpAddr = testCfg.AUTH_HTTP_HOST
	authCfg.Clients.Device.Provider = "test"
	return authCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, config service.Config) func() {
	logger, err := log.NewLogger(config.Log)
	assert.NoError(t, err)
	if err != nil {
		fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)

	auth, err := newService(config)
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
