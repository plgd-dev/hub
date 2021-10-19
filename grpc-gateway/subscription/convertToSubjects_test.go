package subscription

import (
	"sort"
	"testing"

	isEvents "github.com/plgd-dev/hub/identity-store/events"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	kitStrings "github.com/plgd-dev/kit/v2/strings"
	"github.com/stretchr/testify/require"
)

func TestConvertToSubjects(t *testing.T) {
	resourceID := commands.NewResourceID("a", "/light/2").ToUUID()
	type args struct {
		owner             string
		filterDeviceIDs   kitStrings.Set
		filterResourceIDs kitStrings.Set
		bitmask           FilterBitmask
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty",
		},
		{
			name: "all",
			args: args{
				bitmask: FilterBitmaskMax,
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwner+".>", isEvents.WithOwner("")),
			},
		},
		{
			name: "devices registrations",
			args: args{
				bitmask: FilterBitmaskRegistrations,
				owner:   "a",
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("a")),
			},
		},
		{
			name: "device registration",
			args: args{
				bitmask:         FilterBitmaskRegistrations,
				filterDeviceIDs: kitStrings.MakeSet("a"),
				owner:           "b",
			},
			want: []string{
				isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("b")),
			},
		},
		{
			name: "devices metadata",
			args: args{
				bitmask: FilterBitmaskDeviceMetadata,
			},
			want: []string{
				isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadata+".>", isEvents.WithOwner(""), utils.WithDeviceID("*")),
			},
		},
		{
			name: "device metadata",
			args: args{
				bitmask:         FilterBitmaskDeviceMetadata,
				filterDeviceIDs: kitStrings.MakeSet("a"),
			},
			want: []string{
				isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadata+".>", isEvents.WithOwner(""), utils.WithDeviceID("a")),
			},
		},
		{
			name: "custom",
			args: args{
				bitmask:           FilterBitmaskDeviceMetadataUpdated | FilterBitmaskDeviceRegistered | FilterBitmaskDeviceUnregistered | FilterBitmaskResourceChanged,
				filterResourceIDs: kitStrings.MakeSet(resourceID),
				owner:             "c",
			},
			want: []string{
				isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner("c"), utils.WithDeviceID("*"), isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType())),
				isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithOwner("c"), utils.WithDeviceID("*"), utils.WithResourceId(resourceID), isEvents.WithEventType((&events.ResourceChanged{}).EventType())),
				isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrations+".>", isEvents.WithOwner("c")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertToSubjects(tt.args.owner, tt.args.filterDeviceIDs, tt.args.filterResourceIDs, tt.args.bitmask)
			sort.Strings(got)
			require.Equal(t, tt.want, got)
		})
	}
}
