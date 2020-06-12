package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/uri"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientResetHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
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
				path:    uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
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
				path:    uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
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
				require.NoError(t, err)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
				defer cancel()
				msg := pool.AcquireMessage(ctx)
				msg.SetCode(tt.args.code)
				msg.SetToken(tt.args.token)
				msg.SetPath(tt.args.path)
				msg.SetObserve(tt.args.observe)
				resp, err := co.Do(msg)
				require.NoError(t, err)
				assert.Equal(t, tt.wantsCode, resp.Code())
			}
			time.Sleep(time.Second) // to avoid reorder test case
		})
	}
}
