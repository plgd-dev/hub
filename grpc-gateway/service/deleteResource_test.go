package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	exCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestRequestHandlerDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		href string
		ttl  int64
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "/light/1 - MethodNotAllowed",
			args: args{
				href: test.TestResourceLightInstanceHref("1"),
			},
			wantErr:     true,
			wantErrCode: codes.Code(exCodes.MethodNotAllowed),
		},
		{
			name: "invalid Href",
			args: args{
				href: "/unknown",
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "/oic/d - PermissionDenied",
			args: args{
				href: device.ResourceURI,
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "invalid timeToLive",
			args: args{
				href: test.TestResourceLightInstanceHref("1"),
				ttl:  int64(99 * time.Millisecond),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "not found - delete /switches/-1",
			args: args{
				href: test.TestResourceSwitchesInstanceHref("-1"),
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "delete /switches/1",
			args: args{
				href: test.TestResourceSwitchesInstanceHref("1"),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.DeleteResourceRequest{
				ResourceId: commands.NewResourceID(deviceID, tt.args.href),
				TimeToLive: tt.args.ttl,
			}
			got, err := c.DeleteResource(ctx, req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode, status.Convert(err).Code())
				return
			}
			require.NoError(t, err)

			want := pbTest.MakeResourceDeleted(t, deviceID, tt.args.href, "")
			pbTest.CmpResourceDeleted(t, want, got.GetData())
		})
	}
}

func TestRequestHandlerDeleteResourceAfterUnpublish(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	const switchID1 = "1"
	const switchID2 = "2"
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID1, switchID2)
	time.Sleep(200 * time.Millisecond)

	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	cfg := raTest.MakeConfig(t)
	raConn, err := client.New(testCfg.MakeGrpcClientConfig(cfg.APIs.GRPC.Addr), fileWatcher, log.Get(), trace.NewNoopTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	rac := raService.NewResourceAggregateClient(raConn.GRPC())

	respUnpublish, err := rac.UnpublishResourceLinks(ctx, &commands.UnpublishResourceLinksRequest{
		Hrefs:    []string{test.TestResourceSwitchesInstanceHref(switchID2)},
		DeviceId: deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	})
	require.NoError(t, err)
	require.Len(t, respUnpublish.UnpublishedHrefs, 1)
	require.Equal(t, respUnpublish.UnpublishedHrefs[0], test.TestResourceSwitchesInstanceHref(switchID2))
	time.Sleep(200 * time.Millisecond)

	_, err = c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID2)),
	})
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID2)
	time.Sleep(200 * time.Millisecond)

	_, err = c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID1)),
	})
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	rlClient, err := c.GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdFilter: []string{deviceID},
	})
	require.NoError(t, err)
	links := make([]*events.ResourceLinksPublished, 0, 1)
	for {
		link, err := rlClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		links = append(links, link)
	}
	require.Len(t, links, 1)
	require.NotEmpty(t, pbTest.FindResourceInResourceLinksPublishedByHref(links[0], test.TestResourceSwitchesInstanceHref(switchID2)))
}
