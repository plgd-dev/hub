package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapStatus "github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SdkError struct {
	errorCode codes.Code
}

func (e SdkError) GetCode() codes.Code {
	return e.errorCode
}

func (e SdkError) Error() string {
	return fmt.Sprintf("Status code %v", e.errorCode)
}

func TestErrToStatus(t *testing.T) {
	forbidden := pool.NewMessage(context.Background())
	forbidden.SetCode(coapCodes.Forbidden)
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "ol", args: args{err: nil}, want: http.StatusOK},
		{name: "grpc", args: args{err: status.Error(codes.PermissionDenied, "grpc error")}, want: http.StatusForbidden},
		{
			name: "coap",
			args: args{
				err: coapStatus.Error(forbidden, fmt.Errorf("coap error")),
			},
			want: http.StatusForbidden,
		},
		{name: "grpc", args: args{err: fmt.Errorf("unknown error")}, want: http.StatusInternalServerError},
		{name: "sdkError", args: args{err: SdkError{errorCode: codes.PermissionDenied}}, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrToStatus(tt.args.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
