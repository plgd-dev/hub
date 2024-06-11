package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
)

func (r *RequestHandler) DeleteResource(ctx context.Context, req *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	var resp pb.DeleteResourceResponse
	return &resp, handleResourceRequest(
		ctx,
		req,
		"delete",
		r.resourceAggregateClient.SyncDeleteResource,
		r.resourceAggregateClient.DeleteResource,
		&resp,
	)
}
