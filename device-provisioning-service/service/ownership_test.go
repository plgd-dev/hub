package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestOwnership(t *testing.T) {
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

	resp, err := c.Get(ctx, uri.Ownership)
	require.NoError(t, err)

	var ownership doxm.Doxm
	fromCbor(t, resp.Body(), &ownership)

	require.Equal(t, doxm.Doxm{OwnerID: events.OwnerToUUID(test.MakeEnrollmentGroup().Owner)}, ownership)
}
