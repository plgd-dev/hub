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

func (r *Resource) OnResourceUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, updatePending *events.ResourceUpdatePending) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceUpdatePendingLocked(ctx, do)
}

func (r *Resource) OnResourceRetrievePendingLocked(ctx context.Context, do func(ctx context.Context, retrievePending *events.ResourceRetrievePending) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceRetrievePendingLocked(ctx, do)
}

func (r *Resource) OnResourceDeletePendingLocked(ctx context.Context, do func(ctx context.Context, deletePending *events.ResourceDeletePending) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceDeletePendingLocked(ctx, do)
}

func (r *Resource) OnResourceCreatePendingLocked(ctx context.Context, do func(ctx context.Context, createPending *events.ResourceCreatePending) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceCreatePendingLocked(ctx, do)
}

func (r *Resource) OnResourceChangedLocked(ctx context.Context, do func(ctx context.Context, resourceChanged *events.ResourceChanged) error) error {
	if r.projection == nil {
		return nil
	}
	return r.projection.onResourceChangedLocked(ctx, do)
}
