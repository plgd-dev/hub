package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

type grpcErr interface {
	GRPCStatus() *status.Status
}

// ForwardFromError tries to unwrap err as GRPCStatus() and forward original code and details.
func ForwardFromError(code codes.Code, err error) error {
	return ForwardErrorf(code, "%v", err)
}

// ForwardErrorf tries to unwrap args as error with GRPCStatus() and forward original code and details.
func ForwardErrorf(code codes.Code, formatter string, args ...interface{}) error {
	var details []*anypb.Any
	for _, a := range args {
		var grpcErr grpcErr
		if err, ok := a.(error); ok {
			if errors.As(err, &grpcErr) {
				s := grpcErr.GRPCStatus()
				code = s.Code()
				details = s.Proto().GetDetails()
				break
			}
		}
	}
	sProto := status.Newf(code, formatter, args...).Proto()
	sProto.Details = details
	return status.FromProto(sProto).Err()
}
