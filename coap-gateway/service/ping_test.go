package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/tcp"

	"github.com/go-ocf/cloud/coap-gateway/uri"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resourcePingHandler(t *testing.T) {
	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	config.ResourceDirectoryAddr = "localhost:12349"
	deviceDB := t.Name() + "_deviceDB"
	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownDA := testCreateResourceAggregate(t, deviceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownDA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	type args struct {
		ping *oicwkping // nill means get, otherwise it is ping
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name: "invalid interval",
			args: args{
				ping: &oicwkping{
					Interval: 0,
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
				ping: &oicwkping{
					Interval: 1,
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
