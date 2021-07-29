package log

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcErr interface {
	GRPCStatus() *status.Status
}

func LogAndReturnError(err error) error {
	if err == nil {
		return err
	}
	if errors.Is(err, io.EOF) {
		Debugf("%v", err)
		return err
	}
	var grpcErr grpcErr
	if errors.As(err, &grpcErr) {
		if grpcErr.GRPCStatus().Code() == codes.Canceled {
			Debugf("%v", err)
			return err
		}
	}
	if errors.Is(err, context.Canceled) {
		Debugf("%v", err)
		return err
	}
	Error(err)
	return err
}
