package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	authTest "github.com/go-ocf/cloud/authorization/test"
	coapgwTest "github.com/go-ocf/cloud/coap-gateway/test"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	raTest "github.com/go-ocf/cloud/resource-aggregate/test"
	rdTest "github.com/go-ocf/cloud/resource-directory/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientUpdateHandler(t *testing.T) {
	defer authTest.SetUp(t)
	defer raTest.SetUp(t)
	defer rdTest.SetUp(t)
	defer coapgwTest.SetUp(t, true)

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

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
			name: "invalid href",
			args: args{
				href:          uri.ResourceRoute + TestAResourceHref,
				contentFormat: message.TextPlain,
				payload:       []byte{},
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name: "not found",
			args: args{
				href:          uri.ResourceRoute + "/a/b",
				contentFormat: message.TextPlain,
				payload:       []byte{},
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
			wantsCode: coapCodes.Changed,
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
			resp, err := co.Post(ctx, tt.args.href, tt.args.contentFormat, body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode, resp.Code())
		})
	}
}
