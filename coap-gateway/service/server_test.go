package service_test

import (
	"os"
	"sync"
	"testing"

	authConfig "github.com/go-ocf/cloud/authorization/service"
	authService "github.com/go-ocf/cloud/authorization/test"
	"github.com/go-ocf/cloud/coap-gateway/refImpl"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	refImplRA "github.com/go-ocf/cloud/resource-aggregate/refImpl"
	raService "github.com/go-ocf/cloud/resource-aggregate/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)

	coapGWAcmeDirectory := os.Getenv("TEST_COAP_GW_OVERWRITE_LISTEN_ACME_DIRECTORY_URL")
	require.NotEmpty(t, coapGWAcmeDirectory)
	config.Listen.Acme.CADirURL = coapGWAcmeDirectory

	config.Service.Addr = "localhost:12345"
	config.Service.AuthServerAddr = "localhost:12346"
	config.Service.ResourceAggregateAddr = "localhost:12347"
	config.Service.ResourceDirectoryAddr = "localhost:12348"
	//	config.Log.Debug = true

	var authConfig authConfig.Config
	err = envconfig.Process("", &authConfig)
	require.NoError(t, err)
	authConfig.Addr = config.Service.AuthServerAddr

	shutdownAS := authService.NewAuthServer(t, authConfig)
	defer shutdownAS()

	var raCfg refImplRA.Config
	err = envconfig.Process("", &raCfg)
	require.NoError(t, err)
	raCfg.Service.Addr = config.Service.ResourceAggregateAddr
	raCfg.Service.AuthServerAddr = config.Service.AuthServerAddr
	shutdownRA := raService.NewResourceAggregate(t, raCfg)
	defer shutdownRA()

	var waitForEndServe sync.WaitGroup
	waitForEndServe.Add(1)
	defer waitForEndServe.Wait()

	server, err := refImpl.Init(config)
	require.NoError(t, err)
	defer server.Shutdown()

	go func() {
		server.Serve()
		waitForEndServe.Done()
	}()

	co := testCoapDial(t, testCfg.GW_HOST)
	defer co.Close()

	resp, err := co.Get(co.Context(), uri.ResourcePing)
	require.NoError(t, err)
	assert.Equal(t, resp.Code(), coapCodes.Content)
	resp, err = co.Get(co.Context(), uri.ResourceDirectory)
	if err == nil {
		assert.Equal(t, resp.Code(), coapCodes.Unauthorized)
	}
}
