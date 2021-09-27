package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/sdk/schema"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func containsDevMetadataUpdated(t *testing.T, values []*pb.GetEventsResponse, deviceID string, status commands.ConnectionStatus_Status) {
	for _, v := range values {
		if evt := v.GetDeviceMetadataUpdated(); evt != nil {
			if evt.DeviceId != deviceID || evt.Status.Value != status {
				require.Fail(t, "invalid DeviceMetadataUpdated")
			}
			return
		}
	}
	require.Fail(t, "DeviceMetadataUpdated not found")
}

func containsResourceLinksPublished(t *testing.T, values []*pb.GetEventsResponse, deviceID string, links []schema.ResourceLink) {
	for _, v := range values {
		if evt := v.GetResourceLinksPublished(); evt != nil {
			if evt.DeviceId != deviceID || len(links) != len(evt.GetResources()) {
				require.Fail(t, "invalid ResourceLinksPublished")
			}
			return
		}
	}
	require.Fail(t, "ResourceLinksPublished not found")
}

func TestRequestHandler_GetEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, resources)
	defer shutdownDevSim()

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{})
	require.NoError(t, err)
	values := make([]*pb.GetEventsResponse, 0, 8)
	for {
		value, err := client.Recv()
		if err == io.EOF {
			break
		}
		values = append(values, value)
	}
	containsDevMetadataUpdated(t, values, deviceID, commands.ConnectionStatus_ONLINE)
	containsResourceLinksPublished(t, values, deviceID, resources)

	for _, res := range resources {
		client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
			ResourceIdFilter: []string{deviceID + res.Href},
		})
		require.NoError(t, err)
		value, err := client.Recv()
		require.NoError(t, err)

		evt := value.GetResourceChanged()
		require.NotNil(t, evt)
		require.Equal(t, evt.GetResourceId().GetHref(), res.Href)
	}
}
