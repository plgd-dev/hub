package service

import (
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type Resource struct {
	projection *resourceProjection
	Resource   *commands.Resource
}

func (r *Resource) GetResourceChanged() *events.ResourceChanged {
	if r == nil {
		return nil
	}
	return r.projection.GetResourceChanged()
}

func (r *Resource) GetContent() *commands.Content {
	return r.GetResourceChanged().GetContent()
}
