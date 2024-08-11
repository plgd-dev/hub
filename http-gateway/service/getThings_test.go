package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	bridgeTD "github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	bridgeResourcesTD "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	"github.com/plgd-dev/device/v2/client/core"
	bridgeDevice "github.com/plgd-dev/device/v2/cmd/bridge-device/device"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	deviceCoap "github.com/plgd-dev/device/v2/pkg/net/coap"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	schemaMaintenance "github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwService "github.com/plgd-dev/hub/v2/http-gateway/service"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	httpgwUri "github.com/plgd-dev/hub/v2/http-gateway/uri"
	isPb "github.com/plgd-dev/hub/v2/identity-store/pb"
	isTest "github.com/plgd-dev/hub/v2/identity-store/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raPb "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/device/bridge"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	oauthUri "github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/service"
	vd "github.com/plgd-dev/hub/v2/test/virtual-device"
	"github.com/stretchr/testify/require"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type virtualDevice struct {
	name      string
	deviceID  string
	tdEnabled bool
}

func createDevices(ctx context.Context, t *testing.T, numDevices int, protocol commands.Connection_Protocol) []virtualDevice {
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	isConn, err := grpc.NewClient(config.IDENTITY_STORE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = isConn.Close()
	}()
	isClient := isPb.NewIdentityStoreClient(isConn)

	raConn, err := grpc.NewClient(config.RESOURCE_AGGREGATE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raClient := raPb.NewResourceAggregateClient(raConn)

	devices := make([]virtualDevice, 0, numDevices)
	for i := 0; i < numDevices; i++ {
		tdEnabled := (i%2 == 0)
		devices = append(devices, virtualDevice{
			name:      fmt.Sprintf("dev-%v", i),
			deviceID:  uuid.NewString(),
			tdEnabled: tdEnabled,
		})
	}

	numGoRoutines := int64(8)
	sem := semaphore.NewWeighted(numGoRoutines)
	for i := range devices {
		err = sem.Acquire(ctx, 1)
		require.NoError(t, err)
		go func(dev virtualDevice) {
			vd.CreateDevice(ctx, t, dev.name, dev.deviceID, 0, dev.tdEnabled, protocol, isClient, raClient)
			sem.Release(1)
		}(devices[i])
	}
	err = sem.Acquire(ctx, numGoRoutines)
	require.NoError(t, err)
	return devices
}

func TestRequestHandlerGetThings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway | service.SetUpServicesResourceAggregate
	isConfig := isTest.MakeConfig(t)
	isConfig.APIs.GRPC.TLS.ClientCertificateRequired = false
	raConfig := raTest.MakeConfig(t)
	raConfig.APIs.GRPC.TLS.ClientCertificateRequired = false
	tearDown := service.SetUpServices(ctx, t, services, service.WithISConfig(isConfig), service.WithRAConfig(raConfig))
	defer tearDown()

	httpgwCfg := httpgwTest.MakeConfig(t, true)
	shutdownHttp := httpgwTest.New(t, httpgwCfg)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	numDevices := 10
	vds := createDevices(ctx, t, numDevices, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME))

	rb := httpgwTest.NewRequest(http.MethodGet, httpgwUri.Things, nil).AuthToken(token)
	resp := httpgwTest.HTTPDo(t, rb.Build())
	defer func() {
		_ = resp.Body.Close()
	}()

	var v httpgwService.GetThingsResponse
	err := httpTest.UnmarshalJson(resp.StatusCode, resp.Body, &v)
	require.NoError(t, err)
	require.Equal(t, httpgwCfg.UI.WebConfiguration.HTTPGatewayAddress+httpgwUri.Things, v.Base)
	vdsWithTD := []virtualDevice{}
	for _, vd := range vds {
		if vd.tdEnabled {
			vdsWithTD = append(vdsWithTD, vd)
		}
	}
	require.Len(t, v.Links, len(vdsWithTD))
	for _, dev := range vdsWithTD {
		require.Contains(t, v.Links, httpgwService.ThingLink{
			Href: "/" + dev.deviceID,
			Rel:  httpgwService.ThingLinkRelationItem,
		})
	}
}

func TestBridgeDeviceGetThings(t *testing.T) {
	bridgeDeviceCfg, err := test.GetBridgeDeviceConfig()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	httpgwCfg := httpgwTest.MakeConfig(t, true)
	shutdownHttp := httpgwTest.New(t, httpgwCfg)
	defer shutdownHttp()

	var devIDs []string
	for i := 0; i < bridgeDeviceCfg.NumGeneratedBridgedDevices; i++ {
		bdName := test.TestBridgeDeviceInstanceName(strconv.Itoa(i))
		bdID := test.MustFindDeviceByName(bdName, func(d *core.Device) deviceCoap.OptionFunc {
			return deviceCoap.WithQuery("di=" + d.DeviceID())
		})
		devIDs = append(devIDs, bdID)
		bd := bridge.NewDevice(bdID, bdName, bridgeDeviceCfg.NumResourcesPerDevice, true)
		shutdownBd := test.OnboardDevice(ctx, t, c, bd, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, bd.GetDefaultResources())
		defer shutdownBd()
	}

	rb := httpgwTest.NewRequest(http.MethodGet, httpgwUri.Things, nil).AuthToken(token)
	resp := httpgwTest.HTTPDo(t, rb.Build())
	defer func() {
		_ = resp.Body.Close()
	}()

	var v httpgwService.GetThingsResponse
	err = httpTest.UnmarshalJson(resp.StatusCode, resp.Body, &v)
	require.NoError(t, err)
	require.Equal(t, httpgwCfg.UI.WebConfiguration.HTTPGatewayAddress+httpgwUri.Things, v.Base)
	require.Len(t, v.Links, bridgeDeviceCfg.NumGeneratedBridgedDevices)
	for _, devID := range devIDs {
		require.Contains(t, v.Links, httpgwService.ThingLink{
			Href: "/" + devID,
			Rel:  httpgwService.ThingLinkRelationItem,
		})
	}
}

func patchSecurity(td *wotTD.ThingDescription) {
	vCfg := config.MakeValidatorConfig()
	openCfgs := make([]openid.Config, 0, len(vCfg.Endpoints))
	for _, ep := range vCfg.Endpoints {
		host := ep.Authority
		if strings.Contains(host, config.OAUTH_SERVER_HOST) {
			openCfgs = append(openCfgs, openid.Config{
				TokenURL: ep.Authority + oauthUri.Token,
				AuthURL:  ep.Authority + oauthUri.Authorize,
			})
			continue
		}
		if strings.Contains(host, config.M2M_OAUTH_SERVER_HTTP_HOST) {
			openCfgs = append(openCfgs, openid.Config{
				TokenURL: ep.Authority + oauthUri.Token,
			})
			continue
		}
	}
	httpgwService.ThingSetSecurity(td, openCfgs)
}

func getPatchedTD(t *testing.T, deviceCfg bridgeDevice.Config, deviceID string, links []wotTD.IconLinkElement, validateDevices map[string]struct{}, title, host string) *wotTD.ThingDescription {
	td, err := bridgeDevice.GetThingDescription(deviceCfg.ThingDescription.File, deviceCfg.NumResourcesPerDevice)
	require.NoError(t, err)

	baseURL := host + httpgwUri.API
	base, err := url.Parse(baseURL)
	require.NoError(t, err)
	td.Base = *base
	td.Title = title
	id, err := bridgeTD.GetThingDescriptionID(deviceID)
	require.NoError(t, err)
	td.ID = id

	deviceUUID, err := uuid.Parse(deviceID)
	require.NoError(t, err)
	propertyBaseURL := "/" + httpgwUri.DevicesPathKey + "/" + deviceID + "/" + httpgwUri.ResourcesPathKey
	dev, ok := bridgeResourcesTD.GetOCFResourcePropertyElement(schemaDevice.ResourceURI)
	require.True(t, ok)
	dev, err = bridgeResourcesTD.PatchDeviceResourcePropertyElement(dev, deviceUUID, propertyBaseURL, message.AppJSON, bridgeDevice.DeviceResourceType, httpgwService.CreateHTTPForms)
	require.NoError(t, err)
	schemaMap := bridgeDevice.GetDataSchemaForAdditionalProperties()
	for name, schema := range schemaMap {
		dev.Properties.DataSchemaMap[name] = schema
	}
	td.Properties[schemaDevice.ResourceURI] = dev

	patchSecurity(&td)

	mnt, ok := bridgeResourcesTD.GetOCFResourcePropertyElement(schemaMaintenance.ResourceURI)
	require.True(t, ok)
	mnt, err = bridgeResourcesTD.PatchMaintenanceResourcePropertyElement(mnt, deviceUUID, propertyBaseURL, message.AppJSON, httpgwService.CreateHTTPForms)
	require.NoError(t, err)
	td.Properties[schemaMaintenance.ResourceURI] = mnt

	for i := 0; i < deviceCfg.NumResourcesPerDevice; i++ {
		href := bridgeDevice.GetTestResourceHref(i)
		prop := bridgeDevice.GetPropertyDescriptionForTestResource()
		prop, err := bridgeDevice.PatchTestResourcePropertyElement(prop, deviceUUID, propertyBaseURL+href, message.AppJSON, httpgwService.CreateHTTPForms)
		require.NoError(t, err)
		td.Properties[href] = prop
	}

	expectedLinks := make([]wotTD.IconLinkElement, 0, len(links))
	for _, link := range links {
		patchedLink, ok := httpgwService.ThingPatchLink(link, validateDevices)
		if !ok {
			continue
		}
		expectedLinks = append(expectedLinks, patchedLink)
	}
	td.Links = expectedLinks

	return &td
}

func bridgeDeviceInstanceName(idx int) string {
	return test.TestBridgeDeviceInstanceName(strconv.Itoa(idx))
}

func onboardBridgeDevice(ctx context.Context, t *testing.T, idx int, c pb.GrpcGatewayClient, bridgeDeviceCfg bridgeDevice.Config) (string, func()) {
	bdName := bridgeDeviceInstanceName(idx)
	bdID := test.MustFindDeviceByName(bdName, func(d *core.Device) deviceCoap.OptionFunc {
		return deviceCoap.WithQuery("di=" + d.DeviceID())
	})
	bd := bridge.NewDevice(bdID, bdName, bridgeDeviceCfg.NumResourcesPerDevice, true)
	shutdownBd := test.OnboardDevice(ctx, t, c, bd, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, bd.GetDefaultResources())
	return bdID, shutdownBd
}

func TestBridgeDeviceGetThing(t *testing.T) {
	bridgeDeviceCfg, err := test.GetBridgeDeviceConfig()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	raCfg := raTest.MakeConfig(t)
	raCfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg))
	defer tearDown()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	httpgwCfg := httpgwTest.MakeConfig(t, true)
	shutdownHttp := httpgwTest.New(t, httpgwCfg)
	defer shutdownHttp()

	deviceIDs := make([]string, 0, 2)
	validLinkedDevices := make(map[string]struct{}, 2)
	for i := 0; i < 2; i++ {
		bdID, shutdownBd := onboardBridgeDevice(ctx, t, i, c, bridgeDeviceCfg)
		defer shutdownBd()
		deviceIDs = append(deviceIDs, bdID)
		validLinkedDevices[bdID] = struct{}{}
	}

	// update TD links in resource twin
	wotRes, err := c.GetResources(ctx, &pb.GetResourcesRequest{
		ResourceIdFilter: []*pb.ResourceIdFilter{
			{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceIDs[0],
					Href:     bridgeResourcesTD.ResourceURI,
				},
			},
		},
	})
	require.NoError(t, err)
	resources := make([]*pb.Resource, 0, 1)
	for {
		res, errR := wotRes.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		resources = append(resources, res)
	}
	require.Len(t, resources, 1)

	var updateLinksTD wotTD.ThingDescription
	err = json.Decode(resources[0].GetData().GetContent().GetData(), &updateLinksTD)
	require.NoError(t, err)
	links := []wotTD.IconLinkElement{
		{
			Rel:  bridgeTD.StringToPtr("icon"),
			Href: "https://example.com/icon.png",
		},
		{
			Rel:  bridgeTD.StringToPtr(httpgwService.ThingLinkRelationItem),
			Href: "/" + deviceIDs[1],
		},
		{
			Rel:  bridgeTD.StringToPtr(httpgwService.ThingLinkRelationCollection),
			Href: "/" + deviceIDs[1],
		},
		{
			Rel:  bridgeTD.StringToPtr(httpgwService.ThingLinkRelationItem),
			Href: "/" + uuid.NewString(),
		},
	}
	updateLinksTD.Links = links
	data, err := json.Encode(updateLinksTD)
	require.NoError(t, err)

	sub, err := c.SubscribeToEvents(ctx)
	require.NoError(t, err)

	err = sub.Send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
			EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED},
			ResourceIdFilter: []*pb.ResourceIdFilter{
				{
					ResourceId: resources[0].GetData().GetResourceId(),
				},
			},
		}},
	})
	require.NoError(t, err)
	s, err := sub.Recv()
	require.NoError(t, err)
	require.Equal(t, pb.Event_OperationProcessed_ErrorStatus_OK, s.GetOperationProcessed().GetErrorStatus().GetCode())

	// overwrite TD links in resource twin
	raConn, err := grpc.NewClient(config.RESOURCE_AGGREGATE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raC := raPb.NewResourceAggregateClient(raConn)
	_, err = raC.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: resources[0].GetData().GetResourceId(),
		Status:     commands.Status_OK,
		Content: &commands.Content{
			Data:              data,
			CoapContentFormat: int32(message.AppJSON),
			ContentType:       message.AppJSON.String(),
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "test",
			Sequence:     1,
		},
	})
	require.NoError(t, err)

	ev, err := sub.Recv()
	require.NoError(t, err)
	require.Equal(t, commands.Status_OK, ev.GetResourceChanged().GetStatus())

	err = sub.CloseSend()
	require.NoError(t, err)

	type args struct {
		accept   string
		deviceID string
	}
	tests := []struct {
		name     string
		args     args
		want     *wotTD.ThingDescription
		wantCode int
	}{
		{
			name: "json: get from resource twin",
			args: args{
				deviceID: deviceIDs[0],
			},
			want:     getPatchedTD(t, bridgeDeviceCfg, deviceIDs[0], links, validLinkedDevices, bridgeDeviceInstanceName(0), httpgwCfg.UI.WebConfiguration.HTTPGatewayAddress),
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, httpgwUri.AliasDeviceThing, nil).AuthToken(token).Accept(tt.args.accept).DeviceId(tt.args.deviceID)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantCode, resp.StatusCode)
			values := make([]*wotTD.ThingDescription, 0, 1)
			for {
				var td wotTD.ThingDescription
				err := json.ReadFrom(resp.Body, &td)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				values = append(values, &td)
			}
			if tt.wantCode != http.StatusOK {
				require.Empty(t, values)
				return
			}
			require.Len(t, values, 1)
			test.CmpThingDescription(t, tt.want, values[0])
		})
	}
}
