package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	test "github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestClientObserveHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "", true, time.Now().Add(time.Minute))
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	const invalidResPath = uri.ResourceRoute + "/dev0/res0"
	type args struct {
		path    string
		observe uint32
		token   []byte
	}
	tests := []struct {
		name      string
		args      args
		wantsCode codes.Code
	}{
		{
			name: "invalid observe",
			args: args{
				path:    invalidResPath,
				observe: 123,
				token:   nil,
			},
			wantsCode: codes.BadRequest,
		},
		{
			name: "observe - not exist resource",
			args: args{
				path:    invalidResPath,
				observe: 0,
				token:   nil,
			},
			wantsCode: codes.Unauthorized,
		},

		{
			name: "unobserve - not exist resource",
			args: args{
				path:    invalidResPath,
				observe: 1,
				token:   nil,
			},
			wantsCode: codes.BadRequest,
		},
		{
			name: "observe",
			args: args{
				path:    uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 0,
				token:   []byte("observe"),
			},
			wantsCode: codes.Content,
		},
		{
			name: "unobserve",
			args: args{
				path:    uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 1,
				token:   []byte("observe"),
			},
			wantsCode: codes.Content,
		},
	}

	testPrepareDevice(t, co)
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := tcp.NewGetRequest(ctx, pool.New(0, 0), tt.args.path)
			require.NoError(t, err)
			req.SetObserve(tt.args.observe)
			if tt.args.token != nil {
				req.SetToken(tt.args.token)
			}
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
		})
	}
}

func TestClientObserveHandlerCloseObservation(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co1 := testCoapDial(t, testCfg.GW_HOST, "", true, time.Now().Add(time.Minute))
	require.NotEmpty(t, co1)
	defer func() {
		_ = co1.Close()
	}()
	testPrepareDevice(t, co1)
	co2 := testCoapDial(t, testCfg.GW_HOST, "", true, time.Now().Add(time.Minute))
	require.NotEmpty(t, co1)
	defer func() {
		_ = co2.Close()
	}()
	testSignUpIn(t, "observeClient", co2)

	time.Sleep(time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()
	done := make(chan struct{})
	_, err := co2.Observe(ctx, uri.ResourceRoute+"/"+CertIdentity+TestAResourceHref, func(req *pool.Message) {
		fmt.Printf("%+v", req)
		if req.Code() == codes.ServiceUnavailable {
			close(done)
		}
	})
	require.NoError(t, err)

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	_, err = c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIdFilter: []string{CertIdentity},
	})
	require.NoError(t, err)

	select {
	case <-done:
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
}