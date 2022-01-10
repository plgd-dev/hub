package service_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	idTest "github.com/plgd-dev/hub/v2/identity-store/test"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	test "github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconnectNATS(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	testPrepareDevice(t, co)

	ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
	defer cancel()

	test.NATSSStop(ctx, t)

	go func() {
		time.Sleep(time.Second * 3)
		test.NATSSStart(ctx, t)
	}()

	body := bytes.NewReader([]byte("data"))
	resp, err := co.Post(ctx, uri.ResourceRoute+"/"+CertIdentity+TestAResourceHref, message.TextPlain, body)
	require.NoError(t, err)
	assert.Equal(t, coapCodes.Changed.String(), resp.Code().String())
}

func TestReconnectNATSAndGrpcGateway(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
	defer cancel()
	testService.ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	auShutdown := idTest.SetUp(t)
	raShutdown := raTest.SetUp(t)
	rdShutdown := rdTest.SetUp(t)
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.Log.Embedded.Debug = true
	coapgwCfg.Log.DumpCoapMessages = true
	gwShutdown := coapgwTest.New(t, coapgwCfg)
	defer func() {
		gwShutdown()
		raShutdown()
		auShutdown()
		oauthShutdown()
	}()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	testPrepareDevice(t, co)

	test.NATSSStop(ctx, t)
	rdShutdown()

	var rdShutdownAtomic atomic.Value
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 3)
		test.NATSSStart(ctx, t)
		time.Sleep(time.Second * 3)
		rdShutdownAtomic.Store(rdTest.SetUp(t))
		time.Sleep(time.Second * 3)
	}()

	body := bytes.NewReader([]byte("data"))
	resp, err := co.Post(ctx, uri.ResourceRoute+"/"+CertIdentity+TestAResourceHref, message.TextPlain, body)
	require.NoError(t, err)
	assert.Equal(t, coapCodes.Changed.String(), resp.Code().String())
	wg.Wait()
	rdShutdownAtomic.Load().(func())()
}
