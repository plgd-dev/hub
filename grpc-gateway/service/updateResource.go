package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
)

func (r *RequestHandler) UpdateResource(ctx context.Context, req *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	var resp pb.UpdateResourceResponse
	return &resp, handleResourceRequest(
		ctx,
		req,
		"update",
		r.resourceAggregateClient.SyncUpdateResource,
		r.resourceAggregateClient.UpdateResource,
		&resp,
	)
}
