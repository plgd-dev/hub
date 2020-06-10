package test

import (
	"sync"
	"testing"

	"github.com/go-ocf/kit/log"

	"github.com/go-ocf/cloud/grpc-gateway/refImpl"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func SetUp(t *testing.T) (TearDown func()) {
	var grpcCfg refImpl.Config
	err := envconfig.Process("", &grpcCfg)
	require.NoError(t, err)
	grpcCfg.Addr = testCfg.GRPC_HOST
	grpcCfg.Service.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	grpcCfg.JwksURL = testCfg.JWKS_URL
	return NewGrpcGateway(t, grpcCfg)
}

func NewGrpcGateway(t *testing.T, config refImpl.Config) func() {
	log.Setup(config.Log)
	log.Info(config.String())
	server, err := refImpl.Init(config)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.Serve()
	}()

	return func() {
		server.Close()
		wg.Wait()
	}
}
