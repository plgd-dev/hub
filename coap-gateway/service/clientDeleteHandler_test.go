package service_test

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/v2/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/v2/test/config"

	"github.com/plgd-dev/go-coap/v2/tcp"

	"github.com/plgd-dev/cloud/v2/pkg/log"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientDeleteHandler(t *testing.T) {
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
		path string
	}
	tests := []struct {
		name         string
		args         args
		wantsCode    coapCodes.Code
		wantsContent []byte
	}{
		{
			name: "invalid href",
			args: args{
				path: uri.ResourceRoute + TestAResourceHref,
			},
			wantsCode: coapCodes.BadRequest,
		},
		{
			name: "forbidden",
			args: args{
				path: uri.ResourceRoute + "/dev0/res0",
			},
			wantsCode: coapCodes.Forbidden,
		},
		{
			name: "not found",
			args: args{
				path: uri.ResourceRoute + "/" + CertIdentity + "/notFound",
			},
			wantsCode: coapCodes.NotFound,
		},
		{
			name: "found",
			args: args{
				path: uri.ResourceRoute + "/" + CertIdentity + TestAResourceHref,
			},
			wantsCode:    coapCodes.Deleted,
			wantsContent: []byte("hello world"),
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
			req, err := tcp.NewDeleteRequest(ctx, tt.args.path)
			require.NoError(t, err)
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
			if tt.wantsContent != nil {
				require.NotEmpty(t, resp.Body())
				b, err := ioutil.ReadAll(resp.Body())
				require.NoError(t, err)
				assert.Equal(t, tt.wantsContent, b)
			}
		})
	}
}
