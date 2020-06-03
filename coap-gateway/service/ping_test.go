package service_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/tcp"

	authTest "github.com/go-ocf/cloud/authorization/test"
	coapgwTest "github.com/go-ocf/cloud/coap-gateway/test"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	raTest "github.com/go-ocf/cloud/resource-aggregate/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resourcePingHandler(t *testing.T) {
	defer authTest.SetUp(t)
	defer raTest.SetUp(t)
	defer coapgwTest.SetUp(t)

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	type args struct {
		ping interface{} // nill means get, otherwise it is ping
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name: "invalid interval",
			args: args{
				ping: map[interface{}]interface{}{
					"in": 0,
				},
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name:      "get configuration",
			args:      args{},
			wantsCode: coapCodes.Content,
		},
		{
			name: "ping",
			args: args{
				ping: map[interface{}]interface{}{
					"in": 1,
				},
			},
			wantsCode: coapCodes.Valid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			var req *pool.Message
			var err error
			if tt.args.ping != nil {
				out, err := cbor.Encode(tt.args.ping)
				require.NoError(t, err)
				req, err = tcp.NewPostRequest(ctx, uri.ResourcePing, message.AppCBOR, bytes.NewReader(out))
				require.NoError(t, err)
			} else {
				req, err = tcp.NewGetRequest(ctx, uri.ResourcePing)
				require.NoError(t, err)
			}
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode, resp.Code())
		})
	}
}
