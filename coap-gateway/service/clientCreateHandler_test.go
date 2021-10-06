package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	coapgwTest "github.com/plgd-dev/hub/coap-gateway/test"
	"github.com/plgd-dev/hub/coap-gateway/uri"
	"github.com/plgd-dev/hub/pkg/ocf"
	testCfg "github.com/plgd-dev/hub/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientCreateHandler(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.TLS.Enabled = false
	shutdown := setUp(t, coapgwCfg)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "", true)
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	type args struct {
		href          string
		contentFormat message.MediaType
		payload       []byte
	}
	tests := []struct {
		name      string
		args      args
		wantsCode coapCodes.Code
	}{
		{
			name: "forbidden",
			args: args{
				href:          uri.ResourceRoute + "/a/b",
				contentFormat: message.TextPlain,
				payload:       []byte{},
			},
			wantsCode: coapCodes.Forbidden,
		},
		{
			name: "not found",
			args: args{
				href:          uri.ResourceRoute + "/" + CertIdentity + "/notFound",
				contentFormat: message.TextPlain,
				payload:       []byte("data"),
			},
			wantsCode: coapCodes.NotFound,
		},
		{
			name: "valid",
			args: args{
				href:          uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
				contentFormat: message.TextPlain,
				payload:       []byte("data"),
			},
			wantsCode: coapCodes.Created,
		},
	}

	testPrepareDevice(t, co)
	time.Sleep(time.Second) // for publish content of device resources

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout*3600)
			defer cancel()
			var body io.ReadSeeker
			if len(tt.args.payload) > 0 {
				body = bytes.NewReader(tt.args.payload)
			}
			req, err := tcp.NewPostRequest(ctx, tt.args.href, tt.args.contentFormat, body)
			require.NoError(t, err)
			req.SetOptionString(message.URIQuery, "if="+ocf.OC_IF_CREATE)
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
		})
	}
}
