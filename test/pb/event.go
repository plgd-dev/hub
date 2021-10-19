package pb

import (
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/stretchr/testify/require"
)

func ResourceLinkToPublishEvent(deviceID, token string, links []schema.ResourceLink) *pb.Event {
	out := make([]*commands.Resource, 0, 32)
	for _, l := range links {
		link := commands.SchemaResourceLinkToResource(l, time.Time{})
		link.DeviceId = deviceID
		out = append(out, link)
	}
	return &pb.Event{
		Type: &pb.Event_ResourcePublished{
			ResourcePublished: &events.ResourceLinksPublished{
				DeviceId:  deviceID,
				Resources: out,
			},
		},
		CorrelationId: token,
	}
}

func CmpEvents(t *testing.T, expected, got []*pb.Event) {
	require.Len(t, got, len(expected))
	cleanUpResourceLinksPublished := func(ev *pb.Event) {
		if ev.GetResourcePublished() != nil {
			test.CleanUpResourceLinksPublished(ev.GetResourcePublished())
		}
	}
	cleanUpDeviceMetadata := func(ev *pb.Event) {
		if ev.GetDeviceMetadataUpdated() != nil {
			ev.GetDeviceMetadataUpdated().EventMetadata = nil
			ev.GetDeviceMetadataUpdated().AuditContext = nil
			if ev.GetDeviceMetadataUpdated().GetStatus() != nil {
				ev.GetDeviceMetadataUpdated().GetStatus().ValidUntil = 0
			}
		}
	}

	// normalize
	for i := range expected {
		expected[i].SubscriptionId = ""
		got[i].SubscriptionId = ""
		cleanUpResourceLinksPublished(expected[i])
		cleanUpResourceLinksPublished(got[i])
		cleanUpDeviceMetadata(expected[i])
		cleanUpDeviceMetadata(got[i])
	}

	// compare
	for _, gotV := range got {
		test.CheckProtobufs(t, expected, gotV, test.RequireToCheckFunc(require.Contains))
	}
	for _, expectedV := range expected {
		test.CheckProtobufs(t, got, expectedV, test.RequireToCheckFunc(require.Contains))
	}
}

func CmpEventResourceCreatePending(t *testing.T, expected, got *pb.Event) {
	require.NotNil(t, expected.GetResourceCreatePending())
	e := expected.GetResourceCreatePending()
	require.NotNil(t, got.GetResourceCreatePending())
	g := got.GetResourceCreatePending()

	cleanupAuditContext := func(ev *events.ResourceCreatePending) {
		if ev.GetAuditContext() != nil {
			ev.GetAuditContext().CorrelationId = ""
		}
	}
	cleanupAuditContext(e)
	cleanupAuditContext(g)

	e.EventMetadata = nil
	g.EventMetadata = nil

	expectedData := test.DecodeCbor(t, e.GetContent().GetData())
	gotData := test.DecodeCbor(t, g.GetContent().GetData())
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpEventResourceCreated(t *testing.T, expected, got *pb.Event) {
	require.NotNil(t, expected.GetResourceCreated())
	e := expected.GetResourceCreated()
	require.NotNil(t, got.GetResourceCreated())
	g := got.GetResourceCreated()

	cleanupAuditContext := func(ev *events.ResourceCreated) {
		if ev.GetAuditContext() != nil {
			ev.GetAuditContext().CorrelationId = ""
		}
	}
	cleanupAuditContext(e)
	cleanupAuditContext(g)

	e.EventMetadata = nil
	g.EventMetadata = nil

	expectedData, ok := test.DecodeCbor(t, e.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	gotData, ok := test.DecodeCbor(t, g.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	delete(expectedData, "ins")
	delete(gotData, "ins")
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpEventDeviceMetadataUpdated(t *testing.T, expected, got *pb.Event) {
	require.NotNil(t, expected.GetDeviceMetadataUpdated())
	e := expected.GetDeviceMetadataUpdated()
	require.NotNil(t, got.GetDeviceMetadataUpdated())
	g := got.GetDeviceMetadataUpdated()

	cleanup := func(ev *events.DeviceMetadataUpdated) {
		ev.EventMetadata = nil
		ev.AuditContext = nil
		if ev.GetStatus() != nil {
			ev.GetStatus().ValidUntil = 0
		}
	}
	cleanup(e)
	cleanup(g)

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

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
