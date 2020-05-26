package service

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

func (r *RequestHandler) GetClientConfiguration(context.Context, *pb.ClientConfigurationRequest) (*pb.ClientConfigurationResponse, error) {
	return &r.clientConfiguration, nil
}
