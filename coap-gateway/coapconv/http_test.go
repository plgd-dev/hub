package coapconv

import (
	"net/http"
	"testing"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/stretchr/testify/assert"
)

func TestToHTTPCode(t *testing.T) {
	type args struct {
		code coapCodes.Code
		def  int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "coapCodes.Empty", args: args{code: coapCodes.Empty, def: 9999}, want: 9999},
		{name: "coapCodes.Created", args: args{code: coapCodes.Created, def: 9999}, want: http.StatusCreated},
		{name: "coapCodes.Deleted", args: args{code: coapCodes.Deleted, def: 9999}, want: http.StatusOK},
		{name: "coapCodes.Valid", args: args{code: coapCodes.Valid, def: 9999}, want: http.StatusAccepted},
		{name: "coapCodes.Changed", args: args{code: coapCodes.Changed, def: 9999}, want: http.StatusOK},
		{name: "coapCodes.Content", args: args{code: coapCodes.Content, def: 9999}, want: http.StatusOK},
		{name: "coapCodes.Continue", args: args{code: coapCodes.Continue, def: 9999}, want: http.StatusContinue},
		{name: "coapCodes.BadRequest", args: args{code: coapCodes.BadRequest, def: 9999}, want: http.StatusBadRequest},
		{name: "coapCodes.Unauthorized", args: args{code: coapCodes.Unauthorized, def: 9999}, want: http.StatusUnauthorized},
		{name: "coapCodes.BadOption", args: args{code: coapCodes.BadOption, def: 9999}, want: http.StatusBadRequest},
		{name: "coapCodes.Forbidden", args: args{code: coapCodes.Forbidden, def: 9999}, want: http.StatusForbidden},
		{name: "coapCodes.NotFound", args: args{code: coapCodes.NotFound, def: 9999}, want: http.StatusNotFound},
		{name: "coapCodes.MethodNotAllowed", args: args{code: coapCodes.MethodNotAllowed, def: 9999}, want: http.StatusMethodNotAllowed},
		{name: "coapCodes.NotAcceptable", args: args{code: coapCodes.NotAcceptable, def: 9999}, want: http.StatusNotAcceptable},
		{name: "coapCodes.RequestEntityIncomplete", args: args{code: coapCodes.RequestEntityIncomplete, def: 9999}, want: http.StatusBadRequest},
		{name: "coapCodes.PreconditionFailed", args: args{code: coapCodes.PreconditionFailed, def: 9999}, want: http.StatusPreconditionFailed},
		{name: "coapCodes.RequestEntityTooLarge", args: args{code: coapCodes.RequestEntityTooLarge, def: 9999}, want: http.StatusRequestEntityTooLarge},
		{name: "coapCodes.UnsupportedMediaType", args: args{code: coapCodes.UnsupportedMediaType, def: 9999}, want: http.StatusUnsupportedMediaType},
		{name: "coapCodes.InternalServerError", args: args{code: coapCodes.InternalServerError, def: 9999}, want: http.StatusInternalServerError},
		{name: "coapCodes.NotImplemented", args: args{code: coapCodes.NotImplemented, def: 9999}, want: http.StatusNotImplemented},
		{name: "coapCodes.BadGateway", args: args{code: coapCodes.BadGateway, def: 9999}, want: http.StatusBadGateway},
		{name: "coapCodes.ServiceUnavailable", args: args{code: coapCodes.ServiceUnavailable, def: 9999}, want: http.StatusServiceUnavailable},
		{name: "coapCodes.GatewayTimeout", args: args{code: coapCodes.GatewayTimeout, def: 9999}, want: http.StatusGatewayTimeout},
		{name: "coapCodes.ProxyingNotSupported", args: args{code: coapCodes.ProxyingNotSupported, def: 9999}, want: 9999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToHTTPCode(tt.args.code, tt.args.def)
			assert.Equal(t, tt.want, got)
		})
	}
}
