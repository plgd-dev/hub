package service_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pion/dtls/v3"
	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/credential"
	deviceTest "github.com/plgd-dev/device/v2/test"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/udp"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	hubCoapGWTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/sdk"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	kitNet "github.com/plgd-dev/kit/v2/net"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestProvisioningWithRenewal(t *testing.T) {
	coapGWCfg := hubCoapGWTest.MakeConfig(t)
	coapGWCfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = true
	coapGWCfg.APIs.COAP.TLS.DisconnectOnExpiredCertificate = false
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|
		hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesGrpcGateway, hubTestService.WithCOAPGWConfig(coapGWCfg))
	defer hubShutdown()

	caCfg := caService.MakeConfig(t)
	// sign with certificate that will soon expire
	// the validity must be longer than the expiration limit (for tests this is 10s), otherwise the certificate will be replaced right away
	expiresIn := 30 * time.Second
	caCfg.Signer.ValidFrom = caCfgSignerValidFrom
	caCfg.Signer.ExpiresIn = time.Hour + expiresIn
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	dpsCfg := test.MakeConfig(t)
	const expectedTimeCount = 1
	const expectedCredentialsCount = 2 // certificate with short validity should trigger certificate renewal, but nothing else
	const expectedOwnershipCount = 1
	const expectedACLsCount = 1
	const expectedCloudConfigurationCount = 1
	waitIterations := 5 // don't end right away, wait some iterations so we know only credentials renewal was triggered and nothing else
	rh := test.NewRequestHandlerWithCounter(t, dpsCfg, nil,
		func(defaultHandlerCount, processTimeCount, processOwnershipCount, processCloudConfigurationCount, processCredentialsCount, processACLsCount uint64) (bool, error) {
			if defaultHandlerCount > 0 ||
				processTimeCount > expectedTimeCount ||
				processOwnershipCount > expectedOwnershipCount ||
				processCloudConfigurationCount > expectedCloudConfigurationCount ||
				processCredentialsCount > expectedCredentialsCount ||
				processACLsCount > expectedACLsCount {
				return false, fmt.Errorf("invalid counters default(%d:%d) time(%d:%d) owner(%d:%d) cloud(%d:%d) creds(%d:%d) acls(%d:%d)",
					defaultHandlerCount, 0,
					processTimeCount, expectedTimeCount,
					processOwnershipCount, expectedOwnershipCount,
					processCloudConfigurationCount, expectedCloudConfigurationCount,
					processCredentialsCount, expectedCredentialsCount,
					processACLsCount, expectedACLsCount,
				)
			}

			done := defaultHandlerCount == 0 &&
				processTimeCount == expectedTimeCount &&
				processOwnershipCount == expectedOwnershipCount &&
				processCloudConfigurationCount == expectedCloudConfigurationCount &&
				processCredentialsCount == expectedCredentialsCount &&
				processACLsCount == expectedACLsCount
			if done && waitIterations > 0 {
				log.Infof("checking for unexpected events")
				waitIterations--
				return false, nil
			}
			return done, nil
		})

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	dpsShutDown := test.New(t, rh.Cfg())
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			dpsShutDown()
		}
	}()
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, rh.Cfg().APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()
	deferedDpsCleanUp = false
	dpsShutDown()

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func() {
		errC := subClient.CloseSend()
		require.NoError(t, errC)
	}()
	subID, corID := test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	rh.StartDps(service.WithRequestHandler(rh))
	defer rh.StopDps()

	err = rh.Verify(ctx)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)
}

func TestProvisioningNewCertificateDuringConnectionToHub(t *testing.T) {
	defer test.ClearDB(t)

	coapGWCfg := hubCoapGWTest.MakeConfig(t)
	coapGWCfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = true
	coapGWCfg.APIs.COAP.TLS.DisconnectOnExpiredCertificate = false
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|
		hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesGrpcGateway, hubTestService.WithCOAPGWConfig(coapGWCfg))
	defer hubShutdown()

	caCfg := caService.MakeConfig(t)
	// sign with certificate that will soon expire
	// the validity must be longer than the expiration limit (for tests this is 10s), otherwise the certificate will be replaced right away
	expiresIn := 20 * time.Second
	caCfg.Signer.ValidFrom = caCfgSignerValidFrom
	caCfg.Signer.ExpiresIn = time.Hour + expiresIn
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	dpsCfg := test.MakeConfig(t)
	dpsShutDown := test.New(t, dpsCfg)
	defer dpsShutDown()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func(s grpcPb.GrpcGateway_SubscribeToEventsClient) {
		err := s.CloseSend()
		require.NoError(t, err)
	}(subClient)

	_, _ = test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "dpsResource",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{},
				ResourceIdFilter: []*grpcPb.ResourceIdFilter{
					{
						ResourceId: commands.NewResourceID(deviceID, test.ResourcePlgdDpsHref),
					},
				},
			},
		},
	})

	// certificate is expired, so we need to wait at least 10s for renewal - minimal time of expiration certification check is 10s
	log.Infof("waiting for events\n")

	expectedProvisionStatuses := []string{"renew credentials", "provisioned"}
	for _, expectedProvisionStatus := range expectedProvisionStatuses {
		ev, err := subClient.Recv()
		require.NoError(t, err)
		log.Infof("event: %+v\n", ev)
		if ev.GetResourceChanged() != nil {
			var res test.ResourcePlgdDps
			err = cbor.Decode(ev.GetResourceChanged().GetContent().GetData(), &res)
			require.NoError(t, err)
			require.Equal(t, expectedProvisionStatus, res.ProvisionStatus)
			continue
		}
		require.NoError(t, fmt.Errorf("unexpected event: %+v", ev))
	}
}

/**
 * Test device security on CA change:
 *    - use owner with the same ID that is used by the DPS but with a different certificate authority
 *    - keep the connection open
 *    - onboard by DPS, which will own the device and replace certificates
 *    - the original connection should now fail when trying to query the device
 */
func TestOwnerWithUnknownCertificateAuthority(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesResourceDirectory|
		hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|hubTestService.SetUpServicesResourceAggregate|
		hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	dpcCfg := test.MakeConfig(t)
	dpsShutDown := test.New(t, dpcCfg)
	defer dpsShutDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	rootCA, err := os.ReadFile(os.Getenv("TEST_DPS_ROOT_CA_CERT_ALT"))
	require.NoError(t, err)
	rootCAKey, err := os.ReadFile(os.Getenv("TEST_DPS_ROOT_CA_KEY_ALT"))
	require.NoError(t, err)
	devClient, err := sdk.NewClient(sdk.WithID(isEvents.OwnerToUUID(test.DPSOwner)),
		sdk.WithRootCA(rootCA, rootCAKey), sdk.WithValidity("2000-01-01T12:00:00Z", "876000h"))
	require.NoError(t, err)

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	deviceID, err = devClient.OwnDevice(ctx, deviceID, deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)

	c := grpcPb.NewGrpcGatewayClient(conn)
	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func(s grpcPb.GrpcGateway_SubscribeToEventsClient) {
		errC := s.CloseSend()
		require.NoError(t, errC)
	}(subClient)
	subID, corID := test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "registered",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_REGISTERED,
				},
			},
		},
	})

	dpsEndpoint := config.ACTIVE_DPS_SCHEME + "://" + dpcCfg.APIs.COAP.Addr
	err = devClient.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{Endpoint: &dpsEndpoint}, nil)
	require.NoError(t, err)

	err = test.WaitForRegistered(t, subClient, deviceID, subID, corID)
	require.NoError(t, err)

	var resp interface{}
	err = devClient.GetResource(ctx, deviceID, test.ResourcePlgdDpsHref, &resp)
	require.Error(t, err)
	err = devClient.Close(ctx)
	require.NoError(t, err)

	devClient, err = sdk.NewClient(sdk.WithID(isEvents.OwnerToUUID(test.DPSOwner)))
	require.NoError(t, err)
	deviceID, err = devClient.OwnDevice(ctx, deviceID, deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)
	defer func() {
		err = devClient.DisownDevice(ctx, deviceID)
		require.NoError(t, err)
		err = devClient.Close(ctx)
		require.NoError(t, err)
		time.Sleep(time.Second * 2)
	}()
	err = devClient.GetResource(ctx, deviceID, test.ResourcePlgdDpsHref, &resp)
	require.NoError(t, err)
}

type testRequestHandlerWithCustomCredentials struct {
	test.RequestHandlerWithDps
	service.RequestHandle

	subject uuid.UUID
	psk     uuid.UUID
}

func (h *testRequestHandlerWithCustomCredentials) ProcessCredentials(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	msg, err := h.RequestHandle.ProcessCredentials(ctx, req, session, linkedHubs, group)
	if err != nil {
		return nil, err
	}

	if h.psk != uuid.Nil {
		credReq, cf, err := decodeRequest(msg)
		if err != nil {
			return nil, err
		}

		credReq.Credentials = append(credReq.Credentials, credential.Credential{
			Subject: h.subject.String(),
			Type:    credential.CredentialType_SYMMETRIC_PAIR_WISE,
			PrivateData: &credential.CredentialPrivateData{
				DataInternal: h.psk,
				Encoding:     credential.CredentialPrivateDataEncoding_RAW,
			},
		})

		encode, err := coapconv.GetEncoder(cf)
		if err != nil {
			return nil, err
		}
		payload, err := encode(credReq)
		if err != nil {
			return nil, err
		}

		msg.SetBody(bytes.NewReader(payload))
	}
	return msg, nil
}

func decodeRequest(msg *pool.Message) (credential.CredentialUpdateRequest, message.MediaType, error) {
	cf, err := msg.ContentFormat()
	if err != nil {
		return credential.CredentialUpdateRequest{}, 0, err
	}

	type readerFunc = func(w io.Reader, v interface{}) error
	var reader readerFunc
	switch cf {
	case message.AppJSON:
		reader = json.ReadFrom
	case message.AppCBOR, message.AppOcfCbor:
		reader = cbor.ReadFrom
	default:
		return credential.CredentialUpdateRequest{}, 0, fmt.Errorf("unsupported type (%v)", cf)
	}

	var credReq credential.CredentialUpdateRequest
	if err = reader(msg.Body(), &credReq); err != nil {
		return credential.CredentialUpdateRequest{}, 0, err
	}
	return credReq, cf, nil
}

func dialDTLS(ctx context.Context, addr string, subject, psk []byte, opts ...udp.Option) (*coap.ClientCloseHandler, error) {
	cfg := &dtls.Config{
		PSKIdentityHint: subject,
		PSK: func([]byte) ([]byte, error) {
			return psk[:16], nil
		},
		CipherSuites: []dtls.CipherSuiteID{dtls.TLS_ECDHE_PSK_WITH_AES_128_CBC_SHA256},
	}
	return coap.DialUDPSecure(ctx, addr, cfg, opts...)
}

func findSecureUDPEndpoint(ctx context.Context) (kitNet.Addr, error) {
	eps, err := deviceTest.FindDeviceEndpoints(ctx, test.TestDeviceObtName, deviceTest.IP4)
	if err != nil {
		return kitNet.Addr{}, err
	}
	for _, ep := range eps {
		addr, err := ep.GetAddr()
		if err != nil {
			return kitNet.Addr{}, err
		}
		if addr.GetScheme() == string(schema.UDPSecureScheme) {
			return addr, nil
		}
	}
	return kitNet.Addr{}, errors.New("no endpoint found")
}

/**
 * Test device security on a credentials update:
 *    - add preshared key (PSK) to device and create a connection using PSK
 *    - force reprovisioning which updates credentials
 *    - the PSK connection should be closed to force revalidation after a credentials update
 */
func TestDisconnectAfterCredentialsUpdate(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesCoapGateway|
		hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	psk := make([]byte, 16)
	_, err = rand.Read(psk)
	require.NoError(t, err)
	dpsCfg := test.MakeConfig(t)
	pskUUID, err := uuid.FromBytes(psk)
	require.NoError(t, err)

	subjectUUID, err := uuid.Parse(isEvents.OwnerToUUID(test.DPSOwner))
	require.NoError(t, err)

	h := &testRequestHandlerWithCustomCredentials{
		RequestHandlerWithDps: test.MakeRequestHandlerWithDps(t, dpsCfg),
		psk:                   pskUUID,
		subject:               subjectUUID,
	}
	h.StartDps(service.WithRequestHandler(h))
	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, h.Cfg().APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()
	h.StopDps()

	addr, err := findSecureUDPEndpoint(ctx)
	require.NoError(t, err)
	deviceAddr := addr.GetHostname() + ":" + strconv.FormatInt(int64(addr.GetPort()), 10)
	subject, err := h.subject.MarshalBinary()
	require.NoError(t, err)
	psk, err = h.psk.MarshalBinary()
	require.NoError(t, err)
	pskConn, err := dialDTLS(ctx, deviceAddr, subject, psk)
	require.NoError(t, err)
	// get /oic/sec/cred -> succeeds
	err = pskConn.GetResource(ctx, credential.ResourceURI, nil)
	require.NoError(t, err)

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func() {
		errC := subClient.CloseSend()
		require.NoError(t, errC)
	}()
	subID, corID := test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	// reprovisiong without cred for psk
	dpsShutDown := test.New(t, h.Cfg())
	defer dpsShutDown()
	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)

	// pskConn should be closed during reprovision without cred for psk
	closed := false
	select {
	case <-pskConn.Done():
		closed = true
	case <-time.After(time.Second * 10):
	}
	require.True(t, closed)
}
