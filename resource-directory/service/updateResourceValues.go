package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	extCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	"github.com/plgd-dev/cloud/grpc-gateway/pb/errdetails"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func statusToGrpcStatus(status pbRA.Status) codes.Code {
	switch status {
	case pbRA.Status_UNKNOWN:
		return codes.Unknown
	case pbRA.Status_OK:
		return codes.OK
	case pbRA.Status_BAD_REQUEST:
		return codes.InvalidArgument
	case pbRA.Status_UNAUTHORIZED:
		return codes.Unauthenticated
	case pbRA.Status_FORBIDDEN:
		return codes.PermissionDenied
	case pbRA.Status_NOT_FOUND:
		return codes.NotFound
	case pbRA.Status_UNAVAILABLE:
		return codes.Unavailable
	case pbRA.Status_NOT_IMPLEMENTED:
		return codes.Unimplemented
	case pbRA.Status_ACCEPTED:
		return extCodes.Accepted
	}
	return extCodes.InvalidCode
}

func eventContentToContent(s pbRA.Status, c *pbRA.Content) (*pb.Content, error) {
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
	statusCode := statusToGrpcStatus(s)
	switch statusCode {
	case codes.OK:
	case extCodes.Accepted:
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

func toResponse(processed raEvents.ResourceUpdated) (*pb.UpdateResourceValuesResponse, error) {
	content, err := eventContentToContent(processed.GetStatus(), processed.GetContent())
	if err != nil {
		return nil, err
	}
	return &pb.UpdateResourceValuesResponse{
		Content: content,
		Status:  pb.RAStatus2Status(processed.GetStatus()),
	}, nil
}

func (r *RequestHandler) waitForUpdateContentResponse(ctx context.Context, deviceID, resourceID string, notify <-chan raEvents.ResourceUpdated, onTimeout func(ctx context.Context, destDeviceId, resourceID string, notify <-chan raEvents.ResourceUpdated) (*pb.UpdateResourceValuesResponse, error)) (*pb.UpdateResourceValuesResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeoutForRequests)
	defer cancel()
	select {
	case processed := <-notify:
		return toResponse(processed)
	case <-timeoutCtx.Done():
		return onTimeout(ctx, deviceID, resourceID, notify)
	}
}

func (r *RequestHandler) UpdateResourcesValues(ctx context.Context, req *pb.UpdateResourceValuesRequest) (*pb.UpdateResourceValuesResponse, error) {
	accessToken, err := kitNetGrpc.TokenFromMD(ctx)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot update resource: %v", err))
	}
	if req.ResourceId == nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot update resource: invalid ResourceId"))
	}
	deviceID := req.GetResourceId().GetDeviceId()
	href := req.GetResourceId().GetHref()
	errorMsg := fmt.Sprintf("cannot update resource /%v%v", deviceID, href) + ": %v"

	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, errorMsg, err))
	}

	correlationID := correlationIDUUID.String()
	resourceID := req.ResourceId.ID()
	notify := r.updateNotificationContainer.Add(correlationID)
	defer r.updateNotificationContainer.Remove(correlationID)

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

	connectionID := r.fqdn
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	seq := atomic.AddUint64(&r.seqNum, 1)
	raReq := pbRA.UpdateResourceRequest{
		ResourceId:        resourceID,
		CorrelationId:     correlationID,
		ResourceInterface: req.GetResourceInterface(),
		Content: &pbRA.Content{
			Data:              req.GetContent().GetData(),
			ContentType:       req.GetContent().GetContentType(),
			CoapContentFormat: -1,
		},
		CommandMetadata: &pbCQRS.CommandMetadata{
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
