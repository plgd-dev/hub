package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	coapService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func onboardDeviceAndGetDevice(ctx context.Context, t *testing.T, deviceID string, oauthCfg oauthService.Config, coapCfg coapService.Config) (*pb.Device, time.Time /*startOnboard*/, time.Duration /*delta*/) {
	tearDown := service.SetUp(ctx, t, service.WithOAuthConfig(oauthCfg), service.WithCOAPGWConfig(coapCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	startOnboard := time.Now()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	deltaOnboard := time.Since(startOnboard) / 2

	// for update resource-directory cache
	time.Sleep(time.Second)

	client, err := c.GetDevices(ctx, &pb.GetDevicesRequest{})
	require.NoError(t, err)
	devices := make([]*pb.Device, 0, 1)
	for {
		dev, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		assert.NotEmpty(t, dev.ProtocolIndependentId)
		dev.ProtocolIndependentId = ""
		devices = append(devices, dev)
	}
	require.Len(t, devices, 1)
	return devices[0], startOnboard, deltaOnboard
}

func TestDevicesStatusUpdaterEnabledAndDeviceAccessTokenHasNoExpiration(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	oauthCfg := oauthTest.MakeConfig(t)
	oauthCfg.OAuthSigner.Clients.Find(config.OAUTH_MANAGER_CLIENT_ID).AccessTokenLifetime = 0
	coapCfg := coapgwTest.MakeConfig(t)
	expiresIn := time.Second * 20
	coapCfg.Clients.ResourceAggregate.DeviceStatusExpiration.Enabled = true
	coapCfg.Clients.ResourceAggregate.DeviceStatusExpiration.ExpiresIn = expiresIn

	device, startOnboard, deltaOnboard := onboardDeviceAndGetDevice(ctx, t, deviceID, oauthCfg, coapCfg)
	expectedOnlineValidUntil := startOnboard.Add(expiresIn + deltaOnboard).UnixNano()

	assert.Equal(t, commands.Connection_ONLINE, device.Metadata.Connection.Status)
	assert.InDelta(t, expectedOnlineValidUntil, device.Metadata.Connection.OnlineValidUntil, float64(deltaOnboard))
}

func TestDevicesStatusUpdaterDisabledAndDeviceAccessTokenHasNoExpiration(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	oauthCfg := oauthTest.MakeConfig(t)
	oauthCfg.OAuthSigner.Clients.Find(config.OAUTH_MANAGER_CLIENT_ID).AccessTokenLifetime = 0
	coapCfg := coapgwTest.MakeConfig(t)
	coapCfg.Clients.ResourceAggregate.DeviceStatusExpiration.Enabled = false

	device, _, _ := onboardDeviceAndGetDevice(ctx, t, deviceID, oauthCfg, coapCfg)

	assert.Equal(t, commands.Connection_ONLINE, device.Metadata.Connection.Status)
	assert.Equal(t, int64(0), device.Metadata.Connection.OnlineValidUntil)
}

func TestDevicesStatusUpdaterDisabledAndDeviceAccessTokenHasExpiration(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	oauthCfg := oauthTest.MakeConfig(t)
	accessTokenLifetime := time.Second * 10
	oauthCfg.OAuthSigner.Clients.Find(config.OAUTH_MANAGER_CLIENT_ID).AccessTokenLifetime = accessTokenLifetime
	coapCfg := coapgwTest.MakeConfig(t)
	coapCfg.Clients.ResourceAggregate.DeviceStatusExpiration.Enabled = false

	device, startOnboard, deltaOnboard := onboardDeviceAndGetDevice(ctx, t, deviceID, oauthCfg, coapCfg)
	expectedOnlineValidUntil := startOnboard.Add(accessTokenLifetime + deltaOnboard).UnixNano()

	assert.Equal(t, commands.Connection_ONLINE, device.Metadata.Connection.Status)
	assert.InDelta(t, expectedOnlineValidUntil, device.Metadata.Connection.OnlineValidUntil, float64(deltaOnboard))
}

func TestDevicesStatusUpdaterEnabledAndDeviceAccessTokenHasExpiration(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	oauthCfg := oauthTest.MakeConfig(t)
	oauthCfg.OAuthSigner.Clients.Find(config.OAUTH_MANAGER_CLIENT_ID).AccessTokenLifetime = time.Hour
	coapCfg := coapgwTest.MakeConfig(t)
	expiresIn := time.Second * 20
	coapCfg.Clients.ResourceAggregate.DeviceStatusExpiration.Enabled = true
	coapCfg.Clients.ResourceAggregate.DeviceStatusExpiration.ExpiresIn = expiresIn

	device, startOnboard, deltaOnboard := onboardDeviceAndGetDevice(ctx, t, deviceID, oauthCfg, coapCfg)
	expectedOnlineValidUntil := startOnboard.Add(expiresIn + deltaOnboard).UnixNano()

	assert.Equal(t, commands.Connection_ONLINE, device.Metadata.Connection.Status)
	assert.InDelta(t, expectedOnlineValidUntil, device.Metadata.Connection.OnlineValidUntil, float64(deltaOnboard))
}
