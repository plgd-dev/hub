package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/go-coap/v3/dtls"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	pkgCoapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestAclsTCP(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	dpsCfg := test.MakeConfig(t)
	dpsCfg.APIs.COAP.Protocols = []pkgCoapService.Protocol{pkgCoapService.TCP}
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

	resp, err := c.Get(ctx, uri.ACLs)
	require.NoError(t, err)

	var acls acl.UpdateRequest
	fromCbor(t, resp.Body(), &acls)

	require.Len(t, acls.AccessControlList, 3)
}

func TestAclsUDP(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	dpsCfg := test.MakeConfig(t)
	dpsCfg.APIs.COAP.Protocols = []pkgCoapService.Protocol{pkgCoapService.UDP}
	dpsCfg.APIs.COAP.BlockwiseTransfer.Enabled = true
	shutDown := test.New(t, dpsCfg)
	defer shutDown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tlsCfg := setupTLSConfig(t)
	c, err := dtls.Dial(dpsCfg.APIs.COAP.Addr, pkgCoapService.TLSConfigToDTLSConfig(tlsCfg), options.WithContext(ctx))
	require.NoError(t, err)

	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()

	resp, err := c.Get(ctx, uri.ACLs)
	require.NoError(t, err)

	var acls acl.UpdateRequest
	fromCbor(t, resp.Body(), &acls)

	require.Len(t, acls.AccessControlList, 3)
}
