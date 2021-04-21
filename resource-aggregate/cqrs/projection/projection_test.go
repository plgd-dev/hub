package projection

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	cqrsEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	mockEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
)

var d1res1 = commands.Resource{
	DeviceId: "dev1",
	Href:     "/res1",
}

var d1res2 = commands.Resource{
	DeviceId: "dev1",
	Href:     "/res2",
}

var d1res3 = commands.Resource{
	DeviceId: "dev1",
	Href:     "/res3",
}

var d1res4 = commands.Resource{
	DeviceId: "dev1",
	Href:     "/res4",
}

var d1res5 = commands.Resource{
	DeviceId: "dev1",
	Href:     "/res5",
}

var d2res1 = commands.Resource{
	DeviceId: "dev2",
	Href:     "/res1",
}

var d2res2 = commands.Resource{
	DeviceId: "dev2",
	Href:     "/res2",
}

var d3res1 = commands.Resource{
	DeviceId: "dev3",
	Href:     "/res1",
}

var d3res2 = commands.Resource{
	DeviceId: "dev3",
	Href:     "/res2",
}

var d4res1 = commands.Resource{
	DeviceId: "dev4",
	Href:     "/res1",
}

var d4res2 = commands.Resource{
	DeviceId: "dev4",
	Href:     "/res2",
}

var d5res1 = commands.Resource{
	DeviceId: "dev5",
	Href:     "/res1",
}

var d5res2 = commands.Resource{
	DeviceId: "dev5",
	Href:     "/res2",
}

func makeEventMeta(connectionID string, sequence, version uint64) *events.EventMetadata {
	e := events.MakeEventMeta(connectionID, sequence, version)
	e.TimestampMs = 12345
	return e
}

func prepareResourceLinksEventstore(t *testing.T) *mockEventStore.MockEventStore {
	eventstore := mockEventStore.NewMockEventStore()

	d1resID := commands.MakeLinksResourceUUID(d1res1.DeviceId)
	eventstore.Append(d1res1.DeviceId, d1resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d1res1}, d1res1.DeviceId, makeEventMeta("a", 0, 0)))
	eventstore.Append(d1res2.DeviceId, d1resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d1res2, &d1res3}, d1res2.DeviceId, makeEventMeta("a", 0, 1)))
	eventstore.Append(d1res2.DeviceId, d1resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{d1res2.Href}, d1res2.DeviceId, makeEventMeta("a", 0, 2)))
	eventstore.Append(d1res4.DeviceId, d1resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d1res4, &d1res5}, d1res4.DeviceId, makeEventMeta("a", 0, 3)))

	d2resID := commands.MakeLinksResourceUUID(d2res1.DeviceId)
	eventstore.Append(d2res1.DeviceId, d2resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d2res1, &d2res2}, d2res1.DeviceId, makeEventMeta("a", 0, 0)))
	eventstore.Append(d2res1.DeviceId, d2resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res1.Href}, d2res1.DeviceId, makeEventMeta("a", 0, 1)))
	eventstore.Append(d2res2.DeviceId, d2resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res2.Href}, d2res2.DeviceId, makeEventMeta("a", 0, 2)))
	eventstore.Append(d2res2.DeviceId, d2resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res2.Href}, d2res2.DeviceId, makeEventMeta("a", 0, 3)))
	eventstore.Append(d2res1.DeviceId, d2resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d2res1, &d2res1}, d2res1.DeviceId, makeEventMeta("a", 0, 4)))
	eventstore.Append(d2res2.DeviceId, d2resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res2.Href, d2res2.Href}, d2res2.DeviceId, makeEventMeta("a", 0, 5)))
	eventstore.Append(d2res1.DeviceId, d2resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d2res1}, d2res1.DeviceId, makeEventMeta("a", 0, 6)))

	d3resID := commands.MakeLinksResourceUUID(d3res1.DeviceId)
	eventstore.Append(d3res1.DeviceId, d3resID, mockEvents.MakeResourceLinksSnapshotTaken(map[string]*commands.Resource{d3res1.Href: &d3res1, d3res2.Href: &d3res2}, d3res1.DeviceId, makeEventMeta("a", 0, 0)))
	eventstore.Append(d3res1.DeviceId, d3resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{d3res1.Href}, d3res1.DeviceId, makeEventMeta("a", 0, 1)))

	d4resID := commands.MakeLinksResourceUUID(d4res1.DeviceId)
	eventstore.Append(d4res1.DeviceId, d4resID, mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d4res1, &d4res2}, d4res1.DeviceId, makeEventMeta("a", 0, 0)))
	eventstore.Append(d4res1.DeviceId, d4resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{}, d4res1.DeviceId, makeEventMeta("a", 0, 1)))

	d5resID := commands.MakeLinksResourceUUID(d5res1.DeviceId)
	eventstore.Append(d5res1.DeviceId, d5resID, mockEvents.MakeResourceLinksSnapshotTaken(map[string]*commands.Resource{d5res1.Href: &d5res1, d5res2.Href: &d5res2}, d5res1.DeviceId, makeEventMeta("a", 0, 0)))
	eventstore.Append(d5res1.DeviceId, d5resID, mockEvents.MakeResourceLinksUnpublishedEvent([]string{}, d5res1.DeviceId, makeEventMeta("a", 0, 1)))

	return eventstore
}

func prepareResourceStateEventstore(t *testing.T) *mockEventStore.MockEventStore {
	eventstore := mockEventStore.NewMockEventStore()
	resourceChangedEventMetadata := makeEventMeta("", 0, 0)

	d1r1 := commands.NewResourceID(d1res1.DeviceId, d1res1.Href)
	eventstore.Append(d1res1.DeviceId, d1r1.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d1r1, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	eventstore.Append(d1res1.DeviceId, d1r1.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r1, &commands.Content{}, makeEventMeta("a", 0, 1), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d1res1.DeviceId, d1r1.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r1, &commands.Content{}, makeEventMeta("a", 0, 2), mockEvents.MakeAuditContext("userId", "2")))
	eventstore.Append(d1res1.DeviceId, d1r1.ToUUID(), mockEvents.MakeResourceUpdated(d1r1, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 0, 3), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d1res1.DeviceId, d1r1.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r1, &commands.Content{}, makeEventMeta("a", 0, 4), mockEvents.MakeAuditContext("userId", "3")))

	d1r2 := commands.NewResourceID(d1res2.DeviceId, d1res2.Href)
	eventstore.Append(d1res2.DeviceId, d1r2.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d1r2, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "2")))

	d1r3 := commands.NewResourceID(d1res3.DeviceId, d1res3.Href)
	eventstore.Append(d1res3.DeviceId, d1r3.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d1r3, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "2")))
	eventstore.Append(d1res3.DeviceId, d1r3.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r3, &commands.Content{}, makeEventMeta("a", 0, 1), mockEvents.MakeAuditContext("userId", "3")))

	d1r4 := commands.NewResourceID(d1res4.DeviceId, d1res4.Href)
	eventstore.Append(d1res4.DeviceId, d1r4.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d1r4, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	eventstore.Append(d1res4.DeviceId, d1r4.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r4, &commands.Content{}, makeEventMeta("a", 0, 1), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d1res4.DeviceId, d1r4.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r4, &commands.Content{}, makeEventMeta("a", 0, 2), mockEvents.MakeAuditContext("userId", "2")))
	eventstore.Append(d1res4.DeviceId, d1r4.ToUUID(), mockEvents.MakeResourceUpdated(d1r4, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 0, 3), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d1res4.DeviceId, d1r4.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d1r4, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 4), mockEvents.MakeAuditContext("userId", "3")))
	eventstore.Append(d1res4.DeviceId, d1r4.ToUUID(), mockEvents.MakeResourceUpdatePending(d1r4, &commands.Content{}, makeEventMeta("a", 0, 5), mockEvents.MakeAuditContext("userId", "4")))

	d2r1 := commands.NewResourceID(d2res1.DeviceId, d2res1.Href)
	eventstore.Append(d2res1.DeviceId, d2r1.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d2r1, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	eventstore.Append(d2res1.DeviceId, d2r1.ToUUID(), mockEvents.MakeResourceUpdatePending(d2r1, &commands.Content{}, makeEventMeta("a", 0, 1), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d2res1.DeviceId, d2r1.ToUUID(), mockEvents.MakeResourceUpdated(d2r1, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 0, 2), mockEvents.MakeAuditContext("userId", "1")))

	d2r2 := commands.NewResourceID(d2res2.DeviceId, d2res2.Href)
	eventstore.Append(d2res2.DeviceId, d2r2.ToUUID(), mockEvents.MakeResourceStateSnapshotTaken(d2r2, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	eventstore.Append(d2res2.DeviceId, d2r2.ToUUID(), mockEvents.MakeResourceUpdatePending(d2r2, &commands.Content{}, makeEventMeta("a", 0, 1), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d2res2.DeviceId, d2r2.ToUUID(), mockEvents.MakeResourceUpdated(d2r2, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 0, 2), mockEvents.MakeAuditContext("userId", "1")))
	eventstore.Append(d2res2.DeviceId, d2r2.ToUUID(), mockEvents.MakeResourceChangedEvent(d2r2, &commands.Content{}, makeEventMeta("a", 0, 3), mockEvents.MakeAuditContext("userId", "2")))

	return eventstore
}

func TestResourceProjection_Register(t *testing.T) {
	type args struct {
		deviceID string
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
				deviceID: d1res1.DeviceId,
			},
			wantLoaded: true,
		},
		{
			name: "second valid",
			args: args{
				deviceID: d1res1.DeviceId,
			},
		},
		{
			name: "error",
			args: args{
				deviceID: "error",
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceLinksEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context, groupID, aggregateID string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLoaded, err := p.Register(ctx, tt.args.deviceID)
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
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "first time",
			args: args{
				deviceID: d1res1.DeviceId,
			},
		},
		{
			name: "second second",
			args: args{
				deviceID: d1res1.DeviceId,
			},
		},
		{
			name: "third error",
			args: args{
				deviceID: d1res1.DeviceId,
			},
			wantErr: true,
		},
		{
			name: "not registered",
			args: args{
				deviceID: d2res1.DeviceId,
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceLinksEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context, groupID, aggregateID string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	assert.NoError(t, err)
	_, err = p.Register(ctx, d1res1.DeviceId)
	assert.NoError(t, err)
	_, err = p.Register(ctx, d1res1.DeviceId)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.Unregister(tt.args.deviceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceLinksProjection_Models(t *testing.T) {
	type args struct {
		deviceID string
	}
	tests := []struct {
		name string
		args args
		want []eventstore.Model
	}{
		{
			name: "valid dev1",
			args: args{
				deviceID: d1res1.DeviceId,
			},
			want: []eventstore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{
						d1res1.Href: &d1res1,
						d1res3.Href: &d1res3,
						d1res4.Href: &d1res4,
						d1res5.Href: &d1res5,
					},
					DeviceId: d1res1.DeviceId,
					EventMetadata: &events.EventMetadata{
						Version:      3,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
				},
			},
		},
		{
			name: "valid dev2",
			args: args{
				deviceID: d2res1.DeviceId,
			},
			want: []eventstore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{
						d2res1.Href: &d2res1,
					},
					DeviceId: d2res1.DeviceId,
					EventMetadata: &events.EventMetadata{
						Version:      6,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
				},
			},
		},
		{
			name: "valid dev3",
			args: args{
				deviceID: d3res2.DeviceId,
			},
			want: []eventstore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{
						d3res2.Href: &d3res2,
					},
					DeviceId: d3res2.DeviceId,
					EventMetadata: &events.EventMetadata{
						Version:      1,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
				},
			},
		},
		{
			name: "valid dev4",
			args: args{
				deviceID: d4res1.DeviceId,
			},
			want: []eventstore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{},
					DeviceId:  d4res1.DeviceId,
					EventMetadata: &events.EventMetadata{
						Version:      1,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
				},
			},
		},
		{
			name: "valid dev5",
			args: args{
				deviceID: d5res1.DeviceId,
			},
			want: []eventstore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{},
					DeviceId:  d5res1.DeviceId,
					EventMetadata: &events.EventMetadata{
						Version:      1,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
				},
			},
		},
	}

	eventstore := prepareResourceLinksEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context, groupID, aggregateID string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = p.Register(ctx, tt.args.deviceID)
			assert.NoError(t, err)
			got := p.Models(commands.NewResourceID(tt.args.deviceID, commands.ResourceLinksHref))

			mapWant := make(map[string]*events.ResourceLinksSnapshotTaken)
			for _, r := range tt.want {
				m := r.(*events.ResourceLinksSnapshotTaken)
				mapWant[m.GetDeviceId()] = m
			}
			mapGot := make(map[string]*events.ResourceLinksSnapshotTaken)
			for _, r := range got {
				m := r.(*events.ResourceLinksSnapshotTaken)
				mapGot[m.GetDeviceId()] = m
			}

			assert.Equal(t, mapWant, mapGot)
		})
	}
}

func TestResourceStateProjection_Models(t *testing.T) {
	type args struct {
		resourceID *commands.ResourceId
	}
	tests := []struct {
		name string
		args args
		want []eventstore.Model
	}{
		{
			name: "valid dev1r1",
			args: args{
				resourceID: commands.NewResourceID(d1res1.DeviceId, d1res1.Href),
			},
			want: []eventstore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res1.DeviceId, d1res1.Href),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							TimestampMs: 12345,
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      4,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "3",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{
						{
							ResourceId: commands.NewResourceID(d1res1.DeviceId, d1res1.Href),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "2",
							},
							EventMetadata: &events.EventMetadata{
								Version:      2,
								TimestampMs:  12345,
								ConnectionId: "a",
							},
						},
						{
							ResourceId: commands.NewResourceID(d1res1.DeviceId, d1res1.Href),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "3",
							},
							EventMetadata: &events.EventMetadata{
								Version:      4,
								TimestampMs:  12345,
								ConnectionId: "a",
							},
						},
					},
				},
			},
		},
		{
			name: "valid dev1r2",
			args: args{
				resourceID: commands.NewResourceID(d1res2.DeviceId, d1res2.Href),
			},
			want: []eventstore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res2.DeviceId, d1res2.Href),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							TimestampMs: 12345,
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      0,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
				},
			},
		},
		{
			name: "valid dev1r3",
			args: args{
				resourceID: commands.NewResourceID(d1res3.DeviceId, d1res3.Href),
			},
			want: []eventstore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res3.DeviceId, d1res3.Href),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							TimestampMs: 12345,
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      1,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "3",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{
						{
							ResourceId: commands.NewResourceID(d1res3.DeviceId, d1res3.Href),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "3",
							},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								TimestampMs:  12345,
								ConnectionId: "a",
							},
						},
					},
				},
			},
		},

		{
			name: "valid dev1r4",
			args: args{
				resourceID: commands.NewResourceID(d1res4.DeviceId, d1res4.Href),
			},
			want: []eventstore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res4.DeviceId, d1res4.Href),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							TimestampMs: 12345,
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      5,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "4",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{
						{
							ResourceId: commands.NewResourceID(d1res4.DeviceId, d1res4.Href),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "4",
							},
							EventMetadata: &events.EventMetadata{
								Version:      5,
								TimestampMs:  12345,
								ConnectionId: "a",
							},
						},
					},
				},
			},
		},
		{
			name: "valid dev2r1",
			args: args{
				resourceID: commands.NewResourceID(d2res1.DeviceId, d2res1.Href),
			},
			want: []eventstore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d2res1.DeviceId, d2res1.Href),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							TimestampMs: 12345,
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      2,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "1",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{},
				},
			},
		},
		{
			name: "valid dev2r2",
			args: args{
				resourceID: commands.NewResourceID(d2res2.DeviceId, d2res2.Href),
			},
			want: []eventstore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d2res2.DeviceId, d2res2.Href),
					LatestResourceChange: &events.ResourceChanged{
						ResourceId:   commands.NewResourceID(d2res2.DeviceId, d2res2.Href),
						Content:      &commands.Content{},
						AuditContext: &commands.AuditContext{UserId: "userId", CorrelationId: "2"},
						EventMetadata: &events.EventMetadata{
							Version:      3,
							TimestampMs:  12345,
							ConnectionId: "a",
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      3,
						TimestampMs:  12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "1",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{},
				},
			},
		},
	}

	eventstore := prepareResourceStateEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context, groupID, aggregateID string) (cqrsEventStore.Model, error) {
			return events.NewResourceStateSnapshotTaken(), nil
		},
	)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = p.Register(ctx, tt.args.resourceID.GetDeviceId())
			assert.NoError(t, err)
			got := p.Models(tt.args.resourceID)

			mapWant := make(map[string]*events.ResourceStateSnapshotTaken)
			for _, r := range tt.want {
				m := r.(*events.ResourceStateSnapshotTaken)
				mapWant[m.GroupId()] = m
			}
			mapGot := make(map[string]*events.ResourceStateSnapshotTaken)
			for _, r := range got {
				m := r.(*events.ResourceStateSnapshotTaken)
				mapGot[m.GroupId()] = m
			}

			assert.Equal(t, mapWant, mapGot)
		})
	}
}

func TestResourceProjection_ForceUpdate(t *testing.T) {
	type args struct {
		resourceID *commands.ResourceId
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				resourceID: commands.NewResourceID(d1res1.DeviceId, d1res1.Href),
			},
		},
		{
			name: "error",
			args: args{
				resourceID: &commands.ResourceId{},
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceStateEventstore(t)
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(ctx context.Context, groupID, aggregateID string) (cqrsEventStore.Model, error) {
			return events.NewResourceStateSnapshotTaken(), nil
		},
	)
	assert.NoError(t, err)
	_, err = p.Register(ctx, d1res1.DeviceId)
	assert.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ForceUpdate(ctx, tt.args.resourceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
