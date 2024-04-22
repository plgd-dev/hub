package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getAllOnboardEvents(t *testing.T, deviceID string, links []schema.ResourceLink) []*pb.GetEventsResponse {
	publishedResources := events.ResourceLinksSnapshotTaken{
		DeviceId:     deviceID,
		Resources:    make(map[string]*commands.Resource),
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
	}
	for _, r := range test.ResourceLinksToResources(deviceID, links) {
		publishedResources.Resources[r.GetHref()] = r
	}
	deviceMetadata := events.DeviceMetadataSnapshotTaken{
		DeviceId:              deviceID,
		DeviceMetadataUpdated: pbTest.MakeDeviceMetadataUpdated(deviceID, commands.Connection_ONLINE, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_IN_SYNC, ""),
	}
	ret := make([]*pb.GetEventsResponse, 0, 8)
	ret = append(ret, &pb.GetEventsResponse{
		Type: &pb.GetEventsResponse_DeviceMetadataSnapshotTaken{
			DeviceMetadataSnapshotTaken: &deviceMetadata,
		},
	}, &pb.GetEventsResponse{
		Type: &pb.GetEventsResponse_ResourceLinksSnapshotTaken{
			ResourceLinksSnapshotTaken: &publishedResources,
		},
	})
	for _, r := range test.GetAllBackendResourceRepresentations(t, deviceID, test.TestDeviceName) {
		rid := commands.ResourceIdFromString(r.Href) // validate
		ret = append(ret, &pb.GetEventsResponse{
			Type: &pb.GetEventsResponse_ResourceStateSnapshotTaken{
				ResourceStateSnapshotTaken: &events.ResourceStateSnapshotTaken{
					ResourceId:           rid,
					LatestResourceChange: pbTest.MakeResourceChanged(t, deviceID, rid.GetHref(), r.ResourceTypes, "", r.Representation),
					AuditContext:         commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
					ResourceTypes:        r.ResourceTypes,
				},
			},
		})
	}
	return ret
}

func waitAndCheckEvents(t *testing.T, client pb.GrpcGateway_GetEventsClient, expected []*pb.GetEventsResponse) {
	events := make([]*pb.GetEventsResponse, 0, 8)
	for {
		value, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		events = append(events, value)
	}
	pbTest.CmpUpGetEventsResponses(t, expected, events)
}

func TestRequestHandlerGetEventsOnOnboard(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resources)
	defer shutdownDevSim()

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	waitAndCheckEvents(t, client, getAllOnboardEvents(t, deviceID, resources))
}
