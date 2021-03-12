package test

import (
	"sync"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/grpc-gateway/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) refImpl.Config {
	var grpcCfg refImpl.Config
	err := envconfig.Process("", &grpcCfg)
	require.NoError(t, err)
	grpcCfg.Addr = testCfg.GRPC_HOST
	grpcCfg.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	grpcCfg.ResourceAggregateAddr = testCfg.RESOURCE_AGGREGATE_HOST

	grpcCfg.JwksURL = testCfg.JWKS_URL
	grpcCfg.Listen.File.DisableVerifyClientCertificate = true
	return grpcCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg refImpl.Config) func() {

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
