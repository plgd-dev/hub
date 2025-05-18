package projection

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsEventStore "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	mockEvents "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dev1 = test.GenerateDeviceIDbyIdx(1)
	dev2 = test.GenerateDeviceIDbyIdx(2)
)

var d1res1 = commands.Resource{
	DeviceId:      dev1,
	Href:          "/res1",
	ResourceTypes: []string{"typeDev1Res"},
}

var d1res2 = commands.Resource{
	DeviceId:      dev1,
	Href:          "/res2",
	ResourceTypes: []string{"typeDev1Res2"},
}

var d1res3 = commands.Resource{
	DeviceId:      dev1,
	Href:          "/res3",
	ResourceTypes: []string{"typeDev1Res3"},
}

var d1res4 = commands.Resource{
	DeviceId:      dev1,
	Href:          "/res4",
	ResourceTypes: []string{"typeDev1Res4"},
}

var d1res5 = commands.Resource{
	DeviceId:      dev1,
	Href:          "/res5",
	ResourceTypes: []string{"typeDev1Res5"},
}

var d2res1 = commands.Resource{
	DeviceId:      dev2,
	Href:          "/res1",
	ResourceTypes: []string{"typeDev2Res1"},
}

var d2res2 = commands.Resource{
	DeviceId:      dev2,
	Href:          "/res2",
	ResourceTypes: []string{"typeDev2Res2"},
}

var d3res1 = commands.Resource{
	DeviceId:      "dev3",
	Href:          "/res1",
	ResourceTypes: []string{"typeDev3Res1"},
}

var d3res2 = commands.Resource{
	DeviceId:      "dev3",
	Href:          "/res2",
	ResourceTypes: []string{"typeDev3Res2"},
}

var d4res1 = commands.Resource{
	DeviceId:      "dev4",
	Href:          "/res1",
	ResourceTypes: []string{"typeDev4Res1"},
}

var d4res2 = commands.Resource{
	DeviceId:      "dev4",
	Href:          "/res2",
	ResourceTypes: []string{"typeDev4Res2"},
}

var d5res1 = commands.Resource{
	DeviceId:      "dev5",
	Href:          "/res1",
	ResourceTypes: []string{"typeDev5Res1"},
}

var d5res2 = commands.Resource{
	DeviceId:      "dev5",
	Href:          "/res2",
	ResourceTypes: []string{"typeDev5Res2"},
}

func makeEventMeta(connectionID string, version uint64) *events.EventMetadata {
	e := events.MakeEventMeta(connectionID, 0, version, "")
	e.Timestamp = 12345
	return e
}

func prepareResourceLinksEventstore() *mockEvents.MockEventStore {
	eventstore := mockEvents.NewMockEventStore()

	d1resID := commands.MakeLinksResourceUUID(d1res1.GetDeviceId())
	eventstore.Append(d1res1.GetDeviceId(), d1resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d1res1}, d1res1.GetDeviceId(), makeEventMeta("a", 0)))
	eventstore.Append(d1res2.GetDeviceId(), d1resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d1res2, &d1res3}, d1res2.GetDeviceId(), makeEventMeta("a", 1)))
	eventstore.Append(d1res2.GetDeviceId(), d1resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{d1res2.GetHref()}, d1res2.GetDeviceId(), makeEventMeta("a", 2)))
	eventstore.Append(d1res4.GetDeviceId(), d1resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d1res4, &d1res5}, d1res4.GetDeviceId(), makeEventMeta("a", 3)))

	d2resID := commands.MakeLinksResourceUUID(d2res1.GetDeviceId())
	eventstore.Append(d2res1.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d2res1, &d2res2}, d2res1.GetDeviceId(), makeEventMeta("a", 0)))
	eventstore.Append(d2res1.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res1.GetHref()}, d2res1.GetDeviceId(), makeEventMeta("a", 1)))
	eventstore.Append(d2res2.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res2.GetHref()}, d2res2.GetDeviceId(), makeEventMeta("a", 2)))
	eventstore.Append(d2res2.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res2.GetHref()}, d2res2.GetDeviceId(), makeEventMeta("a", 3)))
	eventstore.Append(d2res1.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d2res1, &d2res1}, d2res1.GetDeviceId(), makeEventMeta("a", 4)))
	eventstore.Append(d2res2.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{d2res2.GetHref(), d2res2.GetHref()}, d2res2.GetDeviceId(), makeEventMeta("a", 5)))
	eventstore.Append(d2res1.GetDeviceId(), d2resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d2res1}, d2res1.GetDeviceId(), makeEventMeta("a", 6)))

	d3resID := commands.MakeLinksResourceUUID(d3res1.GetDeviceId())
	eventstore.Append(d3res1.GetDeviceId(), d3resID.String(), mockEvents.MakeResourceLinksSnapshotTaken(map[string]*commands.Resource{d3res1.GetHref(): &d3res1, d3res2.GetHref(): &d3res2}, d3res1.GetDeviceId(), makeEventMeta("a", 0)))
	eventstore.Append(d3res1.GetDeviceId(), d3resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{d3res1.GetHref()}, d3res1.GetDeviceId(), makeEventMeta("a", 1)))

	d4resID := commands.MakeLinksResourceUUID(d4res1.GetDeviceId())
	eventstore.Append(d4res1.GetDeviceId(), d4resID.String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&d4res1, &d4res2}, d4res1.GetDeviceId(), makeEventMeta("a", 0)))
	eventstore.Append(d4res1.GetDeviceId(), d4resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{}, d4res1.GetDeviceId(), makeEventMeta("a", 1)))

	d5resID := commands.MakeLinksResourceUUID(d5res1.GetDeviceId())
	eventstore.Append(d5res1.GetDeviceId(), d5resID.String(), mockEvents.MakeResourceLinksSnapshotTaken(map[string]*commands.Resource{d5res1.GetHref(): &d5res1, d5res2.GetHref(): &d5res2}, d5res1.GetDeviceId(), makeEventMeta("a", 0)))
	eventstore.Append(d5res1.GetDeviceId(), d5resID.String(), mockEvents.MakeResourceLinksUnpublishedEvent([]string{}, d5res1.GetDeviceId(), makeEventMeta("a", 1)))

	return eventstore
}

func prepareResourceStateEventstore() *mockEvents.MockEventStore {
	eventstore := mockEvents.NewMockEventStore()
	resourceChangedEventMetadata := makeEventMeta("", 0)

	d1r1 := commands.NewResourceID(d1res1.GetDeviceId(), d1res1.GetHref())
	eventstore.Append(d1res1.GetDeviceId(), d1r1.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d1r1, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata, ResourceTypes: d1res1.GetResourceTypes()}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "0"), d1res1.GetResourceTypes()))
	eventstore.Append(d1res1.GetDeviceId(), d1r1.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r1, &commands.Content{}, makeEventMeta("a", 1), mockEvents.MakeAuditContext("userId", "1"), time.Time{}, d1res1.GetResourceTypes()))
	eventstore.Append(d1res1.GetDeviceId(), d1r1.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r1, &commands.Content{}, makeEventMeta("a", 2), mockEvents.MakeAuditContext("userId", "2"), time.Time{}, d1res1.GetResourceTypes()))
	eventstore.Append(d1res1.GetDeviceId(), d1r1.ToUUID().String(), mockEvents.MakeResourceUpdated(d1r1, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 3), mockEvents.MakeAuditContext("userId", "1"), d1res1.GetResourceTypes()))
	eventstore.Append(d1res1.GetDeviceId(), d1r1.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r1, &commands.Content{}, makeEventMeta("a", 4), mockEvents.MakeAuditContext("userId", "3"), time.Time{}, d1res1.GetResourceTypes()))

	d1r2 := commands.NewResourceID(d1res2.GetDeviceId(), d1res2.GetHref())
	eventstore.Append(d1res2.GetDeviceId(), d1r2.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d1r2, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata, ResourceTypes: d1res2.GetResourceTypes()}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "2"), d1res2.GetResourceTypes()))

	d1r3 := commands.NewResourceID(d1res3.GetDeviceId(), d1res3.GetHref())
	eventstore.Append(d1res3.GetDeviceId(), d1r3.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d1r3, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata, ResourceTypes: d1res3.GetResourceTypes()}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "2"), d1res3.GetResourceTypes()))
	eventstore.Append(d1res3.GetDeviceId(), d1r3.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r3, &commands.Content{}, makeEventMeta("a", 1), mockEvents.MakeAuditContext("userId", "3"), time.Time{}, d1res3.GetResourceTypes()))

	d1r4 := commands.NewResourceID(d1res4.GetDeviceId(), d1res4.GetHref())
	eventstore.Append(d1res4.GetDeviceId(), d1r4.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d1r4, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata, ResourceTypes: d1res4.GetResourceTypes()}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "0"), d1res4.GetResourceTypes()))
	eventstore.Append(d1res4.GetDeviceId(), d1r4.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r4, &commands.Content{}, makeEventMeta("a", 1), mockEvents.MakeAuditContext("userId", "1"), time.Time{}, d1res4.GetResourceTypes()))
	eventstore.Append(d1res4.GetDeviceId(), d1r4.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r4, &commands.Content{}, makeEventMeta("a", 2), mockEvents.MakeAuditContext("userId", "2"), time.Time{}, d1res4.GetResourceTypes()))
	eventstore.Append(d1res4.GetDeviceId(), d1r4.ToUUID().String(), mockEvents.MakeResourceUpdated(d1r4, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 3), mockEvents.MakeAuditContext("userId", "1"), d1res4.GetResourceTypes()))
	eventstore.Append(d1res4.GetDeviceId(), d1r4.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d1r4, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 4), mockEvents.MakeAuditContext("userId", "3"), d1res4.GetResourceTypes()))
	eventstore.Append(d1res4.GetDeviceId(), d1r4.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d1r4, &commands.Content{}, makeEventMeta("a", 5), mockEvents.MakeAuditContext("userId", "4"), time.Time{}, d1res4.GetResourceTypes()))

	d2r1 := commands.NewResourceID(d2res1.GetDeviceId(), d2res1.GetHref())
	eventstore.Append(d2res1.GetDeviceId(), d2r1.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d2r1, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata, ResourceTypes: d2res1.GetResourceTypes()}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "0"), d2res1.GetResourceTypes()))
	eventstore.Append(d2res1.GetDeviceId(), d2r1.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d2r1, &commands.Content{}, makeEventMeta("a", 1), mockEvents.MakeAuditContext("userId", "1"), time.Time{}, d2res1.GetResourceTypes()))
	eventstore.Append(d2res1.GetDeviceId(), d2r1.ToUUID().String(), mockEvents.MakeResourceUpdated(d2r1, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 2), mockEvents.MakeAuditContext("userId", "1"), d2res1.GetResourceTypes()))

	d2r2 := commands.NewResourceID(d2res2.GetDeviceId(), d2res2.GetHref())
	eventstore.Append(d2res2.GetDeviceId(), d2r2.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(d2r2, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata, ResourceTypes: d2res2.GetResourceTypes()}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "0"), d2res2.GetResourceTypes()))
	eventstore.Append(d2res2.GetDeviceId(), d2r2.ToUUID().String(), mockEvents.MakeResourceUpdatePending(d2r2, &commands.Content{}, makeEventMeta("a", 1), mockEvents.MakeAuditContext("userId", "1"), time.Time{}, d2res2.GetResourceTypes()))
	eventstore.Append(d2res2.GetDeviceId(), d2r2.ToUUID().String(), mockEvents.MakeResourceUpdated(d2r2, commands.Status_OK, &commands.Content{}, makeEventMeta("a", 2), mockEvents.MakeAuditContext("userId", "1"), d2res2.GetResourceTypes()))
	eventstore.Append(d2res2.GetDeviceId(), d2r2.ToUUID().String(), mockEvents.MakeResourceChangedEvent(d2r2, &commands.Content{}, makeEventMeta("a", 3), mockEvents.MakeAuditContext("userId", "2"), d2res2.GetResourceTypes()))

	return eventstore
}

func TestResourceProjectionTestLoadParallelModels(t *testing.T) {
	eventstore := mockEvents.NewMockEventStore()
	resourceChangedEventMetadata := makeEventMeta("", 0)
	numDevices := 350
	numResources := 500
	numParallelRequests := 6

	for i := range numDevices {
		for j := range numResources {
			resourceID := commands.NewResourceID(fmt.Sprintf("dev-%v", i), fmt.Sprintf("res-%v", j))
			eventstore.Append(resourceID.GetDeviceId(), resourceID.ToUUID().String(), mockEvents.MakeResourceStateSnapshotTaken(resourceID, &events.ResourceChanged{Content: &commands.Content{}, EventMetadata: resourceChangedEventMetadata}, makeEventMeta("a", 0), mockEvents.MakeAuditContext("userId", "2"), []string{"type1", "type2"}))
		}
	}

	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(_ context.Context, _, _ string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	require.NoError(t, err)

	err = p.cqrsProjection.Project(ctx, nil)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(numParallelRequests)
	t.Logf("starting %v requests\n", numParallelRequests)
	for i := range numParallelRequests {
		go func(v int) {
			defer wg.Done()
			n := time.Now()
			p.cqrsProjection.Models(nil, func(cqrsEventStore.Model) (wantNext bool) { return true })
			diff := -1 * time.Until(n)
			t.Logf("%v models time %v\n", v, diff)
		}(i)
	}
	wg.Wait()
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
				deviceID: d1res1.GetDeviceId(),
			},
			wantLoaded: true,
		},
		{
			name: "second valid",
			args: args{
				deviceID: d1res1.GetDeviceId(),
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

	eventstore := prepareResourceLinksEventstore()
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(_ context.Context, _, _ string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLoaded, err := p.Register(ctx, tt.args.deviceID)
			if tt.wantLoaded {
				assert.True(t, gotLoaded)
			} else {
				assert.False(t, gotLoaded)
			}
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
				deviceID: d1res1.GetDeviceId(),
			},
		},
		{
			name: "second time",
			args: args{
				deviceID: d1res1.GetDeviceId(),
			},
		},
		{
			name: "third error",
			args: args{
				deviceID: d1res1.GetDeviceId(),
			},
			wantErr: true,
		},
		{
			name: "not registered",
			args: args{
				deviceID: d2res1.GetDeviceId(),
			},
			wantErr: true,
		},
	}

	eventstore := prepareResourceLinksEventstore()
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(_ context.Context, _, _ string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	require.NoError(t, err)
	_, err = p.Register(ctx, d1res1.GetDeviceId())
	require.NoError(t, err)
	_, err = p.Register(ctx, d1res1.GetDeviceId())
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.Unregister(tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
		want []cqrsEventStore.Model
	}{
		{
			name: "valid dev1",
			args: args{
				deviceID: d1res1.GetDeviceId(),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{
						d1res1.GetHref(): &d1res1,
						d1res3.GetHref(): &d1res3,
						d1res4.GetHref(): &d1res4,
						d1res5.GetHref(): &d1res5,
					},
					DeviceId: d1res1.GetDeviceId(),
					EventMetadata: &events.EventMetadata{
						Version:      3,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: mockEvents.StaticAuditContext,
				},
			},
		},
		{
			name: "valid dev2",
			args: args{
				deviceID: d2res1.GetDeviceId(),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{
						d2res1.GetHref(): &d2res1,
					},
					DeviceId: d2res1.GetDeviceId(),
					EventMetadata: &events.EventMetadata{
						Version:      6,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: mockEvents.StaticAuditContext,
				},
			},
		},
		{
			name: "valid dev3",
			args: args{
				deviceID: d3res2.GetDeviceId(),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{
						d3res2.GetHref(): &d3res2,
					},
					DeviceId: d3res2.GetDeviceId(),
					EventMetadata: &events.EventMetadata{
						Version:      1,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: mockEvents.StaticAuditContext,
				},
			},
		},
		{
			name: "valid dev4",
			args: args{
				deviceID: d4res1.GetDeviceId(),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{},
					DeviceId:  d4res1.GetDeviceId(),
					EventMetadata: &events.EventMetadata{
						Version:      1,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: mockEvents.StaticAuditContext,
				},
			},
		},
		{
			name: "valid dev5",
			args: args{
				deviceID: d5res1.GetDeviceId(),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceLinksSnapshotTaken{
					Resources: map[string]*commands.Resource{},
					DeviceId:  d5res1.GetDeviceId(),
					EventMetadata: &events.EventMetadata{
						Version:      1,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: mockEvents.StaticAuditContext,
				},
			},
		},
	}

	eventstore := prepareResourceLinksEventstore()
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(_ context.Context, _, _ string) (cqrsEventStore.Model, error) {
			return events.NewResourceLinksSnapshotTaken(), nil
		},
	)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = p.Register(ctx, tt.args.deviceID)
			require.NoError(t, err)
			got := []cqrsEventStore.Model{}
			p.Models(func(m cqrsEventStore.Model) (wantNext bool) {
				got = append(got, m)
				return true
			}, commands.NewResourceID(tt.args.deviceID, commands.ResourceLinksHref))

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

func TestResourceStateProjectionModels(t *testing.T) {
	type args struct {
		resourceID *commands.ResourceId
	}
	tests := []struct {
		name string
		args args
		want []cqrsEventStore.Model
	}{
		{
			name: "valid dev1r1",
			args: args{
				resourceID: commands.NewResourceID(d1res1.GetDeviceId(), d1res1.GetHref()),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res1.GetDeviceId(), d1res1.GetHref()),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							Timestamp: 12345,
						},
						ResourceTypes: d1res1.GetResourceTypes(),
					},
					EventMetadata: &events.EventMetadata{
						Version:      4,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "3",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{
						{
							ResourceId: commands.NewResourceID(d1res1.GetDeviceId(), d1res1.GetHref()),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "2",
							},
							EventMetadata: &events.EventMetadata{
								Version:      2,
								Timestamp:    12345,
								ConnectionId: "a",
							},
							ResourceTypes: d1res1.GetResourceTypes(),
						},
						{
							ResourceId: commands.NewResourceID(d1res1.GetDeviceId(), d1res1.GetHref()),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "3",
							},
							EventMetadata: &events.EventMetadata{
								Version:      4,
								Timestamp:    12345,
								ConnectionId: "a",
							},
							ResourceTypes: d1res1.GetResourceTypes(),
						},
					},
					ResourceTypes: d1res1.GetResourceTypes(),
				},
			},
		},
		{
			name: "valid dev1r2",
			args: args{
				resourceID: commands.NewResourceID(d1res2.GetDeviceId(), d1res2.GetHref()),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res2.GetDeviceId(), d1res2.GetHref()),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							Timestamp: 12345,
						},
						ResourceTypes: d1res2.GetResourceTypes(),
					},
					EventMetadata: &events.EventMetadata{
						Version:      0,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext:  commands.NewAuditContext("userId", "2", ""),
					ResourceTypes: d1res2.GetResourceTypes(),
				},
			},
		},
		{
			name: "valid dev1r3",
			args: args{
				resourceID: commands.NewResourceID(d1res3.GetDeviceId(), d1res3.GetHref()),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res3.GetDeviceId(), d1res3.GetHref()),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							Timestamp: 12345,
						},
						ResourceTypes: d1res3.GetResourceTypes(),
					},
					EventMetadata: &events.EventMetadata{
						Version:      1,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "3",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{
						{
							ResourceId: commands.NewResourceID(d1res3.GetDeviceId(), d1res3.GetHref()),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "3",
							},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    12345,
								ConnectionId: "a",
							},
							ResourceTypes: d1res3.GetResourceTypes(),
						},
					},
					ResourceTypes: d1res3.GetResourceTypes(),
				},
			},
		},

		{
			name: "valid dev1r4",
			args: args{
				resourceID: commands.NewResourceID(d1res4.GetDeviceId(), d1res4.GetHref()),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d1res4.GetDeviceId(), d1res4.GetHref()),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							Timestamp: 12345,
						},
					},
					EventMetadata: &events.EventMetadata{
						Version:      5,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "4",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{
						{
							ResourceId: commands.NewResourceID(d1res4.GetDeviceId(), d1res4.GetHref()),
							Content:    &commands.Content{},
							AuditContext: &commands.AuditContext{
								UserId:        "userId",
								CorrelationId: "4",
							},
							EventMetadata: &events.EventMetadata{
								Version:      5,
								Timestamp:    12345,
								ConnectionId: "a",
							},
							ResourceTypes: d1res4.GetResourceTypes(),
						},
					},
					ResourceTypes: d1res4.GetResourceTypes(),
				},
			},
		},
		{
			name: "valid dev2r1",
			args: args{
				resourceID: commands.NewResourceID(d2res1.GetDeviceId(), d2res1.GetHref()),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d2res1.GetDeviceId(), d2res1.GetHref()),
					LatestResourceChange: &events.ResourceChanged{
						Content: &commands.Content{},
						EventMetadata: &events.EventMetadata{
							Timestamp: 12345,
						},
						ResourceTypes: d2res1.GetResourceTypes(),
					},
					EventMetadata: &events.EventMetadata{
						Version:      2,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "1",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{},
					ResourceTypes:          d2res1.GetResourceTypes(),
				},
			},
		},
		{
			name: "valid dev2r2",
			args: args{
				resourceID: commands.NewResourceID(d2res2.GetDeviceId(), d2res2.GetHref()),
			},
			want: []cqrsEventStore.Model{
				&events.ResourceStateSnapshotTaken{
					ResourceId: commands.NewResourceID(d2res2.GetDeviceId(), d2res2.GetHref()),
					LatestResourceChange: &events.ResourceChanged{
						ResourceId:   commands.NewResourceID(d2res2.GetDeviceId(), d2res2.GetHref()),
						Content:      &commands.Content{},
						AuditContext: &commands.AuditContext{UserId: "userId", CorrelationId: "2"},
						EventMetadata: &events.EventMetadata{
							Version:      3,
							Timestamp:    12345,
							ConnectionId: "a",
						},
						ResourceTypes: d2res2.GetResourceTypes(),
					},
					EventMetadata: &events.EventMetadata{
						Version:      3,
						Timestamp:    12345,
						ConnectionId: "a",
					},
					AuditContext: &commands.AuditContext{
						UserId:        "userId",
						CorrelationId: "2",
					},
					ResourceUpdatePendings: []*events.ResourceUpdatePending{},
					ResourceTypes:          d2res2.GetResourceTypes(),
				},
			},
		},
	}

	eventstore := prepareResourceStateEventstore()
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(_ context.Context, _, _ string) (cqrsEventStore.Model, error) {
			return events.NewResourceStateSnapshotTaken(), nil
		},
	)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = p.Register(ctx, tt.args.resourceID.GetDeviceId())
			require.NoError(t, err)
			got := []cqrsEventStore.Model{}
			p.Models(func(m cqrsEventStore.Model) (wantNext bool) {
				got = append(got, m)
				return true
			}, tt.args.resourceID)

			mapWant := make(map[string]*events.ResourceStateSnapshotTaken)
			for _, r := range tt.want {
				m := r.(*events.ResourceStateSnapshotTaken)
				mapWant[m.GroupID()] = m
			}
			mapGot := make(map[string]*events.ResourceStateSnapshotTaken)
			for _, r := range got {
				m := r.(*events.ResourceStateSnapshotTaken)
				mapGot[m.GroupID()] = m
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
				resourceID: commands.NewResourceID(d1res1.GetDeviceId(), d1res1.GetHref()),
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

	eventstore := prepareResourceStateEventstore()
	ctx := context.Background()
	p, err := NewProjection(
		ctx,
		"test",
		eventstore,
		nil,
		func(_ context.Context, _, _ string) (cqrsEventStore.Model, error) {
			return events.NewResourceStateSnapshotTaken(), nil
		},
	)
	require.NoError(t, err)
	_, err = p.Register(ctx, d1res1.GetDeviceId())
	require.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ForceUpdate(ctx, tt.args.resourceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
