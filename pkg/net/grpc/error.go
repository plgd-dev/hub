package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

type grpcErr interface {
	GRPCStatus() *status.Status
}

func IsContextCanceled(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	var gErr grpcErr
	if ok := errors.As(err, &gErr); ok {
		return gErr.GRPCStatus().Code() == codes.Canceled
	}
	return false
}

func IsContextDeadlineExceeded(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var gErr grpcErr
	if ok := errors.As(err, &gErr); ok {
		return gErr.GRPCStatus().Code() == codes.DeadlineExceeded
	}
	return false
}

// ForwardFromError tries to unwrap err as GRPCStatus() and forward original code and details.
func ForwardFromError(code codes.Code, err error) error {
	return ForwardErrorf(code, "%v", err)
}

// ForwardErrorf tries to unwrap args as error with GRPCStatus() and forward original code and details.
func ForwardErrorf(code codes.Code, formatter string, args ...interface{}) error {
	var details []*anypb.Any
	for _, a := range args {
		var gErr grpcErr
		if err, ok := a.(error); ok {
			if errors.As(err, &gErr) {
				s := gErr.GRPCStatus()
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

func ErrToStatus(err error) *status.Status {
	var gErr grpcErr
	if errors.As(err, &gErr) {
		return gErr.GRPCStatus()
	}
	return status.Convert(err)
}
