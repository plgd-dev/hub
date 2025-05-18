package virtualdevice

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raPb "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func CreateDeviceResourceLinks(deviceID string, numResources int, tdEnabled bool) []*commands.Resource {
	resources := make([]*commands.Resource, 0, numResources)
	for i := range numResources {
		resources = append(resources, &commands.Resource{
			Href:          fmt.Sprintf("/res-%v", i),
			DeviceId:      deviceID,
			ResourceTypes: []string{fmt.Sprintf("res-type-%v", i)},
			Interfaces:    []string{interfaces.OC_IF_BASELINE},
			Policy: &commands.Policy{
				BitFlags: commands.ToPolicyBitFlags(schema.Observable | schema.Discoverable),
			},
		})
	}
	resources = append(resources, &commands.Resource{
		Href:          device.ResourceURI,
		DeviceId:      deviceID,
		ResourceTypes: []string{device.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_BASELINE},
		Policy: &commands.Policy{
			BitFlags: commands.ToPolicyBitFlags(schema.Observable | schema.Discoverable),
		},
	})
	resources = append(resources, &commands.Resource{
		Href:          platform.ResourceURI,
		DeviceId:      deviceID,
		ResourceTypes: []string{platform.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_BASELINE},
		Policy: &commands.Policy{
			BitFlags: commands.ToPolicyBitFlags(schema.Observable | schema.Discoverable),
		},
	})

	if tdEnabled {
		resources = append(resources, &commands.Resource{
			Href:          thingDescription.ResourceURI,
			DeviceId:      deviceID,
			ResourceTypes: []string{thingDescription.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R},
			Policy: &commands.Policy{
				BitFlags: commands.ToPolicyBitFlags(schema.Discoverable),
			},
		})
	}

	return resources
}

func CreateDevice(ctx context.Context, t *testing.T, name string, deviceID string, numResources int, tdEnabled bool, protocol commands.Connection_Protocol, isClient pb.IdentityStoreClient, raClient raPb.ResourceAggregateClient) {
	const connID = "conn-Id"
	var conSeq uint64
	incSeq := func() uint64 {
		conSeq++
		return conSeq
	}
	_, err := isClient.AddDevice(ctx, &pb.AddDeviceRequest{
		DeviceId: deviceID,
	})
	assert.NoError(t, err) //nolint:testifylint

	for {
		_, err = raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
			DeviceId:      deviceID,
			CorrelationId: uuid.NewString(),
			Update: &commands.UpdateDeviceMetadataRequest_Connection{
				Connection: &commands.Connection{
					Status:         commands.Connection_ONLINE,
					ConnectedAt:    time.Now().UnixNano(),
					Protocol:       protocol,
					ServiceId:      "a0000000-0000-0000-0000-000000000099",
					LocalEndpoints: []string{"coaps+tcp://localhost:5684"},
				},
			},
			TimeToLive: time.Now().Add(time.Hour).UnixNano(),
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: connID,
				Sequence:     incSeq(),
			},
		})
		if err == nil {
			break
		}
		if s, ok := status.FromError(err); ok && s.Code() == codes.PermissionDenied {
			time.Sleep(time.Millisecond * 10)
			// device is still not loaded to owner in resource-aggregate
			continue
		}
		assert.NoError(t, err) //nolint:testifylint
	}

	resources := CreateDeviceResourceLinks(deviceID, numResources, tdEnabled)
	pub := commands.PublishResourceLinksRequest{
		DeviceId:  deviceID,
		Resources: resources,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connID,
			Sequence:     incSeq(),
		},
	}
	_, err = raClient.PublishResourceLinks(ctx, &pub)
	assert.NoError(t, err) //nolint:testifylint

	_, err = raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId:      deviceID,
		CorrelationId: uuid.NewString(),
		Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
			TwinSynchronization: &commands.TwinSynchronization{
				State:     commands.TwinSynchronization_SYNCING,
				SyncingAt: time.Now().UnixNano(),
			},
		},
		TimeToLive: time.Now().Add(time.Hour).UnixNano(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connID,
			Sequence:     incSeq(),
		},
	})
	assert.NoError(t, err) //nolint:testifylint

	for i := range numResources {
		_, err = raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
			ResourceId: commands.NewResourceID(deviceID, fmt.Sprintf("/res-%v", i)),
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: connID,
				Sequence:     incSeq(),
			},
			Content: &commands.Content{
				Data:        []byte(fmt.Sprintf("content res-%v", i)),
				ContentType: message.TextPlain.String(),
			},
			Status: commands.Status_OK,
		})
		assert.NoError(t, err) //nolint:testifylint
	}

	_, err = raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connID,
			Sequence:     incSeq(),
		},
		Content: &commands.Content{
			Data:        test.EncodeToCbor(t, &device.Device{Name: name, ID: deviceID, ResourceTypes: []string{device.ResourceType}, Interfaces: []string{interfaces.OC_IF_BASELINE}}),
			ContentType: message.AppOcfCbor.String(),
		},
		Status: commands.Status_OK,
	})
	assert.NoError(t, err) //nolint:testifylint

	_, err = raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(deviceID, "/oic/p"),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connID,
			Sequence:     incSeq(),
		},
		Content: &commands.Content{
			Data:        test.EncodeToCbor(t, &platform.Platform{ResourceTypes: []string{device.ResourceType}, Interfaces: []string{interfaces.OC_IF_BASELINE}, SerialNumber: fmt.Sprintf("sn %v", deviceID)}),
			ContentType: message.AppOcfCbor.String(),
		},
		Status: commands.Status_OK,
	})
	assert.NoError(t, err) //nolint:testifylint

	_, err = raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId:      deviceID,
		CorrelationId: uuid.NewString(),
		Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
			TwinSynchronization: &commands.TwinSynchronization{
				State:    commands.TwinSynchronization_IN_SYNC,
				InSyncAt: time.Now().UnixNano(),
			},
		},
		TimeToLive: time.Now().Add(time.Hour).UnixNano(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connID,
			Sequence:     incSeq(),
		},
	})
	assert.NoError(t, err)
}

func CreateDevices(ctx context.Context, t *testing.T, numDevices int, numResourcesPerDevice int, protocol commands.Connection_Protocol) {
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	isConn, err := grpc.NewClient(config.IDENTITY_STORE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = isConn.Close()
	}()
	isClient := pb.NewIdentityStoreClient(isConn)

	raConn, err := grpc.NewClient(config.RESOURCE_AGGREGATE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raClient := raPb.NewResourceAggregateClient(raConn)

	numGoRoutines := int64(8)
	sem := semaphore.NewWeighted(numGoRoutines)
	for i := range numDevices {
		err = sem.Acquire(ctx, 1)
		require.NoError(t, err)
		go func(i int) {
			CreateDevice(ctx, t, fmt.Sprintf("dev-%v", i), uuid.NewString(), numResourcesPerDevice, false, protocol, isClient, raClient)
			sem.Release(1)
		}(i)
	}
	err = sem.Acquire(ctx, numGoRoutines)
	require.NoError(t, err)
}
