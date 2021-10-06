package subscription

import (
	"testing"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/stretchr/testify/require"
)

func TestEventsFilterToBitmask(t *testing.T) {
	type args struct {
		commandFilter []pb.SubscribeToEvents_CreateSubscription_Event
	}
	tests := []struct {
		name string
		args args
		want FilterBitmask
	}{
		{
			name: "Empty filter",
			want: FilterBitmask(0xffffffff),
		},
		{
			name: "Single event",
			args: args{
				commandFilter: []pb.SubscribeToEvents_CreateSubscription_Event{pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED},
			},
			want: FilterBitmaskResourceChanged,
		},
		{
			name: "All events",
			args: args{
				commandFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_REGISTERED,
					pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
				},
			},
			want: FilterBitmaskResourceCreatePending |
				FilterBitmaskResourceCreated |
				FilterBitmaskResourceRetrievePending |
				FilterBitmaskResourceRetrieved |
				FilterBitmaskResourceUpdatePending |
				FilterBitmaskResourceUpdated |
				FilterBitmaskResourceDeletePending |
				FilterBitmaskResourceDeleted |
				FilterBitmaskDeviceMetadataUpdatePending |
				FilterBitmaskDeviceMetadataUpdated |
				FilterBitmaskDeviceRegistered |
				FilterBitmaskDeviceUnregistered |
				FilterBitmaskResourceChanged |
				FilterBitmaskResourcesPublished |
				FilterBitmaskResourcesUnpublished,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EventsFilterToBitmask(tt.args.commandFilter)
			require.Equal(t, tt.want, got)
		})
	}
}
