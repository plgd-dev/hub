package test

import (
	"context"
	"sync"
	"testing"

	"github.com/go-ocf/cloud/resource-directory/refImpl"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func SetUp(ctx context.Context, t *testing.T) (TearDown func()) {
	var rdCfg refImpl.Config
	err := envconfig.Process("", &rdCfg)
	require.NoError(t, err)
	rdCfg.Addr = testCfg.RESOURCE_DIRECTORY_HOST
	rdCfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	rdCfg.Service.FQDN = "resource-directory-" + t.Name()
	rdCfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	rdCfg.Service.ResourceAggregateAddr = testCfg.RESOURCE_AGGREGATE_HOST
	rdCfg.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	rdCfg.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	return NewResourceDirectory(t, rdCfg)
}

func NewResourceDirectory(t *testing.T, cfg refImpl.Config) func() {
	t.Log("NewResourceDirectory")
	defer t.Log("NewResourceDirectory done")
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
