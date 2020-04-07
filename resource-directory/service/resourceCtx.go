package service

import (
	"context"
	"sync"

	"github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/cqrs/eventstore"
	httpUtils "github.com/go-ocf/kit/net/http"
	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

type resourceCtx struct {
	lock     sync.Mutex
	snapshot *raEvents.ResourceStateSnapshotTaken
}

func NewResourceCtx() func(context.Context) (eventstore.Model, error) {
	return func(context.Context) (eventstore.Model, error) {
		return &resourceCtx{
			snapshot: raEvents.NewResourceStateSnapshotTaken(func(string, string) error { return nil }),
		}, nil
	}
}

func (m *resourceCtx) cloneLocked() *resourceCtx {
	ra := raEvents.NewResourceStateSnapshotTaken(func(string, string) error { return nil })
	ra.ResourceStateSnapshotTaken.LatestResourceChange = m.snapshot.LatestResourceChange
	ra.ResourceStateSnapshotTaken.EventMetadata = m.snapshot.EventMetadata
	ra.ResourceStateSnapshotTaken.Resource = m.snapshot.Resource
	ra.ResourceStateSnapshotTaken.TimeToLive = m.snapshot.TimeToLive
	ra.ResourceStateSnapshotTaken.IsPublished = m.snapshot.IsPublished
	ra.ResourceStateSnapshotTaken.Id = m.snapshot.Id
	return &resourceCtx{
		snapshot: ra,
	}
}

func (m *resourceCtx) Clone() *resourceCtx {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.cloneLocked()
}

func (m *resourceCtx) Handle(ctx context.Context, iter event.Iter) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.snapshot.Handle(ctx, iter)
}

func (m *resourceCtx) SnapshotEventType() string {
	return httpUtils.ProtobufContentType(&pbRA.ResourceStateSnapshotTaken{})
}
