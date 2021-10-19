package test

import (
	"sort"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
)

type SortResourcesByHref []*commands.Resource

func (a SortResourcesByHref) Len() int      { return len(a) }
func (a SortResourcesByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortResourcesByHref) Less(i, j int) bool {
	return a[i].GetHref() < a[j].GetHref()
}

func SortResources(s []*commands.Resource) []*commands.Resource {
	v := SortResourcesByHref(s)
	sort.Sort(v)
	return v
}

func ResourceLinksToResources(deviceID string, s []schema.ResourceLink) []*commands.Resource {
	r := make([]*commands.Resource, 0, len(s))
	for _, l := range s {
		l.DeviceID = deviceID
		r = append(r, commands.SchemaResourceLinkToResource(l, time.Time{}))
	}
	CleanUpResourcesArray(r)
	return r
}

func ResourceLinksToResources2(deviceID string, s []schema.ResourceLink) []*pb.Resource {
	r := make([]*pb.Resource, 0, len(s))
	for _, l := range s {
		res := pb.Resource{
			Types: l.ResourceTypes,
			Data: &events.ResourceChanged{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     l.Href,
				},
			},
		}
		r = append(r, &res)
	}
	return r
}

func CleanUpResourcesArray(resources []*commands.Resource) []*commands.Resource {
	for _, r := range resources {
		r.ValidUntil = 0
	}
	SortResources(resources)
	return resources
}

func CleanUpResourceLinksSnapshotTaken(e *events.ResourceLinksSnapshotTaken) *events.ResourceLinksSnapshotTaken {
	e.EventMetadata = nil
	for _, r := range e.GetResources() {
		r.ValidUntil = 0
	}
	return e
}

func CleanUpResourceLinksPublished(e *events.ResourceLinksPublished) *events.ResourceLinksPublished {
	e.EventMetadata = nil
	e.AuditContext = nil
	CleanUpResourcesArray(e.GetResources())
	return e
}
