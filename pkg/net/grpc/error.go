package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
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

// ErrToGrpcStatus converts error to grpc status.
func ErrToGrpcStatus(a interface{}) *status.Status {
	var gErr grpcErr
	switch val := a.(type) {
	case *multierror.Error:
		for _, err := range val.Errors {
			if errors.As(err, &gErr) {
				return gErr.GRPCStatus()
			}
		}
	case []error:
		for _, err := range val {
			if errors.As(err, &gErr) {
				return gErr.GRPCStatus()
			}
		}
	case error:
		if errors.As(val, &gErr) {
			return gErr.GRPCStatus()
		}
	}

	return nil
}

// ForwardErrorf tries to unwrap args as error with GRPCStatus() and forward original code and details.
func ForwardErrorf(code codes.Code, formatter string, args ...interface{}) error {
	var details []*anypb.Any
	for _, a := range args {
		if s := ErrToGrpcStatus(a); s != nil {
			code = s.Code()
			details = s.Proto().GetDetails()
			break
		}
	}
	err := fmt.Errorf(formatter, args...)
	sProto := status.New(code, err.Error()).Proto()
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
