package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func waitForEvents(t *testing.T, client pb.GrpcGateway_GetEventsClient) []interface{} {
	events := make([]interface{}, 0, 8)
	for {
		value, err := client.Recv()
		if errors.Is(err, io.EOF) {
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
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg))
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
		time.Sleep(time.Millisecond * 500)
	}
	require.NoError(t, err)
	time.Sleep(time.Second * 4)

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
	require.Len(t, evs, 1)
	for _, ev := range evs {
		switch event := ev.(type) {
		case *events.ResourceStateSnapshotTaken:
			pbTest.CmpResourceStateSnapshotTaken(t, &events.ResourceStateSnapshotTaken{
				ResourceId:           commands.NewResourceID(deviceID, lightHref),
				LatestResourceChange: pbTest.MakeResourceChanged(t, deviceID, lightHref, "", makeLightData(0)),
				AuditContext:         commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
			}, event)
		default:
			assert.Fail(t, "unexpected event", "event: %v", ev)
		}
	}
}

func TestRequestHandlerGetEventsResourceLinksSnapshot(*testing.T) {
	// TODO
}

func TestRequestHandlerGetEventsDeviceMetadataSnapshot(*testing.T) {
	// TODO
}
