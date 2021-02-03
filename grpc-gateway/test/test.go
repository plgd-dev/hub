package test

import (
	"github.com/plgd-dev/kit/config"
	"sync"
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/refImpl"
	"github.com/plgd-dev/cloud/grpc-gateway/service"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var grpcCfg service.Config
	err := config.Load(&grpcCfg)
	require.NoError(t, err)
	grpcCfg.Service.GrpcConfig.Addr = testCfg.GRPC_HOST
	grpcCfg.Service.GrpcConfig.TLSConfig.ClientCertificateRequired = false
	grpcCfg.Clients.RDConfig.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	grpcCfg.Clients.OAuthProvider.JwksURL = testCfg.JWKS_URL
	return grpcCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {

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
