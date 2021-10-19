package test

import (
	"strconv"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
)

func ToCommandsFilter(s []pb.GetPendingCommandsRequest_Command) []string {
	var sf []string
	for _, v := range s {
		sf = append(sf, strconv.FormatInt(int64(v), 10))
	}
	return sf
}
