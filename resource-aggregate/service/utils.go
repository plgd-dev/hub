package service

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func checkTimeToLive(timeToLive int64) error {
	if timeToLive != 0 && timeToLive < int64(time.Millisecond*100) {
		return status.Errorf(codes.InvalidArgument, "timeToLive(`%v`) is less than 100ms", time.Duration(timeToLive))
	}
	return nil
}
