package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	httpService "github.com/plgd-dev/hub/v2/device-provisioning-service/service/http"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/device/ocf"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/sdk"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	DPSCoapGwHost = "127.0.0.1:20132"
	DPSHost       = "127.0.0.1:20030"
)

func TestProvisioning(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	err := test.SendSignalToDocker(test.TestDockerContainerName, "HUP")
	require.NoError(t, err)
	time.Sleep(time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
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

	corID := "allEvents"
	subClient, subID := test.SubscribeToAllEvents(ctx, t, c, corID)

	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Addr = DPSCoapGwHost
	coapgwCfg.APIs.COAP.ExternalAddress = DPSCoapGwHost
	coapgwShutdown := coapgwTest.New(t, coapgwCfg)
	defer coapgwShutdown()

	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	dpcCfg := test.MakeConfig(t)
	dpcCfg.APIs.COAP.Addr = DPSHost
	dpcCfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	dpcCfg.EnrollmentGroups[0].Hubs[0].Gateways = []string{config.ACTIVE_COAP_SCHEME + "://" + coapgwCfg.APIs.COAP.Addr}
	dpsShutDown := test.New(t, dpcCfg)
	defer dpsShutDown()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceName)
	hubTest.WaitForDevice(t, subClient, ocf.NewDevice(deviceID, test.TestDeviceName), subID, corID, test.TestDevsimResources)
	err = subClient.CloseSend()
	require.NoError(t, err)

	request := httpgwTest.NewRequest(http.MethodGet, httpService.ProvisioningRecords, nil).
		Host(test.DPSHTTPHost).AuthToken(token).Build()
	resp := httpgwTest.HTTPDo(t, request)
	defer func() {
		_ = resp.Body.Close()
	}()

	var got []*pb.ProvisioningRecord
	for {
		var dev pb.ProvisioningRecord
		err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &dev)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		got = append(got, &dev)
	}
	require.Len(t, got, 1)
	require.NotEmpty(t, got[0].GetId())
	require.NotEmpty(t, got[0].GetDeviceId())
	require.NotEmpty(t, got[0].GetEnrollmentGroupId())
	require.NotEmpty(t, got[0].GetCreationDate())
	if !config.DPS_UDP_ENABLED {
		// TODO: fix in iotivity-lite? no local endpoints reported for UDP
		require.NotEmpty(t, got[0].GetLocalEndpoints())
	}
	require.NotEmpty(t, got[0].GetAcl().GetAccessControlList())
	require.NotEmpty(t, got[0].GetAcl().GetStatus().GetDate())
	require.NotEmpty(t, got[0].GetAcl().GetStatus().GetCoapCode())
	require.Equal(t, "", got[0].GetAcl().GetStatus().GetErrorMessage())
	require.NotEmpty(t, got[0].GetCloud().GetStatus().GetDate())
	require.NotEmpty(t, got[0].GetCloud().GetStatus().GetCoapCode())
	require.Equal(t, "", got[0].GetCloud().GetStatus().GetErrorMessage())
	require.NotEmpty(t, got[0].GetCloud().GetGateways())
	require.NotEmpty(t, got[0].GetCloud().GetProviderName())
	require.NotEmpty(t, got[0].GetCloud().GetGateways()[0].GetId())
	require.NotEmpty(t, got[0].GetCloud().GetGateways()[0].GetUri())
	require.Equal(t, int32(0), got[0].GetCloud().GetSelectedGateway())
	require.NotEmpty(t, got[0].GetAttestation().GetDate())
	require.NotEmpty(t, got[0].GetAttestation().GetX509().GetCertificatePem())
	require.NotEmpty(t, got[0].GetCredential().GetStatus().GetDate())
	require.NotEmpty(t, got[0].GetCredential().GetStatus().GetCoapCode())
	require.NotEmpty(t, got[0].GetCredential().GetIdentityCertificatePem())
	require.NotEmpty(t, got[0].GetCredential().GetCredentials())
	require.Equal(t, "", got[0].GetCredential().GetStatus().GetErrorMessage())
	require.NotEmpty(t, got[0].GetOwnership().GetOwner())
	require.NotEmpty(t, got[0].GetOwnership().GetStatus().GetDate())
	require.NotEmpty(t, got[0].GetOwnership().GetStatus().GetCoapCode())
	require.Equal(t, "", got[0].GetOwnership().GetStatus().GetErrorMessage())
	require.NotEmpty(t, got[0].GetPlgdTime().GetDate())
	require.NotEmpty(t, got[0].GetPlgdTime().GetCoapCode())
	require.Equal(t, "", got[0].GetPlgdTime().GetErrorMessage())
}

func TestProvisioningFactoryReset(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	dpcCfg := test.MakeConfig(t)
	dpsShutDown := test.New(t, dpcCfg)
	defer dpsShutDown()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)

	token := oauthTest.GetDefaultAccessToken(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	deviceID, _ = test.OnboardDpsSim(ctx, t, c, deviceID, dpcCfg.APIs.COAP.Addr, test.TestDevsimResources)

	// extended validity is needed when running tests with libfaketime
	devClient, err := sdk.NewClient(sdk.WithID(events.OwnerToUUID(test.DPSOwner)), sdk.WithValidity("2000-01-01T12:00:00Z", "876000h"))
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()

	err = devClient.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{Endpoint: new(string)}, nil)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}
	time.Sleep(time.Second * 3)

	deviceID = hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	deviceID, err = devClient.OwnDevice(ctx, deviceID, deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)
	defer func() {
		_ = devClient.DisownDevice(ctx, deviceID)
	}()
	var resp test.ResourcePlgdDps
	err = devClient.GetResource(ctx, deviceID, test.ResourcePlgdDpsHref, &resp)
	require.NoError(t, err)
	test.CleanUpDpsResource(&resp)
	require.Equal(t, test.ResourcePlgdDps{
		Endpoint:        new(string),
		Interfaces:      []string{interfaces.OC_IF_R, interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		ProvisionStatus: "uninitialized",
		ResourceTypes:   []string{test.ResourcePlgdDpsType},
	}, resp)
}

func TestProvisioningWithCloudChange(t *testing.T) {
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
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

	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	var cleanUp fn.FuncList
	deferedCleanUp := true
	defer func() {
		if deferedCleanUp {
			cleanUp.Execute()
		}
	}()
	coapgwShutdown := coapgwTest.SetUp(t)
	cleanUp.AddFunc(coapgwShutdown)

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	dpsCfg := test.MakeConfig(t)
	dpsShutDown := test.New(t, dpsCfg)
	cleanUp.AddFunc(dpsShutDown)
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

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
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)

	deferedCleanUp = false
	dpsShutDown()
	coapgwShutdown()
	// 10secs after connection to coap-gw is lost, DPS client should attempt full reprovision

	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Addr = DPSCoapGwHost
	coapgwCfg.APIs.COAP.ExternalAddress = DPSCoapGwHost
	coapgwShutdown = coapgwTest.New(t, coapgwCfg)
	defer coapgwShutdown()

	store, storeTearDown := test.NewMongoStore(t)
	defer storeTearDown()
	count, err := store.DeleteEnrollmentGroups(ctx, test.DPSOwner, &pb.GetEnrollmentGroupsRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	count, err = store.DeleteHubs(ctx, test.DPSOwner, &pb.GetHubsRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways = []string{config.ACTIVE_COAP_SCHEME + "://" + coapgwCfg.APIs.COAP.Addr}
	dpsShutDown = test.New(t, dpsCfg)
	defer dpsShutDown()

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)
}

func writeToTempFile(t *testing.T, fileName string, data []byte) string {
	f, err := os.CreateTemp("", fileName)
	require.NoError(t, err)
	defer func() {
		err = f.Close()
		require.NoError(t, err)
	}()
	_, err = f.Write(data)
	require.NoError(t, err)
	return f.Name()
}

func TestProvisioningWithPSK(t *testing.T) {
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
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

	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	coapgwShutdown := coapgwTest.SetUp(t)
	defer coapgwShutdown()
	dpsCfg := test.MakeConfig(t)
	psk := testPSK
	pskFile := writeToTempFile(t, "psk.key", []byte(psk))
	defer func() {
		err = os.Remove(pskFile)
		require.NoError(t, err)
	}()
	dpsCfg.EnrollmentGroups[0].PreSharedKeyFile = urischeme.URIScheme(pskFile)
	dpsCfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	dpsShutDown := test.New(t, dpsCfg)
	defer dpsShutDown()
	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	_, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

	addr, err := findSecureUDPEndpoint(ctx)
	require.NoError(t, err)
	deviceAddr := addr.GetHostname() + ":" + strconv.FormatInt(int64(addr.GetPort()), 10)

	uuid.MustParse(events.OwnerToUUID(dpsCfg.EnrollmentGroups[0].Owner))
	subject, err := uuid.MustParse(events.OwnerToUUID(dpsCfg.EnrollmentGroups[0].Owner)).MarshalBinary()
	require.NoError(t, err)

	pskConn, err := dialDTLS(ctx, deviceAddr, subject, []byte(psk))
	require.NoError(t, err)
	defer func() {
		err = pskConn.Close()
		require.NoError(t, err)
	}()
	// get /oic/sec/cred -> succeeds
	err = pskConn.GetResource(ctx, credential.ResourceURI, nil)
	require.NoError(t, err)

	token := oauthTest.GetDefaultAccessToken(t)
	request := httpgwTest.NewRequest(http.MethodGet, httpService.ProvisioningRecords, nil).
		Host(test.DPSHTTPHost).AuthToken(token).Build()
	resp := httpgwTest.HTTPDo(t, request)
	defer func() {
		_ = resp.Body.Close()
	}()

	var got []*pb.ProvisioningRecord
	for {
		var dev pb.ProvisioningRecord
		err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &dev)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		got = append(got, &dev)
	}
	require.Len(t, got, 1)
	require.Equal(t, events.OwnerToUUID(dpsCfg.EnrollmentGroups[0].Owner), got[0].GetCredential().GetPreSharedKey().GetSubjectId())
	require.Equal(t, psk, got[0].GetCredential().GetPreSharedKey().GetKey())
}

func TestProvisioningFromNewDPSAddress(t *testing.T) {
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
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

	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	coapgwShutdown := coapgwTest.SetUp(t)
	defer coapgwShutdown()
	dpsCfg := test.MakeConfig(t)
	psk := testPSK
	pskFile := writeToTempFile(t, "psk.key", []byte(psk))
	defer func() {
		err = os.Remove(pskFile)
		require.NoError(t, err)
	}()
	dpsCfg.EnrollmentGroups[0].PreSharedKeyFile = urischeme.URIScheme(pskFile)
	dpsCfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	dpsShutDown := test.New(t, dpsCfg)
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			dpsShutDown()
		}
	}()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	_, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

	// change DPS to new address from "127.0.0.1:20130" to "127.0.0.1:20230" and restart DPS
	deferedDpsCleanUp = false
	dpsShutDown()
	dpsCfg.APIs.COAP.Addr = "127.0.0.1:20230"
	h := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	h.StartDps(service.WithRequestHandler(h))
	defer h.StopDps()

	// connect to device via DTLS with PSK
	addr, err := findSecureUDPEndpoint(ctx)
	require.NoError(t, err)
	deviceAddr := addr.GetHostname() + ":" + strconv.FormatInt(int64(addr.GetPort()), 10)

	subject, err := uuid.MustParse(events.OwnerToUUID(dpsCfg.EnrollmentGroups[0].Owner)).MarshalBinary()
	require.NoError(t, err)

	pskConn, err := dialDTLS(ctx, deviceAddr, subject, []byte(psk))
	require.NoError(t, err)
	defer func() {
		err = pskConn.Close()
		require.NoError(t, err)
	}()

	// Update DPS address in device
	endpoint := config.ACTIVE_DPS_SCHEME + "://" + dpsCfg.APIs.COAP.Addr
	err = pskConn.UpdateResource(ctx, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{Endpoint: &endpoint}, nil)
	require.NoError(t, err)

	// Verify whether the device is connected to the new DPS address and retrieve the updated configurations
	err = h.Verify(ctx)
	require.NoError(t, err)
}

// provide 2 coap-gateways in the cloud configuration with the same ID and only the second one is reachable,
// the device should connect to the second one after connecting to the first one fails; this is either done
// by cloud status observer when it detects that the cloud is not logged in after a number of checks or by
// the internal retry timeout in IoTivity
//
// the test is done by setting the cloud status observer to check the cloud status 3 times before trying the next one
// which is shorter than the retry timeout in IoTivity
func TestProvisiongConnectToSecondaryServerByObserver(t *testing.T) {
	if !test.TestDeviceObtSupportsTestProperties {
		t.Skip("TestDeviceObt does not support test properties")
	}
	// if cloud is not connected after 3 checks then the next one should be tried
	setLowCloudObserverCheckCount := func(ctx context.Context, deviceID string, devClient *deviceClient.Client) error {
		// the configuration will be reset by factoryReset when the test ends, so the original configuration will be restored
		return devClient.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{TestProperties: test.ResourcePlgdDpsTest{
			CloudStatusObserver: test.ResourcePlgdDpsTestCloudStatusObserver{
				MaxCount: 5,
			},
		}}, nil)
	}
	dpsCfg := test.MakeConfig(t)
	dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways = []string{config.ACTIVE_COAP_SCHEME + "://" + "127.0.0.1:20999"} // should be unreachable
	dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways = append(dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST)
	rh := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	testProvisioningWithDPSHandler(t, rh, time.Minute, test.WithConfigureDevice(setLowCloudObserverCheckCount))
}

// similar to TestProvisiongConnectToSecondaryServerByObserver but we setup the device that it the IoTivity retry timeout
// executes before the cloud status observer detects that the cloud is not logged in
func TestProvisiongConnectToSecondaryServerByRetryTimeout(t *testing.T) {
	if !test.TestDeviceObtSupportsTestProperties {
		t.Skip("TestDeviceObt does not support test properties")
	}
	setShortRetry := func(ctx context.Context, deviceID string, devClient *deviceClient.Client) error {
		// the configuration will be reset by factoryReset when the test ends, so the original configuration will be restored
		return devClient.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{TestProperties: test.ResourcePlgdDpsTest{
			Iotivity: test.ResourcePlgdDpsTestIotivity{
				Retry: []int64{int64(5 * time.Second / time.Millisecond)},
			},
		}}, nil)
	}
	dpsCfg := test.MakeConfig(t)
	dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways = []string{config.ACTIVE_COAP_SCHEME + "://" + "127.0.0.1:20999"} // should be unreachable
	dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways = append(dpsCfg.EnrollmentGroups[0].Hubs[0].Gateways, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST)
	rh := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	testProvisioningWithDPSHandler(t, rh, time.Minute, test.WithConfigureDevice(setShortRetry))
}

// provide 2 coap-gateways in the cloud configuration with different IDs and only the second one is reachable
// cloud observer should detect the problem and switch to the second one
func TestProvisiongConnectToSecondaryCloudByObserver(t *testing.T) {
	if !test.TestDeviceObtSupportsTestProperties {
		t.Skip("TestDeviceObt does not support test properties")
	}
	// if cloud is not connected after 3 checks then the next one should be tried
	setLowCloudObserverCheckCount := func(ctx context.Context, deviceID string, devClient *deviceClient.Client) error {
		// the configuration will be reset by factoryReset when the test ends, so the original configuration will be restored
		return devClient.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{TestProperties: test.ResourcePlgdDpsTest{
			CloudStatusObserver: test.ResourcePlgdDpsTestCloudStatusObserver{
				MaxCount: 5,
			},
		}}, nil)
	}
	hubCfg := test.MakeHubConfig(uuid.NewString(), config.ACTIVE_COAP_SCHEME+"://"+"127.0.0.1:20999") // should be unreachable
	dpsCfg := test.MakeConfig(t)
	dpsCfg.EnrollmentGroups[0].Hubs = append([]service.HubConfig{hubCfg}, dpsCfg.EnrollmentGroups[0].Hubs...)
	rhCfg := defaultTestDpsHandlerConfig()
	rhCfg.expectedCounts[test.HandlerIDCredentials] = 2
	rhCfg.expectedCounts[test.HandlerIDACLs] = 2
	rh := newTestRequestHandler(t, dpsCfg, rhCfg)
	testProvisioningWithDPSHandler(t, rh, time.Minute, test.WithConfigureDevice(setLowCloudObserverCheckCount))
}

// provide 2 coap-gateways in the cloud configuration with different IDs and only the second one is reachable
// the iotivity retry mechanism should switch to the second one
func TestProvisiongConnectToSecondaryCloudByRetryTimeout(t *testing.T) {
	if !test.TestDeviceObtSupportsTestProperties {
		t.Skip("TestDeviceObt does not support test properties")
	}
	setShortRetry := func(ctx context.Context, deviceID string, devClient *deviceClient.Client) error {
		// the configuration will be reset by factoryReset when the test ends, so the original configuration will be restored
		return devClient.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{TestProperties: test.ResourcePlgdDpsTest{
			Iotivity: test.ResourcePlgdDpsTestIotivity{
				Retry: []int64{int64(5 * time.Second / time.Millisecond)},
			},
		}}, nil)
	}
	hubCfg := test.MakeHubConfig(uuid.NewString(), "127.0.0.1:20999") // should be unreachable
	dpsCfg := test.MakeConfig(t)
	dpsCfg.EnrollmentGroups[0].Hubs = append([]service.HubConfig{hubCfg}, dpsCfg.EnrollmentGroups[0].Hubs...)
	rhCfg := defaultTestDpsHandlerConfig()
	rhCfg.expectedCounts[test.HandlerIDCredentials] = 2
	rhCfg.expectedCounts[test.HandlerIDACLs] = 2
	rh := newTestRequestHandler(t, dpsCfg, rhCfg)
	testProvisioningWithDPSHandler(t, rh, time.Minute, test.WithConfigureDevice(setShortRetry))
}
