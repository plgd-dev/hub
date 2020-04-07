package service

import (
	"context"
	"testing"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

func Test_clientResetHandler(t *testing.T) {
	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
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

	NewReset := func(token []byte) gocoap.Message {
		msg := co.NewMessage(gocoap.MessageParams{
			Code:  coapCodes.Empty,
			Token: token,
		})
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
			name: "observe",
			args: args{
				req: NewGetRequest(resourceRoute+"/"+CertIdentity+TestAResourceHref, 0, []byte("observe")),
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "reset",
			args: args{
				req: NewReset([]byte("observe")),
			},
			wantsCode: coapCodes.Empty,
		},
		{
			name: "unobserve",
			args: args{
				req: NewGetRequest(resourceRoute+"/"+CertIdentity+TestAResourceHref, 1, []byte("observe")),
			},
			wantsCode: coapCodes.BadRequest,
		},
	}

	testPrepareDevice(t, co)
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.Code() == coapCodes.Empty {
				ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
				defer cancel()
				err := co.WriteMsgWithContext(ctx, tt.args.req)
				assert.NoError(t, err)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
				defer cancel()
				resp, err := co.ExchangeWithContext(ctx, tt.args.req)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantsCode, resp.Code())
			}
			time.Sleep(time.Second) // to avoid reorder test case
		})
	}
}
