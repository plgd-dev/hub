package events

import (
	"context"
	"fmt"

	"github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
)

type ResourceLinksSnapshotTaken struct {
	pb.ResourceLinksSnapshotTaken
}

func (rls *ResourceLinksSnapshotTaken) AggregateId() string {
	return utils.MakeResourceId(rls.GetDeviceId(), utils.ResourceLinksHref)
}

func (rls *ResourceLinksSnapshotTaken) GroupId() string {
	return rls.GetDeviceId()
}

func (rls *ResourceLinksSnapshotTaken) Version() uint64 {
	return rls.GetEventMetadata().GetVersion()
}

func (rls *ResourceLinksSnapshotTaken) Marshal() ([]byte, error) {
	return proto.Marshal(&rls.ResourceLinksSnapshotTaken)
}

func (rls *ResourceLinksSnapshotTaken) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &rls.ResourceLinksSnapshotTaken)
}

func (rls *ResourceLinksSnapshotTaken) EventType() string {
	return http.ProtobufContentType(&pb.ResourceLinksSnapshotTaken{})
}

func (rls *ResourceLinksSnapshotTaken) HandleEventResourceLinksPublished(ctx context.Context, pub *ResourceLinksPublished) error {
	for rid, res := range pub.GetResources() {
		rls.GetResources()[rid] = res
	}
	return nil
}

func (rls *ResourceLinksSnapshotTaken) HandleEventResourceLinksUnpublished(ctx context.Context, upub *ResourceLinksUnpublished) error {
	if len(upub.Ids) == 0 {
		rls.Resources = make(map[string]*pb.Resource)
	} else {
		for _, rid := range upub.GetIds() {
			delete(rls.GetResources(), rid)
		}
	}
	return nil
}

func (rls *ResourceLinksSnapshotTaken) HandleEventResourceLinksSnapshotTaken(ctx context.Context, s *ResourceLinksSnapshotTaken) error {
	rls.Resources = s.GetResources()
	rls.DeviceId = s.GetDeviceId()
	rls.EventMetadata = s.GetEventMetadata()
	return nil
}

func (rls *ResourceLinksSnapshotTaken) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return status.Errorf(codes.Internal, "cannot determine type of event")
		}
		switch eu.EventType() {
		case http.ProtobufContentType(&pb.ResourceLinksSnapshotTaken{}):
			var s ResourceLinksSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rls.HandleEventResourceLinksSnapshotTaken(ctx, &s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceLinksPublished{}):
			var s ResourceLinksPublished
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rls.HandleEventResourceLinksPublished(ctx, &s); err != nil {
				return err
			}
		case http.ProtobufContentType(&pb.ResourceLinksUnpublished{}):
			var s ResourceLinksUnpublished
			if err := eu.Unmarshal(&s); err != nil {
				return status.Errorf(codes.Internal, "%v", err)
			}
			if err := rls.HandleEventResourceLinksUnpublished(ctx, &s); err != nil {
				return err
			}
		}
	}
	return iter.Err()
}

func (rls *ResourceLinksSnapshotTaken) HandleCommand(ctx context.Context, cmd aggregate.Command, newVersion uint64) ([]eventstore.Event, error) {
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid userID: %v", err)
	}
	switch req := cmd.(type) {
	case *pb.PublishResourceLinksRequest:
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		resMap := rls.GetResources()
		for _, res := range req.GetResources() {
			resMap[res.GetId()] = res
		}

		em := utils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := utils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")

		rlp := ResourceLinksPublished{pb.ResourceLinksPublished{
			Resources:     resMap,
			AuditContext:  &ac,
			EventMetadata: &em,
		},
		}
		err := rls.HandleEventResourceLinksPublished(ctx, &rlp)
		if err != nil {
			return nil, err
		}
		return []eventstore.Event{&rlp}, nil
	case *pb.UnpublishResourceLinksRequest:
		if newVersion == 0 {
			return nil, status.Errorf(codes.NotFound, errInvalidVersion)
		}
		if req.CommandMetadata == nil {
			return nil, status.Errorf(codes.InvalidArgument, errInvalidCommandMetadata)
		}

		em := utils.MakeEventMeta(req.CommandMetadata.ConnectionId, req.CommandMetadata.Sequence, newVersion)
		ac := utils.MakeAuditContext(req.GetAuthorizationContext().GetDeviceId(), userID, "")
		rlu := ResourceLinksUnpublished{pb.ResourceLinksUnpublished{
			Ids:           req.GetIds(),
			AuditContext:  &ac,
			EventMetadata: &em,
		}}
		err := rls.HandleEventResourceLinksUnpublished(ctx, &rlu)
		if err != nil {
			return nil, err
		}
		return []eventstore.Event{&rlu}, nil
	}

	return nil, fmt.Errorf("unknown command")
}

func (rs *ResourceLinksSnapshotTaken) SnapshotEventType() string { return rs.EventType() }

func (rs *ResourceLinksSnapshotTaken) TakeSnapshot(version uint64) (eventstore.Event, bool) {
	rs.EventMetadata.Version = version
	return rs, true
}

func NewResourceLinksSnapshotTaken() *ResourceLinksSnapshotTaken {

	return &ResourceLinksSnapshotTaken{
		ResourceLinksSnapshotTaken: pb.ResourceLinksSnapshotTaken{
			Resources:     make(map[string]*pb.Resource),
			EventMetadata: &pb.EventMetadata{},
		},
	}
}
