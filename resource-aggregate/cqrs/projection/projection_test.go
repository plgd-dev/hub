package projection

import (
	"context"
	"testing"

	kitCqrsUtils "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	mockEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/cqrs/event"
	"github.com/plgd-dev/cqrs/eventstore"
	cqrsEventStore "github.com/plgd-dev/cqrs/eventstore"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/http"
	"github.com/stretchr/testify/assert"
)

type mockResourceCtx struct {
	pbRA.ResourceStateSnapshotTaken
	UpdatePending map[string]raEvents.ResourceUpdatePending
}

func (m *mockResourceCtx) SnapshotEventType() string {
	s := &raEvents.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}

func (m *mockResourceCtx) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler

	for iter.Next(ctx, &eu) {
		log.Debugf("resourceCtx.Handle: DeviceId: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupId, eu.AggregateId, eu.Version, eu.EventType)
		switch eu.EventType {
		case http.ProtobufContentType(&pbRA.ResourceStateSnapshotTaken{}):
			var s raEvents.ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.LatestResourceChange = s.LatestResourceChange
			m.Resource = s.Resource
			m.Id = s.Id
			m.IsPublished = s.IsPublished
			m.EventMetadata = s.EventMetadata
		case http.ProtobufContentType(&pbRA.ResourcePublished{}):
			var s raEvents.ResourcePublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.Id = s.Id
			m.IsPublished = true
			m.Resource = s.Resource
		case http.ProtobufContentType(&pbRA.ResourceUnpublished{}):
			m.IsPublished = false
		case http.ProtobufContentType(&pbRA.ResourceUpdatePending{}):
			var s raEvents.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.UpdatePending[s.AuditContext.CorrelationId] = s
		case http.ProtobufContentType(&pbRA.ResourceUpdated{}):
			var s raEvents.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			delete(m.UpdatePending, s.AuditContext.CorrelationId)
		case http.ProtobufContentType(&pbRA.ResourceChanged{}):
			var s raEvents.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.LatestResourceChange = &s.ResourceChanged
		}
	}
	return nil
}

var res0 = pbRA.Resource{
	Id:       "res0",
	DeviceId: "dev0",
	Href:     "/res0",
}
var res1 = pbRA.Resource{
	Id:       "res1",
	DeviceId: "dev1",
	Href:     "/res1",
}

var res2 = pbRA.Resource{
	Id:       "res2",
	DeviceId: "dev0",
	Href:     "/res2",
}

var res3 = pbRA.Resource{
	Id:       "res3",
	DeviceId: "dev0",
	Href:     "/res3",
}

var res4 = pbRA.Resource{
	Id:       "res4",
	DeviceId: "dev1",
	Href:     "/res4",
}

func makeEventMeta(connectionId string, sequence, version uint64) pb.EventMetadata {
	e := kitCqrsUtils.MakeEventMeta(connectionId, sequence, version)
	e.TimestampMs = 12345
	return e
}

func prepareResourceEventstore(t *testing.T) *mockEventStore.MockEventStore {

	eventstore := mockEventStore.NewMockEventStore()

	eventstore.Append(res0.DeviceId, res0.Id, mockEvents.MakeResourcePublishedEvent(res0, makeEventMeta("a", 0, 0)))

	eventstore.Append(res1.DeviceId, res1.Id, mockEvents.MakeResourcePublishedEvent(res1, makeEventMeta("a", 0, 0)))
	eventstore.Append(res1.DeviceId, res1.Id, mockEvents.MakeResourceUnpublishedEvent(res1.Id, res1.DeviceId, makeEventMeta("a", 0, 1)))

	resourceChangedEventMetadata := makeEventMeta("", 0, 0)
	eventstore.Append(res2.DeviceId, res2.Id, mockEvents.MakeResourcePublishedEvent(res2, makeEventMeta("a", 0, 0)))
	eventstore.Append(res2.DeviceId, res2.Id, mockEvents.MakeResourceStateSnapshotTaken(true, res2, pbRA.ResourceChanged{Content: &pbRA.Content{}, EventMetadata: &resourceChangedEventMetadata}, makeEventMeta("a", 0, 1)))

	eventstore.Append(res3.DeviceId, res3.Id, mockEvents.MakeResourceStateSnapshotTaken(true, res3, pbRA.ResourceChanged{Content: &pbRA.Content{}, EventMetadata: &resourceChangedEventMetadata}, makeEventMeta("a", 0, 0)))
	eventstore.Append(res3.DeviceId, res3.Id, mockEvents.MakeResourceUnpublishedEvent(res3.Id, res3.DeviceId, makeEventMeta("a", 0, 1)))
	eventstore.Append(res3.DeviceId, res3.Id, mockEvents.MakeResourceUpdatePending(res3.DeviceId, res3.Id, pbRA.Content{}, makeEventMeta("a", 0, 2)))
	eventstore.Append(res3.DeviceId, res3.Id, mockEvents.MakeResourcePublishedEvent(res3, makeEventMeta("a", 0, 3)))

	eventstore.Append(res4.DeviceId, res4.Id, mockEvents.MakeResourceStateSnapshotTaken(true, res4, pbRA.ResourceChanged{Content: &pbRA.Content{}, EventMetadata: &resourceChangedEventMetadata}, makeEventMeta("a", 0, 0)))
	eventstore.Append(res4.DeviceId, res4.Id, mockEvents.MakeResourceUnpublishedEvent(res4.Id, res4.DeviceId, makeEventMeta("a", 0, 1)))
	eventstore.Append(res4.DeviceId, res4.Id, mockEvents.MakeResourceUpdatePending(res4.DeviceId, res4.Id, pbRA.Content{}, makeEventMeta("a", 0, 2)))
	eventstore.Append(res4.DeviceId, res4.Id, mockEvents.MakeResourceUpdated(res4.DeviceId, res4.Id, pbRA.Status_OK, pbRA.Content{}, makeEventMeta("a", 0, 3)))

	return eventstore
}

func TestResourceProjection_Register(t *testing.T) {
	type args struct {
		deviceId string
	}
	tests := []struct {
		name       string
		args       args
		wantLoaded bool
		wantErr    bool
	}{
		{
			name: "first valid",
			args: args{
				deviceId: res0.DeviceId,
			},
			wantLoaded: true,
		},
		{
			name: "second valid",
			args: args{
				deviceId: res0.DeviceId,
			},
		},
		{
			name: "error",
			args: args{
				deviceId: "error",
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context) (cqrsEventStore.Model, error) {
			return &mockResourceCtx{
				UpdatePending: make(map[string]raEvents.ResourceUpdatePending),
			}, nil
		},
	)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLoaded, err := p.Register(ctx, tt.args.deviceId)
			if tt.wantLoaded {
				assert.True(t, gotLoaded)
			} else {
				assert.False(t, gotLoaded)
			}
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceProjection_Unregister(t *testing.T) {
	type args struct {
		deviceId string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "first time",
			args: args{
				deviceId: res0.DeviceId,
			},
		},
		{
			name: "second second",
			args: args{
				deviceId: res0.DeviceId,
			},
		},
		{
			name: "third error",
			args: args{
				deviceId: res0.DeviceId,
			},
			wantErr: true,
		},
		{
			name: "not registered",
			args: args{
				deviceId: res1.DeviceId,
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context) (cqrsEventStore.Model, error) {
			return &mockResourceCtx{
				UpdatePending: make(map[string]raEvents.ResourceUpdatePending),
			}, nil
		},
	)
	assert.NoError(t, err)
	_, err = p.Register(ctx, res0.DeviceId)
	assert.NoError(t, err)
	_, err = p.Register(ctx, res0.DeviceId)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.Unregister(tt.args.deviceId)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceProjection_Models(t *testing.T) {
	type args struct {
		deviceId   string
		resourceId string
	}
	tests := []struct {
		name string
		args args
		want []eventstore.Model
	}{
		{
			name: "valid",
			args: args{
				deviceId: res0.DeviceId,
			},
			want: []eventstore.Model{
				&mockResourceCtx{
					ResourceStateSnapshotTaken: pbRA.ResourceStateSnapshotTaken{
						Id:          res0.Id,
						Resource:    &res0,
						IsPublished: true,
					},
					UpdatePending: make(map[string]raEvents.ResourceUpdatePending),
				},
				&mockResourceCtx{
					ResourceStateSnapshotTaken: pbRA.ResourceStateSnapshotTaken{
						Id:          res2.Id,
						Resource:    &res2,
						IsPublished: true,
						LatestResourceChange: &pbRA.ResourceChanged{
							Content: &pbRA.Content{},
							EventMetadata: &pb.EventMetadata{
								TimestampMs: 12345,
							},
						},
						EventMetadata: &pb.EventMetadata{
							Version:      1,
							TimestampMs:  12345,
							ConnectionId: "a",
						},
					},
					UpdatePending: make(map[string]raEvents.ResourceUpdatePending),
				},
				&mockResourceCtx{
					ResourceStateSnapshotTaken: pbRA.ResourceStateSnapshotTaken{
						Id:          res3.Id,
						Resource:    &res3,
						IsPublished: true,
						LatestResourceChange: &pbRA.ResourceChanged{
							Content: &pbRA.Content{},
							EventMetadata: &pb.EventMetadata{
								TimestampMs: 12345,
							},
						},
						EventMetadata: &pb.EventMetadata{
							TimestampMs:  12345,
							ConnectionId: "a",
						},
					},
					UpdatePending: map[string]raEvents.ResourceUpdatePending{
						"": events.ResourceUpdatePending{
							pbRA.ResourceUpdatePending{
								Id:      "res3",
								Content: &pbRA.Content{},
								EventMetadata: &pb.EventMetadata{
									ConnectionId: "a",
									Sequence:     0,
									Version:      2,
									TimestampMs:  12345,
								},
								AuditContext: &pb.AuditContext{
									UserId:   "userId",
									DeviceId: "dev0",
								},
							},
						},
					},
				},
			},
		},
	}

	eventstore := prepareResourceEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context) (cqrsEventStore.Model, error) {
			return &mockResourceCtx{
				UpdatePending: make(map[string]raEvents.ResourceUpdatePending),
			}, nil
		},
	)
	assert.NoError(t, err)
	_, err = p.Register(ctx, res0.DeviceId)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Models(tt.args.deviceId, tt.args.resourceId)

			mapWant := make(map[string]*mockResourceCtx)
			for _, r := range tt.want {
				m := r.(*mockResourceCtx)
				mapWant[m.Id] = m
			}
			mapGot := make(map[string]*mockResourceCtx)
			for _, r := range got {
				m := r.(*mockResourceCtx)
				mapGot[m.Id] = m
			}

			assert.Equal(t, mapWant, mapGot)
		})
	}
}

func TestResourceProjection_ForceUpdate(t *testing.T) {
	type args struct {
		deviceId   string
		resourceId string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceId: res0.DeviceId,
			},
		},
		{
			name: "error",
			args: args{
				deviceId: "error",
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context) (cqrsEventStore.Model, error) {
			return &mockResourceCtx{
				UpdatePending: make(map[string]raEvents.ResourceUpdatePending),
			}, nil
		},
	)
	assert.NoError(t, err)
	_, err = p.Register(ctx, res0.DeviceId)
	assert.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ForceUpdate(ctx, tt.args.deviceId, tt.args.resourceId)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
