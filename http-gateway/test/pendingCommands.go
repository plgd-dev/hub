package test

import (
	"strconv"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
)

func ToCommandsFilter(s []pb.GetPendingCommandsRequest_Command) []string {
	sf := make([]string, 0, len(s))
	for _, v := range s {
		sf = append(sf, strconv.FormatInt(int64(v), 10))
	}
	return sf
}
