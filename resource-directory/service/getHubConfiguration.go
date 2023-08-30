package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
)

func (r *RequestHandler) GetHubConfiguration(context.Context, *pb.HubConfigurationRequest) (*pb.HubConfigurationResponse, error) {
	return r.publicConfiguration.ToProto(r.hubID), nil
}
