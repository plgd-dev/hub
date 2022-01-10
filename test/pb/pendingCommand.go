package pb

import (
	"context"
	"crypto/tls"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwService "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdService "github.com/plgd-dev/hub/v2/resource-directory/test"
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

	oauthShutdown := oauthTest.SetUp(t)
	idShutdown := idService.SetUp(t)
	raShutdown := raService.SetUp(t)
	rdShutdown := rdService.SetUp(t)
	grpcShutdown := grpcgwService.SetUp(t)
	caShutdown := caService.SetUp(t)
	secureGWShutdown := coapgwTest.SetUp(t)

	var closeFunc fn.FuncList
	closeFunc.AddFunc(caShutdown)
	closeFunc.AddFunc(grpcShutdown)
	closeFunc.AddFunc(rdShutdown)
	closeFunc.AddFunc(raShutdown)
	closeFunc.AddFunc(idShutdown)
	closeFunc.AddFunc(oauthShutdown)

	shutdownHttp := httpgwTest.SetUp(t)
	closeFunc.AddFunc(shutdownHttp)

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	closeFunc.AddFunc(func() {
		_ = conn.Close
	})
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	closeFunc.AddFunc(shutdownDevSim)

	secureGWShutdown()

	create := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, err)
	}
	create()
	retrieve := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		})
		require.Error(t, err)
	}
	retrieve()
	update := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, err)
	}
	update()
	delete := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		})
		require.Error(t, err)
	}
	delete()
	updateDeviceMetadata := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:              deviceID,
			ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
		})
		require.Error(t, err)
	}
	updateDeviceMetadata()
	updateDeviceMetadata()

	client, err := c.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{})
	require.NoError(t, err)
	resourcePendings := make([]ResourcePendingEvent, 0, 24)
	devicePendings := make([]DevicePendingEvent, 0, 24)
	for {
		p, err := client.Recv()
		if err == io.EOF {
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

	return c, resourcePendings, devicePendings, closeFunc.ToFunction()
}

func CmpPendingCmds(t *testing.T, want []*pb.PendingCommand, got []*pb.PendingCommand) {
	require.Len(t, got, len(want))

	sort.Sort(SortPendingCommand(want))
	sort.Sort(SortPendingCommand(got))

	for idx := range want {
		switch {
		case got[idx].GetResourceCreatePending() != nil:
			resetCorrelationId := want[idx].GetResourceCreatePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceCreatePending(got[idx].GetResourceCreatePending(), resetCorrelationId)
		case got[idx].GetResourceRetrievePending() != nil:
			resetCorrelationId := want[idx].GetResourceRetrievePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceRetrievePending(got[idx].GetResourceRetrievePending(), resetCorrelationId)
		case got[idx].GetResourceUpdatePending() != nil:
			resetCorrelationId := want[idx].GetResourceUpdatePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceUpdatePending(got[idx].GetResourceUpdatePending(), resetCorrelationId)
		case got[idx].GetResourceDeletePending() != nil:
			resetCorrelationId := want[idx].GetResourceDeletePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpResourceDeletePending(got[idx].GetResourceDeletePending(), resetCorrelationId)
		case got[idx].GetDeviceMetadataUpdatePending() != nil:
			resetCorrelationId := want[idx].GetDeviceMetadataUpdatePending().GetAuditContext().GetCorrelationId() == ""
			CleanUpDeviceMetadataUpdatePending(got[idx].GetDeviceMetadataUpdatePending(), resetCorrelationId)
		}
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
	}
}

func CmpCancelPendingCmdResponses(t *testing.T, want *pb.CancelPendingCommandsResponse, got *pb.CancelPendingCommandsResponse) {
	sort.Strings(want.CorrelationIds)
	sort.Strings(got.CorrelationIds)
	require.Equal(t, want.CorrelationIds, got.CorrelationIds)
}

func CleanUpResourceCreatePending(e *events.ResourceCreatePending, resetCorrelationId bool) *events.ResourceCreatePending {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceCreatePending(t *testing.T, expected, got *events.ResourceCreatePending) {
	require.NotNil(t, expected)
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	e := CleanUpResourceCreatePending(expected, resetCorrelationId)
	require.NotNil(t, got)
	g := CleanUpResourceCreatePending(got, resetCorrelationId)

	expectedData := test.DecodeCbor(t, e.GetContent().GetData())
	gotData := test.DecodeCbor(t, g.GetContent().GetData())
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceCreatePending(t *testing.T, deviceID, href, correlationId string, data interface{}) *events.ResourceCreatePending {
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
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationId),
	}
}

func CleanUpResourceUpdatePending(e *events.ResourceUpdatePending, resetCorrelationId bool) *events.ResourceUpdatePending {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceUpdatePending(t *testing.T, expected, got *events.ResourceUpdatePending) {
	require.NotNil(t, expected)
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	e := CleanUpResourceUpdatePending(expected, resetCorrelationId)
	require.NotNil(t, got)
	g := CleanUpResourceUpdatePending(got, resetCorrelationId)

	expectedData := test.DecodeCbor(t, e.GetContent().GetData())
	gotData := test.DecodeCbor(t, g.GetContent().GetData())
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceUpdatePending(t *testing.T, deviceID, href, correlationId string, data interface{}) *events.ResourceUpdatePending {
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
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationId),
	}
}

func CleanUpResourceRetrievePending(e *events.ResourceRetrievePending, resetCorrelationId bool) *events.ResourceRetrievePending {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceRetrievePending(t *testing.T, expected, got *events.ResourceRetrievePending) {
	require.NotNil(t, expected)
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceRetrievePending(expected, resetCorrelationId)
	require.NotNil(t, got)
	CleanUpResourceRetrievePending(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceRetrievePending(deviceID, href, correlationId string) *events.ResourceRetrievePending {
	return &events.ResourceRetrievePending{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationId),
	}
}

func CleanUpResourceDeletePending(e *events.ResourceDeletePending, resetCorrelationId bool) *events.ResourceDeletePending {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceDeletePending(t *testing.T, expected, got *events.ResourceDeletePending) {
	require.NotNil(t, expected)
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceDeletePending(expected, resetCorrelationId)
	require.NotNil(t, got)
	CleanUpResourceDeletePending(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeResourceDeletePending(deviceID, href, correlationId string) *events.ResourceDeletePending {
	return &events.ResourceDeletePending{
		ResourceId:   &commands.ResourceId{DeviceId: deviceID, Href: href},
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationId),
	}
}

func CleanUpDeviceMetadataUpdatePending(e *events.DeviceMetadataUpdatePending, resetCorrelationId bool) *events.DeviceMetadataUpdatePending {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpDeviceMetadataUpdatePending(t *testing.T, expected, got *events.DeviceMetadataUpdatePending) {
	require.NotNil(t, expected)
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpDeviceMetadataUpdatePending(expected, resetCorrelationId)
	require.NotNil(t, got)
	CleanUpDeviceMetadataUpdatePending(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeDeviceMetadataUpdatePending(deviceID string, shadowSynchronization commands.ShadowSynchronization, correlationId string) *events.DeviceMetadataUpdatePending {
	return &events.DeviceMetadataUpdatePending{
		DeviceId: deviceID,
		UpdatePending: &events.DeviceMetadataUpdatePending_ShadowSynchronization{
			ShadowSynchronization: shadowSynchronization,
		},
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationId),
	}
}
