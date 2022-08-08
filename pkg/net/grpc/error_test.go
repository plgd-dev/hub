package grpc_test

import (
	"fmt"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestForwardErrorf(t *testing.T) {
	type args struct {
		code      codes.Code
		formatter string
		args      []interface{}
	}
	tests := []struct {
		name        string
		args        args
		wantCode    codes.Code
		wantMessage string
		wantDetails []interface{}
	}{
		{
			name: "basic",
			args: args{
				code:      codes.Internal,
				formatter: "abc: %v",
				args:      []interface{}{1},
			},
			wantCode:    codes.Internal,
			wantDetails: []interface{}{},
			wantMessage: "abc: 1",
		},
		{
			name: "with error",
			args: args{
				code:      codes.Internal,
				formatter: "abc: %v",
				args:      []interface{}{fmt.Errorf("def: %v", 1)},
			},
			wantCode:    codes.Internal,
			wantDetails: []interface{}{},
			wantMessage: "abc: def: 1",
		},
		{
			name: "with composite error",
			args: args{
				code:      codes.OK,
				formatter: "abc: %v",
				args:      []interface{}{fmt.Errorf("def: %w", status.Errorf(codes.Internal, "ghi %v", 1))},
			},
			wantCode:    codes.Internal,
			wantDetails: []interface{}{},
			wantMessage: "abc: def: rpc error: code = Internal desc = ghi 1",
		},
		{
			name: "with details",
			args: args{
				code:      codes.OK,
				formatter: "abc: %v",
				args: []interface{}{func() error {
					s := status.Newf(codes.Internal, "ghi %v", 1)
					s, err := s.WithDetails(&errdetails.DebugInfo{Detail: "dbg"})
					require.NoError(t, err)
					return s.Err()
				}()},
			},
			wantCode: codes.Internal,
			wantDetails: []interface{}{
				&errdetails.DebugInfo{Detail: "dbg"},
			},
			wantMessage: "abc: rpc error: code = Internal desc = ghi 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := grpc.ForwardErrorf(tt.args.code, tt.args.formatter, tt.args.args...)
			require.Error(t, err)
			s, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, s.Code())
			test.CheckProtobufs(t, tt.wantDetails, s.Details(), test.AssertToCheckFunc(assert.Equal))
			assert.Equal(t, tt.wantMessage, s.Message())
		})
	}
}
