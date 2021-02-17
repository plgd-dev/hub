package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) DeleteResource(ctx context.Context, req *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	accessToken, err := kitNetGrpc.TokenFromMD(ctx)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot delete resource: %v", err))
	}
	if req.ResourceId == nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete resource: invalid ResourceId"))
	}
	resourceID := req.GetResourceId()
	errorMsg := fmt.Sprintf("cannot delete resource /%v", resourceID) + ": %v"

	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, errorMsg, err))
	}

	correlationID := correlationIDUUID.String()
	notify := r.deleteNotificationContainer.Add(correlationID)
	defer r.updateNotificationContainer.Remove(correlationID)

	loaded, err := r.resourceProjection.Register(ctx, resourceID.GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.NotFound, errorMsg, fmt.Errorf("cannot register device to projection: %w", err)))
	}
	defer r.resourceProjection.Unregister(resourceID.GetDeviceId())

	if !loaded {
		if len(r.resourceProjection.Models(resourceID)) == 0 {
			err = r.resourceProjection.ForceUpdate(ctx, resourceID)
			if err != nil {
				return nil, logAndReturnError(status.Errorf(codes.NotFound, errorMsg, err))
			}
		}
	}

	connectionID := r.fqdn
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	seq := atomic.AddUint64(&r.seqNum, 1)
	raReq := &commands.DeleteResourceRequest{
		ResourceId:    resourceID,
		CorrelationId: correlationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
			Sequence:     seq,
		},
	}

	_, err = r.resourceAggregateClient.DeleteResource(kitNetGrpc.CtxWithToken(ctx, accessToken), raReq)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, errorMsg, err))
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeoutForRequests)
	defer cancel()
	select {
	case processed := <-notify:
		content, err := eventContentToContent(processed.GetStatus(), processed.GetContent())
		if err != nil {
			return nil, err
		}
		return &pb.DeleteResourceResponse{
			Content: content,
			Status:  pb.RAStatus2Status(processed.GetStatus()),
		}, nil
	case <-timeoutCtx.Done():
	}

	return nil, logAndReturnError(status.Errorf(codes.DeadlineExceeded, errorMsg, fmt.Errorf("timeout")))
}
