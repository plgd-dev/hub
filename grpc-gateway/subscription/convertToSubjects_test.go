package subscription

import (
	"sort"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/require"
)

func TestConvertToSubjects(t *testing.T) {
	resourceID := commands.NewResourceID("a", "/light/2")
	type args struct {
		owner         string
		req           *pb.SubscribeToEvents_CreateSubscription
		leadRTEnabled bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "all",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{},
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwner+".>", isEvents.WithOwner("")),
			},
		},
		{
			name: "all - DeviceIdFilter *",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter: []string{"*", "*"},
				},
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwner+".>", isEvents.WithOwner("")),
			},
		},
		{
			name: "all - HrefFilter *",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					HrefFilter: []string{"*", "*"},
				},
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwner+".>", isEvents.WithOwner("")),
			},
		},
		{
			name: "all - ResourceIdFilter *",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID("*", "*"),
						},
						{
							ResourceId: commands.NewResourceID("*", "*"),
						},
					},
				},
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwner+".>", isEvents.WithOwner("")),
			},
		},
		{
			name: "all - All filters *",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter: []string{"*", "*"},
					HrefFilter:     []string{"*", "*"},
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID("*", "*"),
						},
						{
							ResourceId: commands.NewResourceID("*", "*"),
						},
					},
					HttpResourceIdFilter: []string{"*/*", "*/*"},
				},
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwner+".>", isEvents.WithOwner("")),
			},
		},
		{
			name: "devices registrations",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
					},
				},
				owner: "a",
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("a")),
			},
		},
		{
			name: "device registration",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter: []string{"a"},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
					},
				},
				owner: "b",
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("b")),
			},
		},
		{
			name: "devices metadata",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
				},
			},
			want: []string{
				isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadata+".>", isEvents.WithOwner(""), utils.WithDeviceID("*")),
			},
		},
		{
			name: "device metadata",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter: []string{"a"},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
				},
			},
			want: []string{
				isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadata+".>", isEvents.WithOwner(""), utils.WithDeviceID("a")),
			},
		},
		{
			name: "device and href",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: resourceID}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED,
						pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner: "c",
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", resourceID, (&events.ResourceChanged{}).EventType(), false)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "device and href (leadRTEnabled=true)",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: resourceID}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED,
						pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner:         "c",
				leadRTEnabled: true,
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", resourceID, (&events.ResourceChanged{}).EventType(), true)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "href",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					HrefFilter: []string{resourceID.GetHref()},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner: "c",
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID("*"), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", commands.NewResourceID("*", resourceID.GetHref()), (&events.ResourceChanged{}).EventType(), false)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "href (leadRTEnabled=true)",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					HrefFilter: []string{resourceID.GetHref()},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner:         "c",
				leadRTEnabled: true,
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID("*"), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", commands.NewResourceID("*", resourceID.GetHref()), (&events.ResourceChanged{}).EventType(), true)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "href (resourceID)",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: commands.NewResourceID("*", resourceID.GetHref())}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner: "c",
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID("*"), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", commands.NewResourceID("*", resourceID.GetHref()), (&events.ResourceChanged{}).EventType(), false)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "href (resourceID, leadRTEnabled=true)",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: commands.NewResourceID("*", resourceID.GetHref())}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner:         "c",
				leadRTEnabled: true,
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID("*"), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", commands.NewResourceID("*", resourceID.GetHref()), (&events.ResourceChanged{}).EventType(), true)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "device and resourceID",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter:   []string{resourceID.GetDeviceId()},
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: resourceID}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner: "c",
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", commands.NewResourceID(resourceID.GetDeviceId(), "*"), (&events.ResourceChanged{}).EventType(), false)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "device and resourceID (leadRTEnabled=true)",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter:   []string{resourceID.GetDeviceId()},
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: resourceID}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner:         "c",
				leadRTEnabled: true,
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", commands.NewResourceID(resourceID.GetDeviceId(), "*"), (&events.ResourceChanged{}).EventType(), true)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "device, href and resourceID",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter:   []string{resourceID.GetDeviceId()},
					HrefFilter:       []string{resourceID.GetHref()},
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: resourceID}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner: "c",
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", resourceID, (&events.ResourceChanged{}).EventType(), false)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
		{
			name: "device, href and resourceID (leadRTEnabled=true)",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter:   []string{resourceID.GetDeviceId()},
					HrefFilter:       []string{resourceID.GetHref()},
					ResourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: resourceID}},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
				owner:         "c",
				leadRTEnabled: true,
			},
			want: func() []string {
				subjects := []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID(resourceID.GetDeviceId()), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}
				subjects = append(subjects, utils.GetResourceEventSubjects("c", resourceID, (&events.ResourceChanged{}).EventType(), true)...)
				return append(subjects, isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")))
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, bitmask := getFilters(tt.args.req, tt.args.leadRTEnabled)
			got := ConvertToSubjects(tt.args.owner, filters, bitmask)
			sort.Strings(got)
			require.Equal(t, tt.want, got)
		})
	}
}
