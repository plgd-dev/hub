package message

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testError struct {
	isTemporary bool
	isTimeout   bool
	grpcStatus  status.Status
	oauthErr    *oauth2.RetrieveError
}

type oauth2RetrieveError struct {
	oauthErr *oauth2.RetrieveError
}

func (e *oauth2RetrieveError) Unwrap() error {
	return e.oauthErr
}

func (e *oauth2RetrieveError) Error() string {
	return "test oauth error"
}

func (e *testError) Error() string {
	return "test error"
}

func (e *testError) Temporary() bool {
	return e.isTemporary
}

func (e *testError) Timeout() bool {
	return e.isTimeout
}

func (e *testError) GRPCStatus() *status.Status {
	return &e.grpcStatus
}

func (e *testError) Unwrap() error {
	return e.oauthErr
}

func TestIsTempError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil",
			args: args{},
			want: false,
		},
		{
			name: "not temporary",
			args: args{err: fmt.Errorf("err: %w", &testError{grpcStatus: *status.Convert(status.Error(codes.Unauthenticated, "unauthenticated"))})},
			want: false,
		},
		{
			name: "any error as temporary",
			args: args{err: fmt.Errorf("any error")},
			want: true,
		},
		{
			name: "temporary",
			args: args{err: fmt.Errorf("err: %w", &testError{isTemporary: true})},
			want: true,
		},
		{
			name: "timeout",
			args: args{err: fmt.Errorf("err: %w", &testError{isTimeout: true})},
			want: true,
		},
		{
			name: "grpcTemporary",
			args: args{err: fmt.Errorf("err: %w", &testError{grpcStatus: *status.FromContextError(context.DeadlineExceeded)})},
			want: true,
		},
		{
			name: "oauth2Temporary",
			args: args{err: fmt.Errorf("err: %w", &oauth2RetrieveError{oauthErr: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusServiceUnavailable}}})},
			want: true,
		},
		{
			name: "oauth2NotTemporary",
			args: args{err: fmt.Errorf("err: %w", &oauth2RetrieveError{oauthErr: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}}})},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTempError(tt.args.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
