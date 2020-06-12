package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/go-coap/v2/tcp"

	"github.com/go-ocf/cloud/coap-gateway/uri"
	testCfg "github.com/go-ocf/cloud/test/config"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientObserveHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	type args struct {
		path    string
		observe uint32
		token   []byte
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{

		{
			name: "invalid observe",
			args: args{
				path:    uri.ResourceRoute + "/dev0/res0",
				observe: 123,
				token:   nil,
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "observe - not exist resource",
			args: args{
				path:    uri.ResourceRoute + "/dev0/res0",
				observe: 0,
				token:   nil,
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "unobserve - not exist resource",
			args: args{
				path:    uri.ResourceRoute + "/dev0/res0",
				observe: 1,
				token:   nil,
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "observe",
			args: args{
				path:    uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 0,
				token:   []byte("observe"),
			},
			wantsCode: coapCodes.Content,
		},

		{
			name: "unobserve",
			args: args{
				path:    uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 1,
				token:   []byte("observe"),
			},
			wantsCode: coapCodes.Content,
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
			assert.Equal(t, tt.wantsCode, resp.Code())
		})
	}
}
