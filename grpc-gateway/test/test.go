package test

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"

	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-ocf/cloud/grpc-gateway/refImpl"
	"github.com/go-ocf/cloud/grpc-gateway/service"
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
	grpcCfg.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	grpcCfg.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	return NewGrpcGateway(t, grpcCfg)
}

func NewGrpcGateway(t *testing.T, config refImpl.Config) func() {
	log.Setup(config.Log)
	log.Info(config.String())
	listenCertManager, err := certManager.NewCertManager(config.Listen)
	require.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	require.NoError(t, err)
	auth := kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		return ctx, nil
	})
	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	listenTLSConfig.ClientAuth = tls.NoClientCert
	server, err := kitNetGrpc.NewServer(config.Addr, grpc.Creds(credentials.NewTLS(listenTLSConfig)), auth.Stream(), auth.Unary())
	require.NoError(t, err)
	server.AddCloseFunc(func() {
		listenCertManager.Close()
		dialCertManager.Close()
	})
	err = service.AddHandler(server, config.HandlerConfig, dialCertManager.GetClientTLSConfig())
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
