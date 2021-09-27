package service

import (
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type Resource struct {
	projection *resourceProjection
	Resource   *commands.Resource
}

func (r *Resource) GetResourceChanged() *events.ResourceChanged {
	if r == nil {
		return nil
	}
	if r.projection == nil {
		return nil
	}
	return r.projection.content
}

func (r *Resource) GetContent() *commands.Content {
	if r.projection == nil {
		return nil
	}
	return r.GetResourceChanged().GetContent()
}

func (r *Resource) GetStatus() commands.Status {
	if r.projection == nil {
		return commands.Status_UNAVAILABLE
	}
	return r.GetResourceChanged().GetStatus()
}
