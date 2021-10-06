package test

import (
	"reflect"
	"testing"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/stretchr/testify/require"
)

//gocyclo:ignore
func GetWrappedEvent(value *pb.GetEventsResponse) interface{} {
	if event := value.GetDeviceMetadataSnapshotTaken(); event != nil {
		return event
	}
	if event := value.GetDeviceMetadataUpdatePending(); event != nil {
		return event
	}
	if event := value.GetDeviceMetadataUpdated(); event != nil {
		return event
	}
	if event := value.GetResourceChanged(); event != nil {
		return event
	}
	if event := value.GetResourceCreatePending(); event != nil {
		return event
	}
	if event := value.GetResourceCreated(); event != nil {
		return event
	}
	if event := value.GetResourceDeletePending(); event != nil {
		return event
	}
	if event := value.GetResourceDeleted(); event != nil {
		return event
	}
	if event := value.GetResourceLinksPublished(); event != nil {
		return event
	}
	if event := value.GetResourceLinksSnapshotTaken(); event != nil {
		return event
	}
	if event := value.GetResourceLinksUnpublished(); event != nil {
		return event
	}
	if event := value.GetResourceRetrievePending(); event != nil {
		return event
	}
	if event := value.GetResourceRetrieved(); event != nil {
		return event
	}
	if event := value.GetResourceStateSnapshotTaken(); event != nil {
		return event
	}
	if event := value.GetResourceUpdatePending(); event != nil {
		return event
	}
	if event := value.GetResourceUpdated(); event != nil {
		return event
	}
	return nil
}

func CheckGetEventsResponse(t *testing.T, deviceId string, got []*pb.GetEventsResponse) {
	for _, value := range got {
		event := GetWrappedEvent(value)
		r := reflect.ValueOf(event)
		const CheckMethodName = "CheckInitialized"
		m := r.MethodByName(CheckMethodName)
		if !m.IsValid() {
			require.Failf(t, "Invalid type", "Struct %T doesn't have %v method", event, CheckMethodName)
		}
		v := m.Call([]reflect.Value{})
		require.Len(t, v, 1)
		initialized := v[0].Bool()
		require.True(t, initialized)
	}
}
