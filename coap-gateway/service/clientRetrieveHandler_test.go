package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/kit/log"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

func Test_clientRetrieveHandler(t *testing.T) {
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

	NewGetRequest := func(path string) gocoap.Message {
		msg, err := co.NewGetRequest(path)
		assert.NoError(t, err)
		return msg
	}

	NewGetRequestWithInterface := func(path, resourceInterface string) gocoap.Message {
		msg, err := co.NewGetRequest(path)
		assert.NoError(t, err)
		msg.AddOption(gocoap.URIQuery, "if="+resourceInterface)
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
				req: NewGetRequest(resourceRoute + TestAResourceHref),
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name: "not found",
			args: args{
				req: NewGetRequest(resourceRoute + "/dev0/res0"),
			},
			wantsCode: coapCodes.NotFound,
		},
		{
			name: "found",
			args: args{
				req: NewGetRequest(resourceRoute + "/" + CertIdentity + TestAResourceHref),
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "found with interface",
			args: args{
				req: NewGetRequestWithInterface(resourceRoute+"/"+CertIdentity+TestAResourceHref, "oic.if.baseline"),
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
			resp, err := co.ExchangeWithContext(ctx, tt.args.req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
		})
	}
}
