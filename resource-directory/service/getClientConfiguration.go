package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

func (r *RequestHandler) GetClientConfiguration(context.Context, *pb.ClientConfigurationRequest) (*pb.ClientConfigurationResponse, error) {
	return r.publicConfiguration.ToProto(), nil
}
