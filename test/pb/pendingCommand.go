package pb

import (
	"context"
	"crypto/tls"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	caService "github.com/plgd-dev/hub/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/hub/coap-gateway/test"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	grpcgwService "github.com/plgd-dev/hub/grpc-gateway/test"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	idService "github.com/plgd-dev/hub/identity-store/test"
	"github.com/plgd-dev/hub/pkg/fn"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/resource-aggregate/test"
	rdService "github.com/plgd-dev/hub/resource-directory/test"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/service"
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

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
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
			got[idx].GetResourceCreatePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceCreatePending().EventMetadata = nil
		case got[idx].GetResourceRetrievePending() != nil:
			got[idx].GetResourceRetrievePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceRetrievePending().EventMetadata = nil
		case got[idx].GetResourceUpdatePending() != nil:
			got[idx].GetResourceUpdatePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceUpdatePending().EventMetadata = nil
		case got[idx].GetResourceDeletePending() != nil:
			got[idx].GetResourceDeletePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceDeletePending().EventMetadata = nil
		case got[idx].GetDeviceMetadataUpdatePending() != nil:
			got[idx].GetDeviceMetadataUpdatePending().AuditContext.CorrelationId = ""
			got[idx].GetDeviceMetadataUpdatePending().EventMetadata = nil
		}
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
	}
}

func CmpCancelPendingCmdResponses(t *testing.T, want *pb.CancelPendingCommandsResponse, got *pb.CancelPendingCommandsResponse) {
	sort.Strings(want.CorrelationIds)
	sort.Strings(got.CorrelationIds)
	require.Equal(t, want.CorrelationIds, got.CorrelationIds)
}

func MakeResourceCreatePending(t *testing.T, deviceID, href string, data interface{}) *events.ResourceCreatePending {
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
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
	}
}

func MakeResourceUpdatePending(t *testing.T, deviceID, href string, data interface{}) *events.ResourceUpdatePending {
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
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
	}
}
