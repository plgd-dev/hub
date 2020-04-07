package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-ocf/ocf-cloud/coap-gateway/uri"
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
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

	NewPostRequest := func(interval int64) gocoap.Message {
		ping := oicwkping{
			Interval: interval,
		}
		out, err := cbor.Encode(ping)
		msg, err := co.NewPostRequest(uri.ResourcePing, gocoap.AppCBOR, bytes.NewBuffer(out))
		assert.NoError(t, err)
		return msg
	}
	NewGetRequest := func() gocoap.Message {
		msg, err := co.NewGetRequest(uri.ResourcePing)
		assert.NoError(t, err)
		return msg
	}

	type args struct {
		req gocoap.Message
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name: "invalid interval",
			args: args{
				req: NewPostRequest(0),
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name: "get configuration",
			args: args{
				req: NewGetRequest(),
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "ping",
			args: args{
				req: NewPostRequest(1),
			},
			wantsCode: coapCodes.Valid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			resp, err := co.ExchangeWithContext(ctx, tt.args.req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantsCode, resp.Code())
		})
	}
}
