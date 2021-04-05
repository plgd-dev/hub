package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

type Resource struct {
	projection *resourceProjection
	Resource   *commands.Resource
}

func (r *Resource) GetContent() *commands.Content {
	if r.projection == nil {
		return nil
	}
	return r.projection.content.GetContent()
}

func (r *Resource) GetStatus() commands.Status {
	if r.projection == nil {
		return commands.Status_UNAVAILABLE
	}
	return r.projection.content.GetStatus()
}

func (r *Resource) OnResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending *pb.Event_ResourceUpdatePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceUpdatePendingLocked(ctx, do)
}

func (r *Resource) OnResourceRetrievePendingLocked(ctx context.Context, do func(ctx context.Context, retrievePending *pb.Event_ResourceRetrievePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceRetrievePendingLocked(ctx, do)
}

func (r *Resource) OnResourceDeletePendingLocked(ctx context.Context, do func(ctx context.Context, deletePending *pb.Event_ResourceDeletePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceDeletePendingLocked(ctx, do)
}

func (r *Resource) OnResourceCreatePendingLocked(ctx context.Context, do func(ctx context.Context, createPending *pb.Event_ResourceCreatePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceCreatePendingLocked(ctx, do)
}
