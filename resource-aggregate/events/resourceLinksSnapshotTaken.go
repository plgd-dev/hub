package events

import (
	"context"
	"fmt"

	"github.com/plgd-dev/kit/net/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

const eventTypeResourceLinksSnapshotTaken = "ocf.cloud.resourceaggregate.events.resourcelinkssnapshottaken"

func (e *ResourceLinksSnapshotTaken) AggregateId() string {
	return commands.MakeLinksResourceUUID(e.GetDeviceId())
}

func (e *ResourceLinksSnapshotTaken) GroupId() string {
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

func (e *ResourceLinksSnapshotTaken) HandleEventResourceLinksPublished(ctx context.Context, pub *ResourceLinksPublished) error {
	for _, res := range pub.GetResources() {
		e.GetResources()[res.GetHref()] = res
	}
	e.DeviceId = pub.GetDeviceId()
	e.EventMetadata = pub.GetEventMetadata()
	return nil
}

func (e *ResourceLinksSnapshotTaken) HandleEventResourceLinksUnpublished(ctx context.Context, upub *ResourceLinksUnpublished) error {
	if len(upub.GetHrefs()) == 0 {
		for href, _ := range e.GetResources() {
			upub.Hrefs = append(upub.Hrefs, href)
		}
		e.Resources = make(map[string]*commands.Resource)
	} else {
		unpublished := []string{}
		for _, href := range upub.GetHrefs() {
			if _, present := e.GetResources()[href]; present {
				unpublished = append(unpublished, href)
				delete(e.GetResources(), href)
			}
		}
		upub.Hrefs = unpublished
	}
	e.EventMetadata = upub.GetEventMetadata()
	return nil
}

func (e *ResourceLinksSnapshotTaken) HandleEventResourceLinksSnapshotTaken(ctx context.Context, s *ResourceLinksSnapshotTaken) error {
	e.Resources = s.GetResources()
	e.DeviceId = s.GetDeviceId()
	e.EventMetadata = s.GetEventMetadata()
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
		switch eu.EventType() {
		case (&ResourceLinksSnapshotTaken{}).EventType():
			var s ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceLinksSnapshotTaken(ctx, &s); err != nil {
				return err
			}
		case (&ResourceLinksPublished{}).EventType():
			var s ResourceLinksPublished
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceLinksPublished(ctx, &s); err != nil {
				return err
			}
		case (&ResourceLinksUnpublished{}).EventType():
			var s ResourceLinksUnpublished
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := e.HandleEventResourceLinksUnpublished(ctx, &s); err != nil {
				return err
			}
		}
	}
	return iter.Err()
}

func (e *ResourceLinksSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid userID: %v", err)
	}
	switch req := cmd.(type) {
	case *commands.PublishResourceLinksRequest:
		if req.GetCommandMetadata() == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")

		rlp := ResourceLinksPublished{
			Resources:     req.GetResources(),
			DeviceId:      req.GetDeviceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
		}
		err := e.HandleEventResourceLinksPublished(ctx, &rlp)
		if err != nil {
			return nil, err
		}
		return []eventstore.Event{&rlp}, nil
	case *commands.UnpublishResourceLinksRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, errInvalidVersion)
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := MakeEventMeta(req.GetCommandMetadata().GetConnectionId(), req.GetCommandMetadata().GetSequence(), newVersion)
		ac := commands.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")
		rlu := ResourceLinksUnpublished{
			Hrefs:         req.GetHrefs(),
			DeviceId:      req.GetDeviceId(),
			AuditContext:  &ac,
			EventMetadata: &em,
		}
		err := e.HandleEventResourceLinksUnpublished(ctx, &rlu)
		if err != nil {
			return nil, err
		}
		return []eventstore.Event{&rlu}, nil
	}

	return nil, fmt.Errorf("unknown command")
}

func (e *ResourceLinksSnapshotTaken) SnapshotEventType() string { return e.EventType() }

func (e *ResourceLinksSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	e.EventMetadata.Version = version
	return e, true
}

func NewResourceLinksSnapshotTaken() *ResourceLinksSnapshotTaken {

	return &ResourceLinksSnapshotTaken{
		Resources:     make(map[string]*commands.Resource),
		EventMetadata: &EventMetadata{},
	}
}
