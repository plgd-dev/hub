package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/go-coap/v2/tcp"

	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				path:    resourceRoute + "/dev0/res0",
				observe: 123,
				token:   nil,
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "observe - not exist resource",
			args: args{
				path:    resourceRoute + "/dev0/res0",
				observe: 0,
				token:   nil,
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "unobserve - not exist resource",
			args: args{
				path:    resourceRoute + "/dev0/res0",
				observe: 1,
				token:   nil,
			},
			wantsCode: coapCodes.BadRequest,
		},

		{
			name: "observe",
			args: args{
				path:    resourceRoute + "/" + CertIdentity + TestAResourceHref,
				observe: 0,
				token:   []byte("observe"),
			},
			wantsCode: coapCodes.Content,
		},

		{
			name: "unobserve",
			args: args{
				path:    resourceRoute + "/" + CertIdentity + TestAResourceHref,
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
