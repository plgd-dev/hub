package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func resourceError(action string, err error) error {
	return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot %s resource: %v", action, err)
}

type reqRACommand interface {
	*commands.CreateResourceRequest | *commands.DeleteResourceRequest | *commands.UpdateResourceRequest
}

type respCommand[Event any] interface {
	SetData(v Event)
}

type reqCommand[v reqRACommand] interface {
	ToRACommand(ctx context.Context) (v, error)
	GetAsync() bool
}

type (
	syncFunc[Req reqRACommand, Res commands.EventContent] func(ctx context.Context, owner string, req Req) (Res, error)
	asyncFunc[Req reqRACommand, Res any]                  func(ctx context.Context, req Req, opts ...grpc.CallOption) (Res, error)
)

func handleResourceRequest[ReqRA reqRACommand, Event commands.EventContent, AsyncRes any](
	ctx context.Context,
	req reqCommand[ReqRA],
	action string,
	syncFunc syncFunc[ReqRA, Event],
	asyncFunc asyncFunc[ReqRA, AsyncRes],
	resp respCommand[Event],
) error {
	var err error
	var event Event

	raCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return resourceError(action, err)
	}

	if req.GetAsync() {
		_, err = asyncFunc(ctx, raCommand)
		if err != nil {
			return resourceError(action, err)
		}
		return nil
	}

	event, err = syncFunc(ctx, "*", raCommand)
	if err != nil {
		return resourceError(action, err)
	}
	if err = commands.CheckEventContent(event); err != nil {
		return resourceError(action, err)
	}

	resp.SetData(event)
	return nil
}

func (r *RequestHandler) CreateResource(ctx context.Context, req *pb.CreateResourceRequest) (*pb.CreateResourceResponse, error) {
	var resp pb.CreateResourceResponse
	return &resp, handleResourceRequest(
		ctx,
		req,
		"create",
		r.resourceAggregateClient.SyncCreateResource,
		r.resourceAggregateClient.CreateResource,
		&resp,
	)
}
