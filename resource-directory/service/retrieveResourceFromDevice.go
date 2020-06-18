package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) RetrieveResourceFromDevice(ctx context.Context, req *pb.RetrieveResourceFromDeviceRequest) (*pb.RetrieveResourceFromDeviceResponse, error) {
	accessToken, err := kitNetGrpc.TokenFromMD(ctx)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot retrieve resource from device: %v", err))
	}
	if req.ResourceId == nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot retrieve resource from device: invalid ResourceId"))
	}
	deviceID := req.GetResourceId().GetDeviceId()
	href := req.GetResourceId().GetHref()
	errorMsg := fmt.Sprintf("cannot retrieve resource from device /%v%v", deviceID, href) + ": %v"

	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, errorMsg, err))
	}

	correlationID := correlationIDUUID.String()
	resourceID := req.ResourceId.ID()
	notify := r.retrieveNotificationContainer.Add(correlationID)
	defer r.retrieveNotificationContainer.Remove(correlationID)

	loaded, err := r.resourceProjection.Register(ctx, deviceID)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.NotFound, errorMsg, fmt.Errorf("cannot register device to projection: %w", err)))
	}
	defer r.resourceProjection.Unregister(deviceID)

	if !loaded {
		if len(r.resourceProjection.Models(deviceID, resourceID)) == 0 {
			err = r.resourceProjection.ForceUpdate(ctx, deviceID, resourceID)
			if err != nil {
				return nil, logAndReturnError(status.Errorf(codes.NotFound, errorMsg, err))
			}
		}
	}

	connectionID := "grpc-gateway"
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	seq := atomic.AddUint64(&r.seqNum, 1)
	raReq := pbRA.RetrieveResourceRequest{
		ResourceId:        resourceID,
		ResourceInterface: req.GetResourceInterface(),
		CorrelationId:     correlationID,
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: connectionID,
			Sequence:     seq,
		},
	}

	_, err = r.resourceAggregateClient.RetrieveResource(kitNetGrpc.CtxWithToken(ctx, accessToken), &raReq)
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
		return &pb.RetrieveResourceFromDeviceResponse{
			Content: content,
		}, nil
	case <-timeoutCtx.Done():
	}

	return nil, logAndReturnError(status.Errorf(codes.DeadlineExceeded, errorMsg, fmt.Errorf("timeout")))
}
