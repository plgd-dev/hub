package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	raTest "github.com/plgd-dev/hub/resource-aggregate/test"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func waitForEvents(t *testing.T, client pb.GrpcGateway_GetEventsClient) []interface{} {
	events := make([]interface{}, 0, 8)
	for {
		value, err := client.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		event := pbTest.GetWrappedEvent(value)
		require.NotNil(t, event)
		events = append(events, event)
	}
	return events
}

func TestRequestHandlerGetEventsStateSnapshot(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	raCfg := raTest.MakeConfig(t)
	raCfg.Clients.Eventstore.SnapshotThreshold = 5
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, resources)
	defer shutdownDevSim()
	time.Sleep(time.Millisecond * 200)

	lightHref := test.TestResourceLightInstanceHref("1")
	timestampFilter := time.Now().UnixNano()
	for i := 1; i >= 0; i-- {
		_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, lightHref),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": i,
				}),
			},
		})
		require.NoError(t, err)
	}
	require.NoError(t, err)
	time.Sleep(time.Second)

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
		DeviceIdFilter:  []string{deviceID},
		TimestampFilter: timestampFilter,
	})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	makeLightData := func(power int) map[string]interface{} {
		return map[string]interface{}{
			"name":  "Light",
			"power": uint64(power),
			"state": false,
		}
	}

	evs := waitForEvents(t, client)
	require.Len(t, evs, 3)
	for _, ev := range evs {
		switch event := ev.(type) {
		case *events.ResourceStateSnapshotTaken:
			pbTest.CmpResourceStateSnapshotTaken(t, &events.ResourceStateSnapshotTaken{
				ResourceId:           commands.NewResourceID(deviceID, lightHref),
				LatestResourceChange: pbTest.MakeResourceChanged(t, deviceID, lightHref, "", makeLightData(1)),
				ResourceUpdatePendings: []*events.ResourceUpdatePending{
					pbTest.MakeResourceUpdatePending(t, deviceID, lightHref, "", map[string]interface{}{"power": 0}),
				},
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			}, event)
		case *events.ResourceChanged:
			pbTest.CmpResourceChanged(t, pbTest.MakeResourceChanged(t, deviceID, lightHref, "", makeLightData(0)), event, "")
		case *events.ResourceUpdated:
			pbTest.CmpResourceUpdated(t, pbTest.MakeResourceUpdated(t, deviceID, lightHref, "", nil), event)
		default:
			assert.Fail(t, "unexpected event", "event: %v", ev)
		}
	}
}

func TestRequestHandlerGetEventsResourceLinksSnapshot(t *testing.T) {
	// TODO
}

func TestRequestHandlerGetEventsDeviceMetadataSnapshot(t *testing.T) {
	// TODO
}
