package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

func Test_clientUpdateHandler(t *testing.T) {
	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	config.ResourceDirectoryAddr = "localhost:12349"
	config.RequestTimeout = time.Second * 7
	config.Net = "tcp-tls"
	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownRA := testCreateResourceAggregate(t, resourceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownRA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	NewPostRequest := func(href string, contentFormat gocoap.MediaType, payload []byte) gocoap.Message {
		msg, err := co.NewPostRequest(href, contentFormat, bytes.NewBuffer(payload))
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
			name: "invalid href",
			args: args{
				req: NewPostRequest(resourceRoute+TestAResourceHref, gocoap.TextPlain, []byte{}),
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name: "not found",
			args: args{
				req: NewPostRequest(resourceRoute+"/a/b", gocoap.TextPlain, []byte{}),
			},
			wantsCode: coapCodes.NotFound,
		},
		{
			name: "valid",
			args: args{
				req: NewPostRequest(resourceRoute+"/"+CertIdentity+TestAResourceHref, gocoap.TextPlain, []byte("data")),
			},
			wantsCode: coapCodes.Changed,
		},
	}

	testPrepareDevice(t, co)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			resp, err := co.ExchangeWithContext(ctx, tt.args.req)
			assert.NoError(t, err)
			if resp != nil {
				assert.Equal(t, tt.wantsCode, resp.Code())
			}
		})
	}
}
