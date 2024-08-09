package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/device/ocf"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/sdk"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestInvalidOwner(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUp(context.Background(), t)
	defer hubShutdown()

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	corID := "allEvents"
	subClient, subID := test.SubscribeToAllEvents(ctx, t, c, corID)

	dpsCfg := test.MakeConfig(t)
	dpsShutDown := test.New(t, dpsCfg)
	defer dpsShutDown()
	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)
	hubTest.WaitForDevice(t, subClient, ocf.NewDevice(deviceID, test.TestDeviceObtName), subID, corID, test.TestDevsimResources)
	err = subClient.CloseSend()
	require.NoError(t, err)

	subClient, err = client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func(s grpcPb.GrpcGateway_SubscribeToEventsClient) {
		errC := s.CloseSend()
		require.NoError(t, errC)
	}(subClient)
	subID, corID = test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)

	const fakeOwner = "1337"
	dc, err := sdk.NewClient(sdk.WithID(isEvents.OwnerToUUID(fakeOwner)))
	require.NoError(t, err)
	defer func() {
		_ = dc.Close(ctx)
	}()

	// /oic/d, /oic/p, /oic/res are readable to all
	err = dc.GetResource(ctx, deviceID, device.ResourceURI, nil)
	require.NoError(t, err)
	err = dc.GetResource(ctx, deviceID, platform.ResourceURI, nil)
	require.NoError(t, err)
	err = dc.GetResource(ctx, deviceID, resources.ResourceURI, nil)
	require.NoError(t, err)

	dpsOwnerUUID := isEvents.OwnerToUUID(test.DPSOwner)
	// everything else should fail
	// - get / update creds
	err = dc.GetResource(ctx, deviceID, credential.ResourceURI, nil)
	require.Error(t, err)
	err = dc.UpdateResource(ctx, deviceID, credential.ResourceURI, credential.CredentialUpdateRequest{
		ResourceOwner: dpsOwnerUUID,
		Credentials: []credential.Credential{{
			Type: credential.CredentialType_EMPTY,
		}},
	}, nil)
	require.Error(t, err)

	// - get / update ACLs
	err = dc.GetResource(ctx, deviceID, acl.ResourceURI, nil)
	require.Error(t, err)
	err = dc.UpdateResource(ctx, deviceID, credential.ResourceURI, acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: acl.Permission_WRITE,
				Resources: []acl.Resource{
					{
						Href: test.ResourcePlgdDpsHref,
					},
				},
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: dpsOwnerUUID,
					},
				},
			},
		},
	}, nil)
	require.Error(t, err)

	// - get cloud cfg
	err = dc.GetResource(ctx, deviceID, configuration.ResourceURI, nil)
	require.Error(t, err)

	// - get / update dps
	err = dc.GetResource(ctx, deviceID, test.ResourcePlgdDpsHref, nil)
	require.Error(t, err)
	err = dc.UpdateResource(ctx, deviceID, test.ResourcePlgdDpsHref, test.ResourcePlgdDps{ForceReprovision: true}, nil)
	require.Error(t, err)
}
