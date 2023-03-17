//go:build test
// +build test

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRetrieveHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := co.NewGetRequest(ctx, tt.args.path)
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
