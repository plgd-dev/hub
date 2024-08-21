package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestPlgdTime(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	dpsCfg := test.MakeConfig(t)
	shutDown := test.New(t, dpsCfg)
	defer shutDown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()

	resp, err := c.Get(ctx, plgdtime.ResourceURI)
	require.NoError(t, err)

	var plgdTime plgdtime.PlgdTimeUpdate
	fromCbor(t, resp.Body(), &plgdTime)

	require.NotEmpty(t, plgdTime.Time)
}
