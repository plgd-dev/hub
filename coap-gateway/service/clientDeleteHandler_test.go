//go:build test
// +build test

package service_test

import (
	"context"
	"io"
	"testing"
	"time"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientDeleteHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
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
		wantsContent []byte
		wantsCode    coapCodes.Code
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
			defer cancel()
			req, err := co.NewDeleteRequest(ctx, tt.args.path)
			require.NoError(t, err)
			resp, err := co.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantsCode.String(), resp.Code().String())
			if tt.wantsContent != nil {
				require.NotEmpty(t, resp.Body())
				b, err := io.ReadAll(resp.Body())
				require.NoError(t, err)
				assert.Equal(t, tt.wantsContent, b)
			}
		})
	}
}
