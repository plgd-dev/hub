package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestServiceServe(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|
		hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()

	log.Infof("%v\n\n", test.MakeConfig(t))

	cfg := test.MakeConfig(t).String()
	var testCfg service.Config
	err := config.Parse([]byte(cfg), &testCfg)
	require.NoError(t, err)

	shutDown := test.SetUp(t)
	defer shutDown()
}

func TestClientInactivity(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|
		hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	dpsCfg := test.MakeConfig(t)
	dpsCfg.APIs.COAP.InactivityMonitor.Timeout = time.Second * 1
	shutDown := test.New(t, dpsCfg)
	defer shutDown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()

	time.Sleep(time.Second * 2)

	_, err = c.Get(ctx, plgdtime.ResourceURI)
	require.NoError(t, err)

	select {
	case <-c.Done():
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
}
