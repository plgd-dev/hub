package pb

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

func OperationProcessedOK() *pb.Event_OperationProcessed_ {
	return &pb.Event_OperationProcessed_{
		OperationProcessed: &pb.Event_OperationProcessed{
			ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
				Code: pb.Event_OperationProcessed_ErrorStatus_OK,
			},
		},
	}
}

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
				DeviceId:     deviceID,
				Resources:    out,
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
		CorrelationId: token,
	}
}

func GetEventType(ev *pb.Event) string {
	return fmt.Sprintf("%T", ev.GetType())
}

func GetEventID(ev *pb.Event) string {
	switch v := ev.GetType().(type) {
	case *pb.Event_DeviceRegistered_:
		return fmt.Sprintf("%T", ev.GetType())
	case *pb.Event_DeviceUnregistered_:
		return fmt.Sprintf("%T", ev.GetType())
	case *pb.Event_ResourcePublished:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourcePublished.GetDeviceId())
	case *pb.Event_ResourceUnpublished:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceUnpublished.GetDeviceId())
	case *pb.Event_ResourceChanged:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceChanged.GetResourceId().ToString())
	case *pb.Event_OperationProcessed_:
		return fmt.Sprintf("%T", ev.GetType())
	case *pb.Event_SubscriptionCanceled_:
		return fmt.Sprintf("%T", ev.GetType())
	case *pb.Event_ResourceUpdatePending:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceUpdatePending.GetResourceId().ToString())
	case *pb.Event_ResourceUpdated:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceUpdated.GetResourceId().ToString())
	case *pb.Event_ResourceRetrievePending:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceRetrievePending.GetResourceId().ToString())
	case *pb.Event_ResourceRetrieved:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceRetrieved.GetResourceId().ToString())
	case *pb.Event_ResourceDeletePending:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceDeletePending.GetResourceId().ToString())
	case *pb.Event_ResourceDeleted:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceDeleted.GetResourceId().ToString())
	case *pb.Event_ResourceCreatePending:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceCreatePending.GetResourceId().ToString())
	case *pb.Event_ResourceCreated:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceCreated.GetResourceId().ToString())
	case *pb.Event_DeviceMetadataUpdatePending:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.DeviceMetadataUpdatePending.GetDeviceId())
	case *pb.Event_DeviceMetadataUpdated:
		return fmt.Sprintf("%T:%v", ev.GetType(), v.DeviceMetadataUpdated.GetDeviceId())
	}
	return ""
}

// Remove fields with unpredictable values
func CleanUpEvent(t *testing.T, ev *pb.Event) {
	noop := func(ev *pb.Event) {
		// nothing to do
	}

	typeName := func(v interface{}) string {
		return reflect.TypeOf(v).String()
	}

	cleanupHandlerFn := map[string]func(ev *pb.Event){
		typeName(&pb.Event_DeviceRegistered_{}):   noop,
		typeName(&pb.Event_DeviceUnregistered_{}): noop,
		typeName(&pb.Event_ResourcePublished{}): func(ev *pb.Event) {
			CleanUpResourceLinksPublished(ev.GetResourcePublished())
		},
		typeName(&pb.Event_ResourceUnpublished{}): func(ev *pb.Event) {
			CleanUpResourceLinksUnpublished(ev.GetResourceUnpublished())
		},
		typeName(&pb.Event_ResourceChanged{}): func(ev *pb.Event) {
			CleanUpResourceChanged(ev.GetResourceChanged())
		},
		typeName(&pb.Event_OperationProcessed_{}):   noop,
		typeName(&pb.Event_SubscriptionCanceled_{}): noop,
		typeName(&pb.Event_ResourceUpdatePending{}): func(ev *pb.Event) {
			CleanUpResourceUpdatePending(ev.GetResourceUpdatePending())
		},
		typeName(&pb.Event_ResourceUpdated{}): func(ev *pb.Event) {
			CleanUpResourceUpdated(ev.GetResourceUpdated())
		},
		typeName(&pb.Event_ResourceRetrievePending{}): func(ev *pb.Event) {
			CleanUpResourceRetrievePending(ev.GetResourceRetrievePending())
		},
		typeName(&pb.Event_ResourceRetrieved{}): func(ev *pb.Event) {
			CleanUpResourceRetrieved(ev.GetResourceRetrieved())
		},
		typeName(&pb.Event_ResourceDeletePending{}): func(ev *pb.Event) {
			CleanUpResourceDeletePending(ev.GetResourceDeletePending())
		},
		typeName(&pb.Event_ResourceDeleted{}): func(ev *pb.Event) {
			CleanResourceDeleted(ev.GetResourceDeleted())
		},
		typeName(&pb.Event_ResourceCreatePending{}): func(ev *pb.Event) {
			CleanUpResourceCreatePending(ev.GetResourceCreatePending())
		},
		typeName(&pb.Event_ResourceCreated{}): func(ev *pb.Event) {
			CleanUpResourceCreated(ev.GetResourceCreated())
		},
		typeName(&pb.Event_DeviceMetadataUpdatePending{}): func(ev *pb.Event) {
			CleanUpDeviceMetadataUpdatePending(ev.GetDeviceMetadataUpdatePending())
		},
		typeName(&pb.Event_DeviceMetadataUpdated{}): func(ev *pb.Event) {
			CleanUpDeviceMetadataUpdated(ev.GetDeviceMetadataUpdated())
		},
	}

	handler, ok := cleanupHandlerFn[GetEventType(ev)]
	require.True(t, ok)

	handler(ev)
}

func CmpEvent(t *testing.T, expected, got *pb.Event) {
	require.Equal(t, GetEventType(expected), GetEventType(got))

	type cmpFn = func(t *testing.T, e, g *pb.Event)

	typeName := func(v interface{}) string {
		return reflect.TypeOf(v).String()
	}

	cmpFnMap := map[string]cmpFn{
		typeName(&pb.Event_ResourceChanged{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceChanged(t, e.GetResourceChanged(), g.GetResourceChanged())
		},
		typeName(&pb.Event_ResourceUpdatePending{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceUpdatePending(t, e.GetResourceUpdatePending(), g.GetResourceUpdatePending())
		},
		typeName(&pb.Event_ResourceUpdated{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceUpdated(t, e.GetResourceUpdated(), g.GetResourceUpdated())
		},
		typeName(&pb.Event_ResourceRetrieved{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceRetrieved(t, e.GetResourceRetrieved(), g.GetResourceRetrieved())
		},
		typeName(&pb.Event_ResourceDeleted{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceDeleted(t, e.GetResourceDeleted(), g.GetResourceDeleted())
		},
		typeName(&pb.Event_ResourceCreatePending{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceCreatePending(t, e.GetResourceCreatePending(), g.GetResourceCreatePending())
		},
		typeName(&pb.Event_ResourceCreated{}): func(t *testing.T, e, g *pb.Event) {
			CmpResourceCreated(t, e.GetResourceCreated(), g.GetResourceCreated())
		},
	}

	cmp, ok := cmpFnMap[GetEventType(expected)]
	if !ok {
		cmp = func(t *testing.T, e, g *pb.Event) {
			CleanUpEvent(t, e)
			CleanUpEvent(t, g)
			test.CheckProtobufs(t, e, g, test.RequireToCheckFunc(require.Equal))
		}
	}

	cmp(t, expected, got)
}

func CmpEvents(t *testing.T, expected, got []*pb.Event) {
	require.Len(t, got, len(expected))

	// normalize
	for i := range expected {
		expected[i].SubscriptionId = ""
		got[i].SubscriptionId = ""
		CleanUpEvent(t, expected[i])
		CleanUpEvent(t, got[i])
	}

	// compare
	for _, gotV := range got {
		test.CheckProtobufs(t, expected, gotV, test.RequireToCheckFunc(require.Contains))
	}
	for _, expectedV := range expected {
		test.CheckProtobufs(t, got, expectedV, test.RequireToCheckFunc(require.Contains))
	}
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
