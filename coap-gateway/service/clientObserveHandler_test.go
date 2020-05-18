package service

import (
	"context"
	"testing"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

func Test_clientObserveHandler(t *testing.T) {
	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.Addr = "127.0.0.1:5684"
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	config.ResourceDirectoryAddr = "localhost:12349"

	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownRA := testCreateResourceAggregate(t, resourceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownRA()
	shutdownRS := testCreateResourceDirectory(t, resourceDB, config.ResourceDirectoryAddr, config.AuthServerAddr)
	defer shutdownRS()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	NewGetRequest := func(path string, observe uint32, token []byte) gocoap.Message {
		msg, err := co.NewGetRequest(path)
		msg.SetObserve(observe)
		if token != nil {
			msg.SetToken(token)
		}
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
			name: "invalid observe",
			args: args{
				req: NewGetRequest(resourceRoute+"/dev0/res0", 123, nil),
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "observe - not exist resource",
			args: args{
				req: NewGetRequest(resourceRoute+"/dev0/res0", 0, nil),
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "unobserve - not exist resource",
			args: args{
				req: NewGetRequest(resourceRoute+"/dev0/res0", 1, nil),
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "observe",
			args: args{
				req: NewGetRequest(resourceRoute+"/"+CertIdentity+TestAResourceHref, 0, []byte("observe")),
			},
			wantsCode: coapCodes.Content,
		},

		{
			name: "unobserve",
			args: args{
				req: NewGetRequest(resourceRoute+"/"+CertIdentity+TestAResourceHref, 1, []byte("observe")),
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
			resp, err := co.ExchangeWithContext(ctx, tt.args.req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantsCode, resp.Code())
		})
	}
}
