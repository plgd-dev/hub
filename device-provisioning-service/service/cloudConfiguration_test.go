package service_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func GetCredentials(ctx context.Context, t *testing.T, c *coapTcpClient.Conn, deviceID string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csrReq, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, deviceID, priv)
	require.NoError(t, err)

	req := service.CredentialsRequest{
		CSR: service.CSR{
			Encoding: csr.CertificateEncoding_PEM,
			Data:     string(csrReq),
		},
	}

	resp, err := c.Post(ctx, uri.Credentials, message.AppOcfCbor, bytes.NewReader(toCbor(t, req)))
	require.NoError(t, err)

	var creds credential.CredentialUpdateRequest
	fromCbor(t, resp.Body(), &creds)

	require.Len(t, creds.Credentials, 2)
}

func TestCloudPOST(t *testing.T) {
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

	resp, err := c.Post(ctx, uri.CloudConfiguration, message.AppOcfCbor, bytes.NewReader(toCbor(t, service.ProvisionCloudConfigurationRequest{DeviceID: uuid.NewString()})))
	require.NoError(t, err)

	var cfg cloud.ConfigurationUpdateRequest
	fromCbor(t, resp.Body(), &cfg)

	require.NotEmpty(t, cfg.AuthorizationCode)
	require.NotEmpty(t, cfg.AuthorizationProvider)
	require.NotEmpty(t, cfg.CloudID)
	require.NotEmpty(t, cfg.URL)
}

func TestInvalidateEnrollmentGroupHubCache(t *testing.T) {
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	dpsCfg := test.MakeConfig(t)
	dpsCfg.Clients.Storage.CacheExpiration = time.Minute
	shutDown := test.New(t, dpsCfg)
	defer shutDown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()

	resp, err := c.Post(ctx, uri.CloudConfiguration, message.AppOcfCbor, bytes.NewReader(toCbor(t, service.ProvisionCloudConfigurationRequest{DeviceID: uuid.NewString()})))
	require.NoError(t, err)

	var cfg cloud.ConfigurationUpdateRequest
	fromCbor(t, resp.Body(), &cfg)

	require.NotEmpty(t, cfg.AuthorizationCode)
	require.NotEmpty(t, cfg.AuthorizationProvider)
	require.NotEmpty(t, cfg.CloudID)
	require.NotEmpty(t, cfg.URL)

	eg, hubs, err := dpsCfg.EnrollmentGroups[0].ToProto()
	require.NoError(t, err)

	// test invalidate hub cache
	hub := hubs[0]
	hub.Gateways = []string{"invalidateCacheHub"}
	err = store.UpdateHub(ctx, hub.GetOwner(), hub)
	require.NoError(t, err)

	// wait for sync caches
	time.Sleep(time.Millisecond * 500)

	resp, err = c.Post(ctx, uri.CloudConfiguration, message.AppOcfCbor, bytes.NewReader(toCbor(t, service.ProvisionCloudConfigurationRequest{DeviceID: uuid.NewString()})))
	require.NoError(t, err)
	var cfg1 cloud.ConfigurationUpdateRequest
	fromCbor(t, resp.Body(), &cfg1)
	require.Equal(t, hub.GetGateways()[0], cfg1.URL)

	// test invalidate enrollment group cache
	hub.Id = "00000000-0000-0000-0000-000000012345"
	hub.HubId = hub.GetId()
	hub.Gateways[0] = "invalidateEnrollmentGroup"
	err = store.CreateHub(ctx, hub.GetOwner(), hub)
	require.NoError(t, err)
	eg.HubIds = []string{hub.GetId()}
	err = store.UpdateEnrollmentGroup(ctx, eg.GetOwner(), eg)
	require.NoError(t, err)

	// wait for sync caches
	time.Sleep(time.Millisecond * 500)

	c2, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c2.Close()
		require.NoError(t, errC)
	}()
	resp, err = c2.Post(ctx, uri.CloudConfiguration, message.AppOcfCbor, bytes.NewReader(toCbor(t, service.ProvisionCloudConfigurationRequest{DeviceID: uuid.NewString()})))
	require.NoError(t, err)
	var cfg2 cloud.ConfigurationUpdateRequest
	fromCbor(t, resp.Body(), &cfg2)
	require.Equal(t, hub.GetGateways()[0], cfg2.URL)
}
