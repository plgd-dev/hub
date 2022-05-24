package virtualdevice

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/go-coap/v2/message"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raPb "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func CreateDevice(ctx context.Context, t *testing.T, name string, deviceID string, numResources int, isClient pb.IdentityStoreClient, raClient raPb.ResourceAggregateClient, grpcClient grpcPb.GrpcGatewayClient) {
	client, err := grpcClient.SubscribeToEvents(ctx)
	require.NoError(t, err)

	err = client.Send(&grpcPb.SubscribeToEvents{
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_REGISTERED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err := client.Recv()
	require.NoError(t, err)
	require.NotEmpty(t, ev.GetOperationProcessed())
	require.Equal(t, ev.GetOperationProcessed().GetErrorStatus().GetCode(), grpcPb.Event_OperationProcessed_ErrorStatus_OK)

	_, err = isClient.AddDevice(ctx, &pb.AddDeviceRequest{
		DeviceId: deviceID,
	})
	require.NoError(t, err)

	for {
		ev, err = client.Recv()
		require.NoError(t, err)
		require.NotEmpty(t, ev.GetDeviceRegistered().GetDeviceIds())
		if ev.GetDeviceRegistered().GetDeviceIds()[0] == deviceID {
			break
		}
	}

	err = client.CloseSend()
	require.NoError(t, err)

	_, err = raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId:      deviceID,
		CorrelationId: uuid.NewString(),
		Update: &commands.UpdateDeviceMetadataRequest_Status{
			Status: &commands.ConnectionStatus{
				Value:        commands.ConnectionStatus_ONLINE,
				ValidUntil:   time.Now().Add(time.Hour).UnixNano(),
				ConnectionId: "conn-id",
			},
		},
		TimeToLive: time.Now().Add(time.Hour).UnixNano(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "conn-id",
			Sequence:     0,
		},
	})
	require.NoError(t, err)

	_, err = raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId:      deviceID,
		CorrelationId: uuid.NewString(),
		Update: &commands.UpdateDeviceMetadataRequest_Status{
			Status: &commands.ConnectionStatus{
				Value:        commands.ConnectionStatus_OFFLINE,
				ConnectionId: "conn-id",
			},
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "conn-id",
			Sequence:     1,
		},
	})
	require.NoError(t, err)

	resources := make([]*commands.Resource, 0, numResources)
	for i := 0; i < numResources; i++ {
		resources = append(resources, &commands.Resource{
			Href:          fmt.Sprintf("/res-%v", i),
			DeviceId:      deviceID,
			ResourceTypes: []string{fmt.Sprintf("res-type-%v", i)},
			Interfaces:    []string{interfaces.OC_IF_BASELINE},
		})
	}
	resources = append(resources, &commands.Resource{
		Href:          "/oic/d",
		DeviceId:      deviceID,
		ResourceTypes: []string{device.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_BASELINE},
	})
	resources = append(resources, &commands.Resource{
		Href:          "/oic/p",
		DeviceId:      deviceID,
		ResourceTypes: []string{platform.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_BASELINE},
	})
	pub := commands.PublishResourceLinksRequest{
		DeviceId:  deviceID,
		Resources: resources,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "conn-id",
			Sequence:     0,
		},
	}
	_, err = raClient.PublishResourceLinks(ctx, &pub)
	require.NoError(t, err)

	for i := 0; i < numResources; i++ {
		_, err = raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
			ResourceId: commands.NewResourceID(deviceID, fmt.Sprintf("/res-%v", i)),
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: "conn-id",
				Sequence:     0,
			},
			Content: &commands.Content{
				Data:        []byte(fmt.Sprintf("content res-%v", i)),
				ContentType: message.TextPlain.String(),
			},
			Status: commands.Status_OK,
		})
		require.NoError(t, err)
	}
	_, err = raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "conn-id",
			Sequence:     0,
		},
		Content: &commands.Content{
			Data:        test.EncodeToCbor(t, &device.Device{Name: name, ID: deviceID, ResourceTypes: []string{device.ResourceType}, Interfaces: []string{interfaces.OC_IF_BASELINE}}),
			ContentType: message.AppOcfCbor.String(),
		},
		Status: commands.Status_OK,
	})
	require.NoError(t, err)

	_, err = raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(deviceID, "/oic/p"),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "conn-id",
			Sequence:     0,
		},
		Content: &commands.Content{
			Data:        test.EncodeToCbor(t, &platform.Platform{ResourceTypes: []string{device.ResourceType}, Interfaces: []string{interfaces.OC_IF_BASELINE}, SerialNumber: fmt.Sprintf("sn %v", deviceID)}),
			ContentType: message.AppOcfCbor.String(),
		},
		Status: commands.Status_OK,
	})
	require.NoError(t, err)
}

func CreateDevices(ctx context.Context, t *testing.T, numDevices int, numResourcesPerDevice int) {
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	isConn, err := grpc.Dial(config.IDENTITY_STORE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = isConn.Close()
	}()
	isClient := pb.NewIdentityStoreClient(isConn)

	raConn, err := grpc.Dial(config.RESOURCE_AGGREGATE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raClient := raPb.NewResourceAggregateClient(raConn)

	grpcConn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = grpcConn.Close()
	}()
	grpcClient := grpcPb.NewGrpcGatewayClient(grpcConn)

	numGoRoutines := int64(8)
	sem := semaphore.NewWeighted(numGoRoutines)
	for i := 0; i < numDevices; i++ {
		err := sem.Acquire(ctx, 1)
		require.NoError(t, err)
		go func(i int) {
			CreateDevice(ctx, t, fmt.Sprintf("dev-%v", i), uuid.NewString(), numResourcesPerDevice, isClient, raClient, grpcClient)
			sem.Release(1)
		}(i)
	}
	err = sem.Acquire(ctx, numGoRoutines)
	require.NoError(t, err)
}
