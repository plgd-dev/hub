package pb

import (
	"encoding/base64"
	"slices"
	"strings"

	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

func (e *Event_ResourceChanged) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceChanged.GetResourceId()
}

func (e *Event_ResourceUpdatePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceUpdatePending.GetResourceId()
}

func (e *Event_ResourceUpdated) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceUpdated.GetResourceId()
}

func (e *Event_ResourceRetrievePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceRetrievePending.GetResourceId()
}

func (e *Event_ResourceRetrieved) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceRetrieved.GetResourceId()
}

func (e *Event_ResourceDeletePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceDeletePending.GetResourceId()
}

func (e *Event_ResourceDeleted) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceDeleted.GetResourceId()
}

func (e *Event_ResourceCreatePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceCreatePending.GetResourceId()
}

func (e *Event_ResourceCreated) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceCreated.GetResourceId()
}

func (f *ResourceIdFilter) ToString() string {
	if f == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(f.GetResourceId().GetDeviceId())
	if f.GetResourceId().GetHref() == "" {
		return sb.String()
	}
	sb.WriteString(f.GetResourceId().GetHref())
	if len(f.GetEtag()) == 0 {
		return sb.String()
	}
	for i, etag := range f.GetEtag() {
		if i == 0 {
			sb.WriteString("?")
		} else {
			sb.WriteString("&")
		}
		sb.WriteString("etag=")
		sb.WriteString(base64.StdEncoding.EncodeToString(etag))
	}
	return sb.String()
}

func (r *Resource) Clone() *Resource {
	return &Resource{
		Types: slices.Clone(r.GetTypes()),
		Data:  r.GetData().Clone(),
	}
}
