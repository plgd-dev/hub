//go:build test
// +build test

package service_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/service"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconnectNATS(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
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
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.Log.DumpBody = true
	teardown := testService.SetUpServices(ctx, t, testService.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth|service.SetUpServicesMachine2MachineOAuth|testService.SetUpServicesCertificateAuthority|testService.SetUpServicesId|testService.SetUpServicesResourceAggregate|
		testService.SetUpServicesCoapGateway, testService.WithCOAPGWConfig(coapgwCfg))
	defer teardown()
	rdShutdown := rdTest.SetUp(t)

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
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
