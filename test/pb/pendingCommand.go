package pb

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type SortPendingCommand []*pb.PendingCommand

func (a SortPendingCommand) Len() int      { return len(a) }
func (a SortPendingCommand) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortPendingCommand) Less(i, j int) bool {
	toKey := func(v *pb.PendingCommand) string {
		switch {
		case v.GetResourceCreatePending() != nil:
			return v.GetResourceCreatePending().GetResourceId().GetDeviceId() + v.GetResourceCreatePending().GetResourceId().GetHref()
		case v.GetResourceRetrievePending() != nil:
			return v.GetResourceRetrievePending().GetResourceId().GetDeviceId() + v.GetResourceRetrievePending().GetResourceId().GetHref()
		case v.GetResourceUpdatePending() != nil:
			return v.GetResourceUpdatePending().GetResourceId().GetDeviceId() + v.GetResourceUpdatePending().GetResourceId().GetHref()
		case v.GetResourceDeletePending() != nil:
			return v.GetResourceDeletePending().GetResourceId().GetDeviceId() + v.GetResourceDeletePending().GetResourceId().GetHref()
		case v.GetDeviceMetadataUpdatePending() != nil:
			return v.GetDeviceMetadataUpdatePending().GetDeviceId()
		}
		return ""
	}

	return toKey(a[i]) < toKey(a[j])
}

type ResourcePendingEvent struct {
	ResourceId    *commands.ResourceId
	CorrelationID string
}

type DevicePendingEvent struct {
	DeviceID      string
	CorrelationID string
}

func InitPendingEvents(ctx context.Context, t *testing.T) (pb.GrpcGatewayClient, []ResourcePendingEvent, []DevicePendingEvent, func()) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	service.ClearDB(ctx, t)

	var closeFunc fn.FuncList
	deferedClose := true
	defer func() {
		if deferedClose {
			closeFunc.Execute()
		}
	}()

	tearDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth|service.SetUpServicesId|service.SetUpServicesResourceAggregate|
		service.SetUpServicesResourceDirectory|service.SetUpServicesCertificateAuthority|service.SetUpServicesGrpcGateway)
	closeFunc.AddFunc(tearDown)

	deferedsecureGWShutdown := true
	secureGWShutdown := coapgwTest.SetUp(t)
	closeFunc.AddFunc(func() {
		if deferedsecureGWShutdown {
			secureGWShutdown()
		}
	})

	shutdownHttp := httpgwTest.SetUp(t)
	closeFunc.AddFunc(shutdownHttp)

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	closeFunc.AddFunc(func() {
		_ = conn.Close
	})
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	closeFunc.AddFunc(shutdownDevSim)

	deferedsecureGWShutdown = false
	secureGWShutdown()

	createFn := func() {
		_, errC := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
			Async: true,
		})
		require.NoError(t, errC)
	}
	createFn()
	retrieveFn := func() {
		retrieveCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errG := c.GetResourceFromDevice(retrieveCtx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		})
		require.Error(t, errG)
	}
	retrieveFn()
	updateFn := func() {
		_, errU := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
			Async: true,
		})
		require.NoError(t, errU)
	}
	updateFn()
	deleteFn := func() {
		_, errD := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Async:      true,
		})
		require.NoError(t, errD)
	}
	deleteFn()
	updateDeviceMetadataFn := func() {
		updateCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errU := c.UpdateDeviceMetadata(updateCtx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:    deviceID,
			TwinEnabled: false,
		})
		require.Error(t, errU)
	}
	updateDeviceMetadataFn()
	updateDeviceMetadataFn()

	client, err := c.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{})
	require.NoError(t, err)
	resourcePendings := make([]ResourcePendingEvent, 0, 24)
	devicePendings := make([]DevicePendingEvent, 0, 24)
	for {
		p, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		switch {
		case p.GetDeviceMetadataUpdatePending() != nil:
			v := p.GetDeviceMetadataUpdatePending()
			devicePendings = append(devicePendings, DevicePendingEvent{
				DeviceID:      v.GetDeviceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceCreatePending() != nil:
			v := p.GetResourceCreatePending()
			resourcePendings = append(resourcePendings, ResourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceRetrievePending() != nil:
			v := p.GetResourceRetrievePending()
			resourcePendings = append(resourcePendings, ResourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceUpdatePending() != nil:
			v := p.GetResourceUpdatePending()
			resourcePendings = append(resourcePendings, ResourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceDeletePending() != nil:
			v := p.GetResourceDeletePending()
			resourcePendings = append(resourcePendings, ResourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		}
	}

	deferedClose = false
	return c, resourcePendings, devicePendings, closeFunc.ToFunction()
}

func CmpPendingCmds(t *testing.T, want []*pb.PendingCommand, got []*pb.PendingCommand) {
	require.Len(t, got, len(want))

	sort.Sort(SortPendingCommand(want))
	sort.Sort(SortPendingCommand(got))

	for idx := range want {
		switch {
		case got[idx].GetResourceCreatePending() != nil:
			resetCorrelationID := want[idx].GetResourceCreatePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceCreatePending(got[idx].GetResourceCreatePending(), resetCorrelationID)
		case got[idx].GetResourceRetrievePending() != nil:
			resetCorrelationID := want[idx].GetResourceRetrievePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceRetrievePending(got[idx].GetResourceRetrievePending(), resetCorrelationID)
		case got[idx].GetResourceUpdatePending() != nil:
			resetCorrelationID := want[idx].GetResourceUpdatePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceUpdatePending(got[idx].GetResourceUpdatePending(), resetCorrelationID)
		case got[idx].GetResourceDeletePending() != nil:
			resetCorrelationID := want[idx].GetResourceDeletePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceDeletePending(got[idx].GetResourceDeletePending(), resetCorrelationID)
		case got[idx].GetDeviceMetadataUpdatePending() != nil:
			resetCorrelationID := want[idx].GetDeviceMetadataUpdatePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpDeviceMetadataUpdatePending(got[idx].GetDeviceMetadataUpdatePending(), resetCorrelationID)
		}
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
	}
}

func CmpCancelPendingCmdResponses(t *testing.T, want *pb.CancelPendingCommandsResponse, got *pb.CancelPendingCommandsResponse) {
	sort.Strings(want.GetCorrelationIds())
	sort.Strings(got.GetCorrelationIds())
	require.Equal(t, want.GetCorrelationIds(), got.GetCorrelationIds())
}

func CleanUpResourceCreatePending(e *events.ResourceCreatePending, resetCorrelationID bool) *events.ResourceCreatePending {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	return e
}

func CmpResourceCreatePending(t *testing.T, expected, got *events.ResourceCreatePending) {
	require.NotNil(t, expected)
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	e := CleanUpResourceCreatePending(expected, resetCorrelationID)
	require.NotNil(t, got)
	g := CleanUpResourceCreatePending(got, resetCorrelationID)

	expectedData := test.DecodeCbor(t, e.GetContent().GetData())
	gotData := test.DecodeCbor(t, g.GetContent().GetData())
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceCreatePending(t *testing.T, deviceID, href string, resourceTypes []string, correlationID string, data interface{}) *events.ResourceCreatePending {
	return &events.ResourceCreatePending{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &commands.Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: -1,
			Data:              test.EncodeToCbor(t, data),
		},
		AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
		ResourceTypes: resourceTypes,
	}
}

func CleanUpResourceUpdatePending(e *events.ResourceUpdatePending, resetCorrelationID bool) *events.ResourceUpdatePending {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	return e
}

func CmpResourceUpdatePending(t *testing.T, expected, got *events.ResourceUpdatePending) {
	require.NotNil(t, expected)
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	e := CleanUpResourceUpdatePending(expected, resetCorrelationID)
	require.NotNil(t, got)
	g := CleanUpResourceUpdatePending(got, resetCorrelationID)

	expectedData := test.DecodeCbor(t, e.GetContent().GetData())
	gotData := test.DecodeCbor(t, g.GetContent().GetData())
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceUpdatePending(t *testing.T, deviceID, href string, resourceTypes []string, correlationID string, data interface{}) *events.ResourceUpdatePending {
	return &events.ResourceUpdatePending{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &commands.Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: -1,
			Data:              test.EncodeToCbor(t, data),
		},
		AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
		ResourceTypes: resourceTypes,
	}
}

func CleanUpResourceRetrievePending(e *events.ResourceRetrievePending, resetCorrelationID bool) *events.ResourceRetrievePending {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	return e
}

func CmpResourceRetrievePending(t *testing.T, expected, got *events.ResourceRetrievePending) {
	require.NotNil(t, expected)
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceRetrievePending(expected, resetCorrelationID)
	require.NotNil(t, got)
	CleanUpResourceRetrievePending(got, resetCorrelationID)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceRetrievePending(deviceID, href string, resourceTypes []string, correlationID string) *events.ResourceRetrievePending {
	return &events.ResourceRetrievePending{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
		ResourceTypes: resourceTypes,
	}
}

func CleanUpResourceDeletePending(e *events.ResourceDeletePending, resetCorrelationID bool) *events.ResourceDeletePending {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	return e
}

func CmpResourceDeletePending(t *testing.T, expected, got *events.ResourceDeletePending) {
	require.NotNil(t, expected)
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceDeletePending(expected, resetCorrelationID)
	require.NotNil(t, got)
	CleanUpResourceDeletePending(got, resetCorrelationID)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceDeletePending(deviceID, href string, resourceTypes []string, correlationID string) *events.ResourceDeletePending {
	return &events.ResourceDeletePending{
		ResourceId:    &commands.ResourceId{DeviceId: deviceID, Href: href},
		AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
		ResourceTypes: resourceTypes,
	}
}

func CleanUpDeviceMetadataUpdatePending(e *events.DeviceMetadataUpdatePending, resetCorrelationID bool) *events.DeviceMetadataUpdatePending {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	return e
}

func CmpDeviceMetadataUpdatePending(t *testing.T, expected, got *events.DeviceMetadataUpdatePending) {
	require.NotNil(t, expected)
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpDeviceMetadataUpdatePending(expected, resetCorrelationID)
	require.NotNil(t, got)
	CleanUpDeviceMetadataUpdatePending(got, resetCorrelationID)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeDeviceMetadataUpdatePending(deviceID string, twinEnabled bool, correlationID string) *events.DeviceMetadataUpdatePending {
	return &events.DeviceMetadataUpdatePending{
		DeviceId: deviceID,
		UpdatePending: &events.DeviceMetadataUpdatePending_TwinEnabled{
			TwinEnabled: twinEnabled,
		},
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
	}
}
