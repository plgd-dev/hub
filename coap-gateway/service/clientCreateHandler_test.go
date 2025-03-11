//go:build test
// +build test

package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestClientCreateHandler(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.TLS.Enabled = new(bool)
	coapgwCfg.Log.DumpBody = true
	coapgwCfg.Log.Level = zap.DebugLevel
	shutdown := setUp(t, testService.WithCOAPGWConfig(coapgwCfg))
	defer shutdown()
	log.Setup(coapgwCfg.Log)
	co := testCoapDial(t, "", false, true, time.Now().Add(time.Minute))
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	type args struct {
		href          string
		payload       []byte
		contentFormat message.MediaType
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			var body io.ReadSeeker
			if len(tt.args.payload) > 0 {
				body = bytes.NewReader(tt.args.payload)
			}
			req, err := co.NewPostRequest(ctx, tt.args.href, tt.args.contentFormat, body)
			require.NoError(t, err)
			req.SetOptionString(message.URIQuery, uri.InterfaceQueryKeyPrefix+interfaces.OC_IF_CREATE)
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
		})
	}
}
