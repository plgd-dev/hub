package service

import (
	"context"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
)

func (r *RequestHandler) GetHubConfiguration(context.Context, *pb.HubConfigurationRequest) (*pb.HubConfigurationResponse, error) {
	return r.publicConfiguration.ToProto(), nil
}
