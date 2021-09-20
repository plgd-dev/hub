package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientObserveHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

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
				path:    uri.ResourceRoute + "/dev0/res0",
				observe: 123,
				token:   nil,
			},
			wantsCode: codes.BadRequest,
		},
		{
			name: "observe - not exist resource",
			args: args{
				path:    uri.ResourceRoute + "/dev0/res0",
				observe: 0,
				token:   nil,
			},
			wantsCode: codes.Unauthorized,
		},

		{
			name: "unobserve - not exist resource",
			args: args{
				path:    uri.ResourceRoute + "/dev0/res0",
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
			req, err := tcp.NewGetRequest(ctx, tt.args.path)
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

func Test_clientObserveHandler_closeObservation(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co1 := testCoapDial(t, testCfg.GW_HOST, "")
	require.NotEmpty(t, co1)
	defer func() {
		_ = co1.Close()
	}()
	testPrepareDevice(t, co1)
	co2 := testCoapDial(t, testCfg.GW_HOST, "")
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

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))
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
