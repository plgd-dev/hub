package service

import (
	"context"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
)

func (r *RequestHandler) GetCloudConfiguration(context.Context, *pb.CloudConfigurationRequest) (*pb.CloudConfigurationResponse, error) {
	return r.publicConfiguration.ToProto(), nil
}
