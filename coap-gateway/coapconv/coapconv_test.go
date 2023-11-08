package coapconv_test

import (
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/stretchr/testify/require"
)

func TestCoapConvOperationToCoapCode(t *testing.T) {
	type args struct {
		op coapconv.Operation
	}
	tests := []struct {
		name string
		args args
		want codes.Code
	}{
		{
			name: "Create Operation",
			args: args{op: coapconv.Create},
			want: codes.Created,
		},
		{
			name: "Retrieve Operation",
			args: args{op: coapconv.Retrieve},
			want: codes.Content,
		},
		{
			name: "Update Operation",
			args: args{op: coapconv.Update},
			want: codes.Changed,
		},
		{
			name: "Delete Operation",
			args: args{op: coapconv.Delete},
			want: codes.Deleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, coapconv.StatusToCoapCode(commands.Status_OK, tt.args.op))
		})
	}
}

func TestCoapConvStatusToCoapCode(t *testing.T) {
	type args struct {
		status commands.Status
	}
	tests := []struct {
		name string
		args args
		want codes.Code
	}{
		{
			name: "Created",
			args: args{status: commands.Status_CREATED},
			want: codes.Created,
		},
		{
			name: "Accepted",
			args: args{status: commands.Status_ACCEPTED},
			want: codes.Valid,
		},
		{
			name: "Not Modified",
			args: args{status: commands.Status_NOT_MODIFIED},
			want: codes.Valid,
		},
		{
			name: "Bad Request",
			args: args{status: commands.Status_BAD_REQUEST},
			want: codes.BadRequest,
		},
		{
			name: "Unauthorized",
			args: args{status: commands.Status_UNAUTHORIZED},
			want: codes.Unauthorized,
		},
		{
			name: "Forbidden",
			args: args{status: commands.Status_FORBIDDEN},
			want: codes.Forbidden,
		},
		{
			name: "Not Found",
			args: args{status: commands.Status_NOT_FOUND},
			want: codes.NotFound,
		},
		{
			name: "Service Unavailable",
			args: args{status: commands.Status_UNAVAILABLE},
			want: codes.ServiceUnavailable,
		},
		{
			name: "Not Implemented",
			args: args{status: commands.Status_NOT_IMPLEMENTED},
			want: codes.NotImplemented,
		},
		{
			name: "Not Allowed",
			args: args{status: commands.Status_METHOD_NOT_ALLOWED},
			want: codes.BadRequest, // TODO: what about codes.MethodNotAllowed ?
		},
		{
			name: "Error",
			args: args{status: commands.Status_ERROR},
			want: codes.BadRequest,
		},
		{
			name: "Canceled",
			args: args{status: commands.Status_CANCELED},
			want: codes.BadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, coapconv.StatusToCoapCode(tt.args.status, coapconv.Retrieve))
		})
	}
}

func TestCoapCodeToStatus(t *testing.T) {
	type args struct {
		code      codes.Code
		operation coapconv.Operation
	}
	tests := []struct {
		name string
		args args
		want commands.Status
	}{
		{
			name: "Coap Code Valid, Retrieve Operation",
			args: args{code: codes.Valid, operation: coapconv.Retrieve},
			want: commands.Status_NOT_MODIFIED,
		},
		{
			name: "Coap Code Valid, Create Operation",
			args: args{code: codes.Valid, operation: coapconv.Create},
			want: commands.Status_ACCEPTED,
		},
		{
			name: "Coap Code Changed",
			args: args{code: codes.Changed, operation: coapconv.Retrieve},
			want: commands.Status_OK,
		},
		{
			name: "Coap Code Content",
			args: args{code: codes.Content, operation: coapconv.Retrieve},
			want: commands.Status_OK,
		},
		{
			name: "Coap Code Deleted",
			args: args{code: codes.Deleted, operation: coapconv.Retrieve},
			want: commands.Status_OK,
		},
		{
			name: "Coap Code BadRequest",
			args: args{code: codes.BadRequest, operation: coapconv.Retrieve},
			want: commands.Status_BAD_REQUEST,
		},
		{
			name: "Coap Code Unauthorized",
			args: args{code: codes.Unauthorized, operation: coapconv.Retrieve},
			want: commands.Status_UNAUTHORIZED,
		},
		{
			name: "Coap Code Forbidden",
			args: args{code: codes.Forbidden, operation: coapconv.Retrieve},
			want: commands.Status_FORBIDDEN,
		},
		{
			name: "Coap Code NotFound",
			args: args{code: codes.NotFound, operation: coapconv.Retrieve},
			want: commands.Status_NOT_FOUND,
		},
		{
			name: "Coap Code ServiceUnavailable",
			args: args{code: codes.ServiceUnavailable, operation: coapconv.Retrieve},
			want: commands.Status_UNAVAILABLE,
		},
		{
			name: "Coap Code NotImplemented",
			args: args{code: codes.NotImplemented, operation: coapconv.Retrieve},
			want: commands.Status_NOT_IMPLEMENTED,
		},
		{
			name: "Coap Code MethodNotAllowed",
			args: args{code: codes.MethodNotAllowed, operation: coapconv.Retrieve},
			want: commands.Status_METHOD_NOT_ALLOWED,
		},
		{
			name: "Coap Code Created",
			args: args{code: codes.Created, operation: coapconv.Retrieve},
			want: commands.Status_CREATED,
		},
		{
			name: "Unknown Coap Code",
			args: args{code: 123, operation: coapconv.Retrieve},
			want: commands.Status_ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, coapconv.CoapCodeToStatus(tt.args.code, tt.args.operation))
		})
	}
}

func TestCoapConvMakeMediaType(t *testing.T) {
	type args struct {
		coapContentFormat int32
		contentType       string
	}
	tests := []struct {
		name    string
		args    args
		want    message.MediaType
		wantErr bool
	}{
		{
			name: "Valid coapContentFormat",
			args: args{coapContentFormat: int32(message.TextPlain), contentType: ""},
			want: message.TextPlain,
		},
		{
			name: "TextPlain",
			args: args{coapContentFormat: -1, contentType: message.TextPlain.String()},
			want: message.TextPlain,
		},
		{
			name: "AppJson",
			args: args{coapContentFormat: -1, contentType: message.AppJSON.String()},
			want: message.AppJSON,
		},
		{
			name: "AppCBOR",
			args: args{coapContentFormat: -1, contentType: message.AppCBOR.String()},
			want: message.AppCBOR,
		},
		{
			name: "AppOcfCbor",
			args: args{coapContentFormat: -1, contentType: message.AppOcfCbor.String()},
			want: message.AppOcfCbor,
		},
		{
			name:    "Unknown contentType",
			args:    args{coapContentFormat: -1, contentType: message.AppXML.String()},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mediaType, err := coapconv.MakeMediaType(tt.args.coapContentFormat, tt.args.contentType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, mediaType)
		})
	}
}
