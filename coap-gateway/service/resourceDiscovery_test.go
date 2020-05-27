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

func Test_resourceDirectoryFind(t *testing.T) {
	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	config.ResourceDirectoryAddr = "localhost:12349"
	config.RequestTimeout = time.Second * 2
	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownRA := testCreateResourceAggregate(t, resourceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownRA()
	shutdownRD := testCreateResourceDirectory(t, resourceDB, config.ResourceDirectoryAddr, config.AuthServerAddr)
	defer shutdownRD()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	type args struct {
		queries []string
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name:      "without query",
			args:      args{},
			wantsCode: coapCodes.Content,
		},
		{
			name: "with di",
			args: args{
				queries: []string{"di=" + CertIdentity},
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "with rt",
			args: args{
				queries: []string{"rt=" + TestBResourceType},
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "with di & rt",
			args: args{
				queries: []string{"di=" + CertIdentity, "rt=" + TestAResourceType},
			},
			wantsCode: coapCodes.Content,
		},
		{
			name: "di not exist",
			args: args{
				queries: []string{"di=1234"},
			},
			wantsCode: coapCodes.NotFound,
		},
	}

	testPrepareDevice(t, co)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := tcp.NewGetRequest(ctx, "/oic/res")
			require.NoError(t, err)
			for _, q := range tt.args.queries {
				req.AddQuery(q)
			}
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode, resp.Code())
		})
	}
}
