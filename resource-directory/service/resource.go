package service

import (
	"context"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
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

func (r *Resource) OnResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending *events.ResourceUpdatePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceUpdatePendingLocked(ctx, do)
}

func (r *Resource) OnResourceRetrievePendingLocked(ctx context.Context, do func(ctx context.Context, retrievePending *events.ResourceRetrievePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceRetrievePendingLocked(ctx, do)
}

func (r *Resource) OnResourceDeletePendingLocked(ctx context.Context, do func(ctx context.Context, deletePending *events.ResourceDeletePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceDeletePendingLocked(ctx, do)
}

func (r *Resource) OnResourceCreatePendingLocked(ctx context.Context, do func(ctx context.Context, createPending *events.ResourceCreatePending, version uint64) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceCreatePendingLocked(ctx, do)
}
