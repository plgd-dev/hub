package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/authorization/oauth"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	c2cConnectorTest "github.com/plgd-dev/cloud/cloud2cloud-connector/test"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	c2cGwUri "github.com/plgd-dev/cloud/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/kit/codec/json"
)

func setUp(ctx context.Context, t *testing.T, deviceID string, supportedEvents store.Events) func() {
	cloud1 := test.SetUp(ctx, t)
	cloud2 := c2cConnectorTest.SetUpCloudWithConnector(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	cloud1Conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c1 := pb.NewGrpcGatewayClient(cloud1Conn)
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c1, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

	rootCAs := make([]string, 0, 1)
	certs := test.GetRootCertificateAuthorities(t)
	for _, c := range certs {
		rootCAs = append(rootCAs, string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c.Raw,
		})))
	}

	linkedCloud := store.LinkedCloud{
		Name: t.Name(),
		Endpoint: store.Endpoint{
			URL:     "https://" + testCfg.C2C_GW_HOST + c2cGwUri.Version,
			RootCAs: rootCAs,
		},
		OAuth: oauth.Config{
			ClientID:     oauthService.ClientTest,
			Audience:     testCfg.C2C_GW_HOST,
			ClientSecret: "testClientSecret",
			//Scopes:       []string{"testScopes"},
			AuthURL:  testCfg.OAUTH_MANAGER_ENDPOINT_AUTHURL,
			TokenURL: testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL,
		},
		SupportedSubscriptionsEvents: supportedEvents,
	}
	data, err := json.Encode(linkedCloud)
	require.NoError(t, err)

	req := test.NewHTTPRequest(http.MethodPost, "https://"+c2cConnectorTest.C2C_CONNECTOR_HOST+uri.LinkedClouds, bytes.NewBuffer(data)).AuthToken(oauthTest.GetServiceToken(t)).Build(ctx, t)
	resp := test.DoHTTPRequest(t, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	var linkCloud store.LinkedCloud
	err = json.ReadFrom(resp.Body, &linkCloud)
	require.NoError(t, err)
	req = test.NewHTTPRequest(http.MethodGet, "https://"+c2cConnectorTest.C2C_CONNECTOR_HOST+uri.Version+"/clouds/"+linkCloud.ID+"/accounts", nil).AuthToken(oauthTest.GetServiceToken(t)).Build(ctx, t)
	resp = test.DoHTTPRequest(t, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// for pulling
	time.Sleep(time.Second * 10)

	req = test.NewHTTPRequest(http.MethodGet, "https://"+c2cConnectorTest.C2C_CONNECTOR_HOST+uri.Version+"/clouds", nil).AuthToken(oauthTest.GetServiceToken(t)).Build(ctx, t)
	resp = test.DoHTTPRequest(t, req)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	fmt.Println(string(b))

	return func() {
		req := test.NewHTTPRequest(http.MethodDelete, "https://"+c2cConnectorTest.C2C_CONNECTOR_HOST+uri.Version+"/clouds/"+linkCloud.ID, nil).AuthToken(oauthTest.GetServiceToken(t)).Build(ctx, t)
		resp := test.DoHTTPRequest(t, req)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()

		cloud2()
		shutdownDevSim()
		cloud1Conn.Close()
		cloud1()
	}
}

func testRequestHandler_GetDevices(t *testing.T, events store.Events) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Device
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetDevicesRequest{},
			},
			want: []*pb.Device{
				{
					Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
					Interfaces: []string{"oic.if.r", "oic.if.baseline"},
					Id:         deviceID,
					Name:       test.TestDeviceName,
					IsOnline:   true,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()
	tearDown := setUp(ctx, t, deviceID, events)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(c2cConnectorTest.GRPC_GATEWAY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetDevices(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				devices := make([]*pb.Device, 0, 1)
				for {
					dev, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					assert.NotEmpty(t, dev.ProtocolIndependentId)
					dev.ProtocolIndependentId = ""
					devices = append(devices, dev)
				}
				test.CheckProtobufs(t, tt.want, devices, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}

func TestRequestHandler_GetDevices(t *testing.T) {
	type args struct {
		events store.Events
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "full pulling",
		},
		{
			name: "full events",
			args: args{
				events: store.Events{
					Devices:  events.AllDevicesEvents,
					Device:   events.AllDeviceEvents,
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "resource events + device,devices pulling",
			args: args{
				events: store.Events{
					Resource: events.AllResourceEvents,
				},
			},
		},

		{
			name: "resource, device events + devices pulling",
			args: args{
				events: store.Events{
					Device:   events.AllDeviceEvents,
					Resource: events.AllResourceEvents,
				},
			},
		},

		{
			name: "device, devices events + resource pulling",
			args: args{
				events: store.Events{
					Devices: events.AllDevicesEvents,
					Device:  events.AllDeviceEvents,
				},
			},
		},

		{
			name: "pull resource, devices + static device events",
			args: args{
				events: store.Events{
					StaticDeviceEvents: true,
				},
			},
		},
		{
			name: "resource, devices events + static device events",
			args: args{
				events: store.Events{
					Devices:            events.AllDevicesEvents,
					Resource:           events.AllResourceEvents,
					StaticDeviceEvents: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRequestHandler_GetDevices(t, tt.args.events)
		})
	}
}
