package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
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

	type args struct {
		code    codes.Code
		token   message.Token
		observe uint32
		path    string
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name: "observe",
			args: args{
				code:    codes.GET,
				path:    resourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 0,
				token:   message.Token("observe"),
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "reset",
			args: args{
				code:  codes.Empty,
				token: message.Token("observe"),
			},
			wantsCode: coapCodes.Empty,
		},
		{
			name: "unobserve",
			args: args{
				code:    codes.GET,
				path:    resourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 1,
				token:   message.Token("observe"),
			},
			wantsCode: coapCodes.BadRequest,
		},
	}

	testPrepareDevice(t, co)
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.code == coapCodes.Empty {
				ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
				defer cancel()
				msg := pool.AcquireMessage(ctx)
				msg.SetCode(tt.args.code)
				msg.SetToken(tt.args.token)
				err := co.WriteMessage(msg)
				assert.NoError(t, err)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
				defer cancel()
				msg := pool.AcquireMessage(ctx)
				msg.SetCode(tt.args.code)
				msg.SetToken(tt.args.token)
				msg.SetPath(tt.args.path)
				msg.SetObserve(tt.args.observe)
				resp, err := co.Do(msg)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantsCode, resp.Code())
			}
			time.Sleep(time.Second) // to avoid reorder test case
		})
	}
}
