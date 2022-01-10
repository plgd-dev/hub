package service_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/device/schema/resources"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceDirectoryFind(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	type args struct {
		queries []string
	}
	tests := []struct {
		name         string
		args         args
		wantsCode    coapCodes.Code
		emptyContent bool
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
			wantsCode:    coapCodes.Content,
			emptyContent: true,
		},
	}

	testPrepareDevice(t, co)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := tcp.NewGetRequest(ctx, pool.New(0, 0), resources.ResourceURI)
			require.NoError(t, err)
			for _, q := range tt.args.queries {
				req.AddQuery(q)
			}
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode, resp.Code())
			if tt.wantsCode != coapCodes.Content {
				return
			}
			var data interface{}
			err = cbor.ReadFrom(resp.Body(), &data)
			require.NoError(t, err)
			if tt.emptyContent {
				require.Empty(t, data)
				return
			}
			require.NotEmpty(t, data)
		})
	}
}
