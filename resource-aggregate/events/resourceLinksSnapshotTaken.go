package events

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/coap-gateway/resource"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/propagation"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceLinksSnapshotTaken = "resourcelinkssnapshottaken"

func (e *ResourceLinksSnapshotTaken) AggregateID() string {
	return commands.MakeLinksResourceUUID(e.GetDeviceId()).String()
}

func (e *ResourceLinksSnapshotTaken) GroupID() string {
	return e.GetDeviceId()
}

func (e *ResourceLinksSnapshotTaken) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceLinksSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceLinksSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceLinksSnapshotTaken) EventType() string {
	return eventTypeResourceLinksSnapshotTaken
}

func (e *ResourceLinksSnapshotTaken) IsSnapshot() bool {
	return true
}

func (e *ResourceLinksSnapshotTaken) ETag() *eventstore.ETagData {
	return nil
}

func (e *ResourceLinksSnapshotTaken) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceLinksSnapshotTaken) ServiceID() (string, bool) {
	return "", false
}

func (e *ResourceLinksSnapshotTaken) Types() []string {
	return nil
}

func (e *ResourceLinksSnapshotTaken) CloneData(event *ResourceLinksSnapshotTaken) {
	e.DeviceId = event.GetDeviceId()
	e.Resources = commands.CloneResourcesMap(event.GetResources())
	e.EventMetadata = event.GetEventMetadata().Clone()
	e.AuditContext = event.GetAuditContext().Clone()
}

func (e *ResourceLinksSnapshotTaken) CopyData(event *ResourceLinksSnapshotTaken) {
	e.DeviceId = event.GetDeviceId()
	e.Resources = event.GetResources()
	e.EventMetadata = event.GetEventMetadata()
	e.AuditContext = event.GetAuditContext()
}

func (e *ResourceLinksSnapshotTaken) CheckInitialized() bool {
	return e.GetResources() != nil &&
		e.GetDeviceId() != "" &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}

// Examine published resources by the ResourceLinksPublished, compare it with cached resources and
// return array of new or changed resources.
func (e *ResourceLinksSnapshotTaken) GetNewPublishedLinks(pub *ResourceLinksPublished) []*commands.Resource {
	if e.GetResources() == nil {
		return pub.GetResources()
	}

	published := make([]*commands.Resource, 0, len(pub.GetResources()))

	for _, resPub := range pub.GetResources() {
		resSnap, ok := e.GetResources()[resPub.GetHref()]
		if !ok || !EqualResource(resPub, resSnap) {
			published = append(published, resPub)
		}
	}

	return published
}

func (e *ResourceLinksSnapshotTaken) HandleEventResourceLinksPublished(pub *ResourceLinksPublished) []*commands.Resource {
	published := e.GetNewPublishedLinks(pub)

	for _, res := range published {
		if e.GetResources() == nil {
			e.Resources = make(map[string]*commands.Resource)
		}
		e.GetResources()[res.GetHref()] = res
	}
	e.DeviceId = pub.GetDeviceId()
	e.EventMetadata = pub.GetEventMetadata()
	e.AuditContext = pub.GetAuditContext()
	return published
}

func (e *ResourceLinksSnapshotTaken) unpublishResourceLinksAndUpdateCache(instanceIDs []int64, upub *ResourceLinksUnpublished) []string {
	if len(e.GetResources()) == 0 {
		return nil
	}

	if len(upub.GetHrefs()) == 0 && len(instanceIDs) == 0 {
		unpublished := make([]string, 0, len(e.GetResources()))
		for href := range e.GetResources() {
			unpublished = append(unpublished, href)
		}
		e.Resources = make(map[string]*commands.Resource)
		return unpublished
	}

	unpublished := make([]string, 0, len(upub.GetHrefs())+len(instanceIDs))
	for _, href := range upub.GetHrefs() {
		if _, present := e.GetResources()[href]; present {
			unpublished = append(unpublished, href)
			delete(e.GetResources(), href)
		}
	}

	if len(instanceIDs) == 0 {
		return unpublished
	}

	instanceIDToHref := map[int64]string{}
	for href := range e.GetResources() {
		instanceIDToHref[resource.GetInstanceID(href)] = href
	}

	for _, insID := range instanceIDs {
		if href, present := instanceIDToHref[insID]; present {
			unpublished = append(unpublished, href)
			delete(instanceIDToHref, insID)
			delete(e.GetResources(), href)
		}
	}

	return unpublished
}

func (e *ResourceLinksSnapshotTaken) HandleEventResourceLinksUnpublished(instanceIDs []int64, upub *ResourceLinksUnpublished) []string {
	unpublished := e.unpublishResourceLinksAndUpdateCache(instanceIDs, upub)
	e.EventMetadata = upub.GetEventMetadata()
	e.AuditContext = upub.GetAuditContext()
	return unpublished
}

func (e *ResourceLinksSnapshotTaken) HandleEventResourceLinksSnapshotTaken(s *ResourceLinksSnapshotTaken) {
	e.CopyData(s)
}

func (e *ResourceLinksSnapshotTaken) handleByEvent(eu eventstore.EventUnmarshaler) error {
	switch eu.EventType() {
	case (&ResourceLinksSnapshotTaken{}).EventType():
		var s ResourceLinksSnapshotTaken
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		e.HandleEventResourceLinksSnapshotTaken(&s)
	case (&ResourceLinksPublished{}).EventType():
		var s ResourceLinksPublished
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		e.HandleEventResourceLinksPublished(&s)
	case (&ResourceLinksUnpublished{}).EventType():
		var s ResourceLinksUnpublished
		if err := eu.Unmarshal(&s); err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		e.HandleEventResourceLinksUnpublished(nil, &s)
	}
	return nil
}

func (e *ResourceLinksSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		if err := e.handleByEvent(eu); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (e *ResourceLinksSnapshotTakenForCommand) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	switch req := cmd.(type) {
	case *commands.PublishResourceLinksRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
		ac := commands.NewAuditContext(e.userID, "", e.owner)

		rlp := ResourceLinksPublished{
			Resources:            req.GetResources(),
			DeviceId:             req.GetDeviceId(),
			AuditContext:         ac,
			EventMetadata:        em,
			OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		}
		published := e.HandleEventResourceLinksPublished(&rlp)
		if len(published) == 0 {
			return nil, nil
		}
		rlp.Resources = published
		return []eventstore.Event{&rlp}, nil
	case *commands.UnpublishResourceLinksRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, errInvalidVersion)
		}
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion, e.hubID)
		ac := commands.NewAuditContext(e.userID, "", e.owner)
		rlu := ResourceLinksUnpublished{
			DeviceId:             req.GetDeviceId(),
			Hrefs:                req.GetHrefs(),
			AuditContext:         ac,
			EventMetadata:        em,
			OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
		}
		unpublished := e.HandleEventResourceLinksUnpublished(req.GetInstanceIds(), &rlu)
		if len(unpublished) == 0 {
			return nil, nil
		}
		rlu.Hrefs = unpublished
		return []eventstore.Event{&rlu}, nil
	}

	return nil, fmt.Errorf("unknown command (%T)", cmd)
}

func (e *ResourceLinksSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	// we need to return as new event because `e` is a pointer,
	// otherwise ResourceLinksSnapshotTaken.Handle override version/resource of snapshot which will be fired to eventbus
	resources := make(map[string]*commands.Resource)
	for key, resource := range e.GetResources() {
		resources[key] = resource
	}
	return &ResourceLinksSnapshotTaken{
		DeviceId:      e.GetDeviceId(),
		EventMetadata: MakeEventMeta(e.GetEventMetadata().GetConnectionId(), e.GetEventMetadata().GetSequence(), version, e.GetEventMetadata().GetHubId()),
		Resources:     resources,
		AuditContext:  e.GetAuditContext(),
	}, true
}

type ResourceLinksSnapshotTakenForCommand struct {
	userID string
	owner  string
	hubID  string
	*ResourceLinksSnapshotTaken
}

func NewResourceLinksSnapshotTakenForCommand(userID string, owner string, hubID string) *ResourceLinksSnapshotTakenForCommand {
	return &ResourceLinksSnapshotTakenForCommand{
		ResourceLinksSnapshotTaken: NewResourceLinksSnapshotTaken(),
		userID:                     userID,
		owner:                      owner,
		hubID:                      hubID,
	}
}

func NewResourceLinksSnapshotTaken() *ResourceLinksSnapshotTaken {
	return &ResourceLinksSnapshotTaken{
		Resources:     make(map[string]*commands.Resource),
		EventMetadata: &EventMetadata{},
	}
}

func (e *ResourceLinksSnapshotTaken) ToResourceLinksPublished() *ResourceLinksPublished {
	resources := make([]*commands.Resource, 0, len(e.GetResources()))
	for _, r := range e.GetResources() {
		resources = append(resources, r)
	}

	return &ResourceLinksPublished{
		DeviceId:      e.GetDeviceId(),
		EventMetadata: e.GetEventMetadata(),
		Resources:     resources,
		AuditContext:  e.GetAuditContext(),
	}
}
