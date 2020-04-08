package service_test

import (
	"os"
	"sync"
	"testing"

	authConfig "github.com/go-ocf/cloud/authorization/service"
	authService "github.com/go-ocf/cloud/authorization/test/service"
	"github.com/go-ocf/cloud/coap-gateway/refImpl"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	refImplRA "github.com/go-ocf/cloud/resource-aggregate/refImpl"
	raService "github.com/go-ocf/cloud/resource-aggregate/test/service"
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
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

	co := testCoapDial(t, config.Service.Addr, config.Service.Net)
	defer co.Close()

	resp, err := co.Get(uri.ResourcePing)
	require.NoError(t, err)
	assert.Equal(t, resp.Code(), coapCodes.Content)
	resp, err = co.Get(uri.ResourceDirectory)
	if err == nil {
		assert.Equal(t, resp.Code(), coapCodes.Unauthorized)
	}
}

func testCoapDial(t *testing.T, host, net string) *gocoap.ClientConn {
	c := &gocoap.Client{Net: net, Handler: func(w gocoap.ResponseWriter, req *gocoap.Request) {
		switch req.Msg.Code() {
		case coapCodes.POST, coapCodes.GET, coapCodes.PUT, coapCodes.DELETE:
			w.SetContentFormat(gocoap.TextPlain)
			w.Write([]byte("hello world"))
		}
	}}
	conn, err := c.Dial(host)
	require.NoError(t, err)
	return conn
}

var (
	CertIdentity = "b5a2a42e-b285-42f1-a36b-034c8fc8efd5"
)
