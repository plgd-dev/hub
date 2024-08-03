package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	m2mOauthTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
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

func getResources(ctx context.Context, c pb.GrpcGatewayClient, req *pb.GetResourcesRequest) ([]*pb.Resource, error) {
	client, err := c.GetResources(ctx, req)
	if err != nil {
		return nil, err
	}
	values := make([]*pb.Resource, 0, 1)
	for {
		value, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestRequestHandlerGetResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()
	const switchID = "1"
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)...)
	// for update resource-directory cache
	time.Sleep(time.Second)

	// get resource from device via HUB
	lightResourceData, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
	})
	require.NoError(t, err)

	type args struct {
		req *pb.GetResourcesRequest
	}
	tests := []struct {
		name  string
		args  args
		cmpFn func(*testing.T, []*pb.Resource, []*pb.Resource)
		want  []*pb.Resource
	}{
		{
			name: "invalid deviceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{"unknown"},
				},
			},
		},
		{
			name: "invalid resourceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID("unknown", ""),
						},
					},
				},
			},
		},
		{
			name: "invalid typeFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					TypeFilter: []string{"unknown"},
				},
			},
		},
		{
			name: "valid deviceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{deviceID},
				},
			},
			cmpFn: pbTest.CmpResourceValuesBasic,
			want:  test.ResourceLinksToResources2(deviceID, resourceLinks),
		},
		{
			name: "valid resourceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
						},
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.CORE_LIGHT},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
						map[string]interface{}{
							"state": false,
							"power": uint64(0),
							"name":  "Light",
						}),
				},
			},
		},
		{
			name: "valid resourceIdFilter with ETag",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
							Etag: [][]byte{
								lightResourceData.GetData().GetEtag(),
							},
						},
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.CORE_LIGHT},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     test.TestResourceLightInstanceHref("1"),
						},
						Status: commands.Status_NOT_MODIFIED,
						Content: &commands.Content{
							CoapContentFormat: -1,
						},
						AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
						ResourceTypes: test.TestResourceLightInstanceResourceTypes,
					},
				},
			},
		},
		{
			name: "valid typeFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					TypeFilter: []string{types.BINARY_SWITCH},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.BINARY_SWITCH},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "",
						map[string]interface{}{
							"value": false,
						}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := getResources(ctx, c, tt.args.req)
			require.NoError(t, err)
			if tt.cmpFn != nil {
				tt.cmpFn(t, tt.want, values)
				return
			}
			pbTest.CmpResourceValues(t, tt.want, values)
		})
	}
}

func TestRequestHandlerGetResourcesWithM2MTokenVerification(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	grpcCfg := grpcgwTest.MakeConfig(t)
	grpcCfg.APIs.GRPC.Authorization.TokenVerification.CacheExpiration = time.Second * 2
	tearDown := service.SetUp(ctx, t, service.WithGRPCGWConfig(grpcCfg))
	defer tearDown()
	validTokenStr := oauthTest.GetDefaultAccessToken(t)
	ctxWithToken := pkgGrpc.CtxWithToken(ctx, validTokenStr)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctxWithToken, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	req := &pb.GetResourcesRequest{ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1"))}}}
	exp := &pb.Resource{
		Types: []string{types.CORE_LIGHT},
		Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
			map[string]interface{}{
				"state": false,
				"power": uint64(0),
				"name":  "Light",
			}),
	}

	tokenStr := m2mOauthTest.GetDefaultAccessToken(t)
	values, err := getResources(pkgGrpc.CtxWithToken(ctx, tokenStr), c, req)
	require.NoError(t, err)
	require.NotEmpty(t, values)
	pbTest.CmpResourceValues(t, []*pb.Resource{exp.Clone()}, values)

	// invalid token
	_, err = getResources(pkgGrpc.CtxWithToken(ctx, "invalid"), c, req)
	require.Error(t, err)

	// whitelisted tokens expire
	time.Sleep(grpcCfg.APIs.GRPC.Authorization.TokenVerification.CacheExpiration + time.Second)

	// blacklist the token
	token, err := pkgJwt.ParseToken(tokenStr)
	require.NoError(t, err)
	tokenID, err := token.GetID()
	require.NoError(t, err)
	m2mOauthTest.DeleteTokens(ctx, t, []string{tokenID}, validTokenStr)
	// request should fail
	_, err = getResources(pkgGrpc.CtxWithToken(ctx, tokenStr), c, req)
	require.ErrorContains(t, err, pkgJwt.ErrBlackListedToken.Error())

	// repeated requests should still fail, but use the cache
	_, err = getResources(pkgGrpc.CtxWithToken(ctx, tokenStr), c, req)
	require.ErrorContains(t, err, pkgJwt.ErrBlackListedToken.Error())

	// non-blacklisted tokens should still work
	values, err = getResources(pkgGrpc.CtxWithToken(ctx, validTokenStr), c, req)
	require.NoError(t, err)
	require.NotEmpty(t, values)
	pbTest.CmpResourceValues(t, []*pb.Resource{exp.Clone()}, values)

	// parallel whitelisted requests -> cache should be used, only a single request should be made
	var wg sync.WaitGroup
	tokenStr2 := m2mOauthTest.GetDefaultAccessToken(t)
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			values2, err2 := getResources(pkgGrpc.CtxWithToken(ctx, tokenStr2), c, req)
			assert.NoError(t, err2)
			assert.NotEmpty(t, values2)
			pbTest.CmpResourceValues(t, []*pb.Resource{exp.Clone()}, values2)
		}()
	}
	wg.Wait()

	// wait for expiration
	time.Sleep(grpcCfg.APIs.GRPC.Authorization.TokenVerification.CacheExpiration)

	// blacklist the token
	tokenStr3 := m2mOauthTest.GetDefaultAccessToken(t)
	token, err = pkgJwt.ParseToken(tokenStr3)
	require.NoError(t, err)
	tokenID, err = token.GetID()
	require.NoError(t, err)
	m2mOauthTest.DeleteTokens(ctx, t, []string{tokenID}, validTokenStr)
	// parallel blacklisted requests -> cache should be used, only a single request should be made
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err2 := getResources(pkgGrpc.CtxWithToken(ctx, tokenStr3), c, req)
			assert.ErrorContains(t, err2, pkgJwt.ErrBlackListedToken.Error())
		}()
	}
	wg.Wait()
}
