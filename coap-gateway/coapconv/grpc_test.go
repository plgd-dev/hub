package coapconv

import (
	"testing"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestToGrpcCode(t *testing.T) {
	type args struct {
		code coapCodes.Code
		def  codes.Code
	}
	tests := []struct {
		name string
		args args
		want codes.Code
	}{
		{name: "coapCodes.Empty", args: args{code: coapCodes.Empty, def: codes.DataLoss}, want: codes.Unknown},
		{name: "coapCodes.Created", args: args{code: coapCodes.Created, def: codes.DataLoss}, want: codes.OK},
		{name: "coapCodes.Deleted", args: args{code: coapCodes.Deleted, def: codes.DataLoss}, want: codes.OK},
		{name: "coapCodes.Valid", args: args{code: coapCodes.Valid, def: codes.DataLoss}, want: codes.OK},
		{name: "coapCodes.Changed", args: args{code: coapCodes.Changed, def: codes.DataLoss}, want: codes.OK},
		{name: "coapCodes.Content", args: args{code: coapCodes.Content, def: codes.DataLoss}, want: codes.OK},
		{name: "coapCodes.Continue", args: args{code: coapCodes.Continue, def: codes.DataLoss}, want: codes.OK},
		{name: "coapCodes.BadRequest", args: args{code: coapCodes.BadRequest, def: codes.DataLoss}, want: codes.InvalidArgument},
		{name: "coapCodes.Unauthorized", args: args{code: coapCodes.Unauthorized, def: codes.DataLoss}, want: codes.Unauthenticated},
		{name: "coapCodes.BadOption", args: args{code: coapCodes.BadOption, def: codes.DataLoss}, want: codes.InvalidArgument},
		{name: "coapCodes.Forbidden", args: args{code: coapCodes.Forbidden, def: codes.DataLoss}, want: codes.PermissionDenied},
		{name: "coapCodes.NotFound", args: args{code: coapCodes.NotFound, def: codes.DataLoss}, want: codes.NotFound},
		{name: "coapCodes.MethodNotAllowed", args: args{code: coapCodes.MethodNotAllowed, def: codes.DataLoss}, want: codes.PermissionDenied},
		{name: "coapCodes.NotAcceptable", args: args{code: coapCodes.NotAcceptable, def: codes.DataLoss}, want: codes.InvalidArgument},
		{name: "coapCodes.RequestEntityIncomplete", args: args{code: coapCodes.RequestEntityIncomplete, def: codes.DataLoss}, want: codes.InvalidArgument},
		{name: "coapCodes.PreconditionFailed", args: args{code: coapCodes.PreconditionFailed, def: codes.DataLoss}, want: codes.FailedPrecondition},
		{name: "coapCodes.RequestEntityTooLarge", args: args{code: coapCodes.RequestEntityTooLarge, def: codes.DataLoss}, want: codes.OutOfRange},
		{name: "coapCodes.UnsupportedMediaType", args: args{code: coapCodes.UnsupportedMediaType, def: codes.DataLoss}, want: codes.InvalidArgument},
		{name: "coapCodes.InternalServerError", args: args{code: coapCodes.InternalServerError, def: codes.DataLoss}, want: codes.Internal},
		{name: "coapCodes.NotImplemented", args: args{code: coapCodes.NotImplemented, def: codes.DataLoss}, want: codes.Unimplemented},
		{name: "coapCodes.BadGateway", args: args{code: coapCodes.BadGateway, def: codes.DataLoss}, want: codes.Unavailable},
		{name: "coapCodes.ServiceUnavailable", args: args{code: coapCodes.ServiceUnavailable, def: codes.DataLoss}, want: codes.Unavailable},
		{name: "coapCodes.GatewayTimeout", args: args{code: coapCodes.GatewayTimeout, def: codes.DataLoss}, want: codes.Unavailable},
		{name: "coapCodes.ProxyingNotSupported", args: args{code: coapCodes.ProxyingNotSupported, def: codes.DataLoss}, want: codes.Unimplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToGrpcCode(tt.args.code, tt.args.def)
			assert.Equal(t, tt.want, got)
		})
	}
}
