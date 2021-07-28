package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/go-coap/v2/message"

	"github.com/plgd-dev/go-coap/v2/tcp"

	"github.com/plgd-dev/cloud/pkg/log"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientRetrieveHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	type args struct {
		path  string
		query string
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name: "invalid href",
			args: args{
				path: uri.ResourceRoute + TestAResourceHref,
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name: "not found",
			args: args{
				path: uri.ResourceRoute + "/dev0/res0",
			},
			wantsCode: coapCodes.NotFound,
		},
		{
			name: "found",
			args: args{
				path: uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "found with interface",
			args: args{
				path:  uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
				query: "if=oic.if.baseline",
			},
			wantsCode: coapCodes.Content,
		},
	}

	testPrepareDevice(t, co)
	time.Sleep(time.Second) // for publish content of device resources

	log.Setup(log.Config{
		Debug: true,
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := tcp.NewGetRequest(ctx, tt.args.path)
			require.NoError(t, err)
			if tt.args.query != "" {
				req.SetOptionString(message.URIQuery, tt.args.query)
			}
			resp, err := co.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
		})
	}
}
