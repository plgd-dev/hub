package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	extCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	"github.com/plgd-dev/cloud/grpc-gateway/pb/errdetails"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/go-coap/v2/message"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func statusToGrpcStatus(status commands.Status) codes.Code {
	return pb.RAStatus2Status(status).ToGrpcCode()
}

func eventContentToContent(s commands.Status, c *commands.Content) (*pb.Content, error) {
	var content *pb.Content
	if c != nil {
		contentType := c.GetContentType()
		if contentType == "" && c.GetCoapContentFormat() >= 0 {
			contentType = message.MediaType(c.GetCoapContentFormat()).String()
		}
		content = &pb.Content{
			Data:        c.GetData(),
			ContentType: contentType,
		}
	}
	statusCode := pb.RAStatus2Status(s).ToGrpcCode()
	switch statusCode {
	case codes.OK:
	case extCodes.Accepted:
	case extCodes.Created:
	default:
		s := status.New(statusCode, "response from device")
		if content != nil {
			newS, err := s.WithDetails(&errdetails.DeviceError{
				Content: &errdetails.Content{
					Data:        content.GetData(),
					ContentType: content.GetContentType(),
				},
			})
			if err == nil {
				s = newS
			}
		}
		return nil, s.Err()
	}
	return content, nil
}

func toResponse(processed *events.ResourceUpdated) (*pb.UpdateResourceResponse, error) {
	content, err := eventContentToContent(processed.GetStatus(), processed.GetContent())
	if err != nil {
		return nil, err
	}
	return &pb.UpdateResourceResponse{
		Content: content,
		Status:  pb.RAStatus2Status(processed.GetStatus()),
	}, nil
}

func (r *RequestHandler) waitForUpdateContentResponse(ctx context.Context, deviceID, resourceID string, notify <-chan *events.ResourceUpdated, onTimeout func(ctx context.Context, destDeviceId, resourceID string, notify <-chan *events.ResourceUpdated) (*pb.UpdateResourceResponse, error)) (*pb.UpdateResourceResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeoutForRequests)
	defer cancel()
	select {
	case processed := <-notify:
		return toResponse(processed)
	case <-timeoutCtx.Done():
		return onTimeout(ctx, deviceID, resourceID, notify)
	}
}

func (r *RequestHandler) UpdateResourcesValues(ctx context.Context, req *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	accessToken, err := kitNetGrpc.TokenFromMD(ctx)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot update resource: %v", err))
	}
	if req.ResourceId == nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot update resource: invalid ResourceId"))
	}

	errorMsg := fmt.Sprintf("cannot update resource /%v", req.GetResourceId()) + ": %v"

	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, errorMsg, err))
	}

	correlationID := correlationIDUUID.String()
	notify := r.updateNotificationContainer.Add(correlationID)
	defer r.updateNotificationContainer.Remove(correlationID)

	loaded, err := r.resourceProjection.Register(ctx, req.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.NotFound, errorMsg, fmt.Errorf("cannot register device to projection: %w", err)))
	}
	defer r.resourceProjection.Unregister(req.GetResourceId().GetDeviceId())

	if !loaded {
		if len(r.resourceProjection.Models(req.GetResourceId())) == 0 {
			err = r.resourceProjection.ForceUpdate(ctx, req.GetResourceId())
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
	raReq := commands.UpdateResourceRequest{
		ResourceId:        req.GetResourceId(),
		CorrelationId:     correlationID,
		ResourceInterface: req.GetResourceInterface(),
		Content: &commands.Content{
			Data:              req.GetContent().GetData(),
			ContentType:       req.GetContent().GetContentType(),
			CoapContentFormat: -1,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
			Sequence:     seq,
		},
	}

	_, err = r.resourceAggregateClient.UpdateResource(kitNetGrpc.CtxWithToken(ctx, accessToken), &raReq)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, errorMsg, err))
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeoutForRequests)
	defer cancel()
	select {
	case processed := <-notify:
		return toResponse(processed)
	case <-timeoutCtx.Done():
	}

	return nil, logAndReturnError(status.Errorf(codes.DeadlineExceeded, errorMsg, fmt.Errorf("timeout")))
}
