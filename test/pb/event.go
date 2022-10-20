package pb

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
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

var makeEventFromDataFn = map[string]func(e interface{}) *pb.Event{
	getTypeName(&pb.Event_DeviceRegistered{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_DeviceRegistered_{DeviceRegistered: e.(*pb.Event_DeviceRegistered)}}
	},
	getTypeName(&pb.Event_DeviceRegistered{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_DeviceUnregistered_{DeviceUnregistered: e.(*pb.Event_DeviceUnregistered)}}
	},
	getTypeName(&events.ResourceLinksPublished{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourcePublished{ResourcePublished: e.(*events.ResourceLinksPublished)}}
	},
	getTypeName(&events.ResourceLinksUnpublished{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceUnpublished{ResourceUnpublished: e.(*events.ResourceLinksUnpublished)}}
	},
	getTypeName(&events.ResourceChanged{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceChanged{ResourceChanged: e.(*events.ResourceChanged)}}
	},
	getTypeName(&pb.Event_OperationProcessed{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_OperationProcessed_{OperationProcessed: e.(*pb.Event_OperationProcessed)}}
	},
	getTypeName(&pb.Event_SubscriptionCanceled{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_SubscriptionCanceled_{SubscriptionCanceled: e.(*pb.Event_SubscriptionCanceled)}}
	},
	getTypeName(&events.ResourceUpdatePending{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceUpdatePending{ResourceUpdatePending: e.(*events.ResourceUpdatePending)}}
	},
	getTypeName(&events.ResourceUpdated{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceUpdated{ResourceUpdated: e.(*events.ResourceUpdated)}}
	},
	getTypeName(&events.ResourceRetrievePending{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceRetrievePending{ResourceRetrievePending: e.(*events.ResourceRetrievePending)}}
	},
	getTypeName(&events.ResourceRetrieved{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceRetrieved{ResourceRetrieved: e.(*events.ResourceRetrieved)}}
	},
	getTypeName(&events.ResourceDeletePending{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceDeletePending{ResourceDeletePending: e.(*events.ResourceDeletePending)}}
	},
	getTypeName(&events.ResourceDeleted{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceDeleted{ResourceDeleted: e.(*events.ResourceDeleted)}}
	},
	getTypeName(&events.ResourceCreatePending{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceCreatePending{ResourceCreatePending: e.(*events.ResourceCreatePending)}}
	},
	getTypeName(&events.ResourceCreated{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_ResourceCreated{ResourceCreated: e.(*events.ResourceCreated)}}
	},
	getTypeName(&events.DeviceMetadataUpdatePending{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_DeviceMetadataUpdatePending{DeviceMetadataUpdatePending: e.(*events.DeviceMetadataUpdatePending)}}
	},
	getTypeName(&events.DeviceMetadataUpdated{}): func(e interface{}) *pb.Event {
		return &pb.Event{Type: &pb.Event_DeviceMetadataUpdated{DeviceMetadataUpdated: e.(*events.DeviceMetadataUpdated)}}
	},
}

func getTypeName(v interface{}) string {
	return reflect.TypeOf(v).String()
}

// Try to create a *pb.Event which wraps the given data (data must be castable to one of pb.isEvent_Type types)
func ToEvent(data interface{}) *pb.Event {
	makeEvent, ok := makeEventFromDataFn[getTypeName(data)]
	if !ok {
		return nil
	}
	return makeEvent(data)
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
		return fmt.Sprintf("%T:%v:%v:%v:%v", ev.GetType(), v.DeviceMetadataUpdated.GetDeviceId(), v.DeviceMetadataUpdated.GetShadowSynchronization(), v.DeviceMetadataUpdated.GetStatus().GetValue(), v.DeviceMetadataUpdated.GetShadowSynchronizationStatus().GetValue())
	}
	return ""
}

func CleanUpDeviceRegistered(deviceRegistered *pb.Event_DeviceRegistered) {
	deviceRegistered.OpenTelemetryCarrier = nil
}

func CleanUpDeviceUnregistered(deviceUnregistered *pb.Event_DeviceUnregistered) {
	deviceUnregistered.OpenTelemetryCarrier = nil
}

var cleanupEventFn = map[string]func(ev *pb.Event){
	getTypeName(&pb.Event_DeviceRegistered_{}): func(ev *pb.Event) {
		CleanUpDeviceRegistered(ev.GetDeviceRegistered())
	},
	getTypeName(&pb.Event_DeviceUnregistered_{}): func(ev *pb.Event) {
		CleanUpDeviceUnregistered(ev.GetDeviceUnregistered())
	},
	getTypeName(&pb.Event_ResourcePublished{}): func(ev *pb.Event) {
		CleanUpResourceLinksPublished(ev.GetResourcePublished(), false)
	},
	getTypeName(&pb.Event_ResourceUnpublished{}): func(ev *pb.Event) {
		CleanUpResourceLinksUnpublished(ev.GetResourceUnpublished(), false)
	},
	getTypeName(&pb.Event_ResourceChanged{}): func(ev *pb.Event) {
		CleanUpResourceChanged(ev.GetResourceChanged(), false)
	},
	getTypeName(&pb.Event_OperationProcessed_{}): func(ev *pb.Event) {
		// nothing to do
	},
	getTypeName(&pb.Event_SubscriptionCanceled_{}): func(ev *pb.Event) {
		// nothing to do
	},
	getTypeName(&pb.Event_ResourceUpdatePending{}): func(ev *pb.Event) {
		CleanUpResourceUpdatePending(ev.GetResourceUpdatePending(), false)
	},
	getTypeName(&pb.Event_ResourceUpdated{}): func(ev *pb.Event) {
		CleanUpResourceUpdated(ev.GetResourceUpdated(), false)
	},
	getTypeName(&pb.Event_ResourceRetrievePending{}): func(ev *pb.Event) {
		CleanUpResourceRetrievePending(ev.GetResourceRetrievePending(), false)
	},
	getTypeName(&pb.Event_ResourceRetrieved{}): func(ev *pb.Event) {
		CleanUpResourceRetrieved(ev.GetResourceRetrieved(), false)
	},
	getTypeName(&pb.Event_ResourceDeletePending{}): func(ev *pb.Event) {
		CleanUpResourceDeletePending(ev.GetResourceDeletePending(), false)
	},
	getTypeName(&pb.Event_ResourceDeleted{}): func(ev *pb.Event) {
		CleanResourceDeleted(ev.GetResourceDeleted(), false)
	},
	getTypeName(&pb.Event_ResourceCreatePending{}): func(ev *pb.Event) {
		CleanUpResourceCreatePending(ev.GetResourceCreatePending(), false)
	},
	getTypeName(&pb.Event_ResourceCreated{}): func(ev *pb.Event) {
		CleanUpResourceCreated(ev.GetResourceCreated(), false)
	},
	getTypeName(&pb.Event_DeviceMetadataUpdatePending{}): func(ev *pb.Event) {
		CleanUpDeviceMetadataUpdatePending(ev.GetDeviceMetadataUpdatePending(), false)
	},
	getTypeName(&pb.Event_DeviceMetadataUpdated{}): func(ev *pb.Event) {
		CleanUpDeviceMetadataUpdated(ev.GetDeviceMetadataUpdated(), false)
	},
}

// Remove fields with unpredictable values
func CleanUpEvent(t *testing.T, ev *pb.Event) {
	handler, ok := cleanupEventFn[GetEventType(ev)]
	require.True(t, ok)
	handler(ev)
}

var compareEventFn = map[string]func(t *testing.T, e, g *pb.Event, cmpInterface string){
	getTypeName(&pb.Event_ResourceChanged{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) {
		CmpResourceChanged(t, e.GetResourceChanged(), g.GetResourceChanged(), cmpInterface)
	},
	getTypeName(&pb.Event_ResourceUpdatePending{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceUpdatePending(t, e.GetResourceUpdatePending(), g.GetResourceUpdatePending())
	},
	getTypeName(&pb.Event_ResourceUpdated{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceUpdated(t, e.GetResourceUpdated(), g.GetResourceUpdated())
	},
	getTypeName(&pb.Event_ResourceRetrievePending{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceRetrievePending(t, e.GetResourceRetrievePending(), g.GetResourceRetrievePending())
	},
	getTypeName(&pb.Event_ResourceRetrieved{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceRetrieved(t, e.GetResourceRetrieved(), g.GetResourceRetrieved())
	},
	getTypeName(&pb.Event_ResourceDeletePending{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceDeletePending(t, e.GetResourceDeletePending(), g.GetResourceDeletePending())
	},
	getTypeName(&pb.Event_ResourceDeleted{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceDeleted(t, e.GetResourceDeleted(), g.GetResourceDeleted())
	},
	getTypeName(&pb.Event_ResourceCreatePending{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceCreatePending(t, e.GetResourceCreatePending(), g.GetResourceCreatePending())
	},
	getTypeName(&pb.Event_ResourceCreated{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpResourceCreated(t, e.GetResourceCreated(), g.GetResourceCreated())
	},
	getTypeName(&pb.Event_DeviceMetadataUpdatePending{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpDeviceMetadataUpdatePending(t, e.GetDeviceMetadataUpdatePending(), g.GetDeviceMetadataUpdatePending())
	},
	getTypeName(&pb.Event_DeviceMetadataUpdated{}): func(t *testing.T, e, g *pb.Event, cmpInterface string) { //nolint:unparam
		CmpDeviceMetadataUpdated(t, e.GetDeviceMetadataUpdated(), g.GetDeviceMetadataUpdated())
	},
}

func CmpEvent(t *testing.T, expected, got *pb.Event, cmpInterface string) {
	require.Equal(t, GetEventType(expected), GetEventType(got))
	cmp, ok := compareEventFn[GetEventType(expected)]
	if !ok {
		cmp = func(t *testing.T, e, g *pb.Event, cmpInterface string) {
			CleanUpEvent(t, e)
			CleanUpEvent(t, g)
			test.CheckProtobufs(t, e, g, test.RequireToCheckFunc(require.Equal))
		}
	}

	cmp(t, expected, got, cmpInterface)
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

var getWrappedEventFromEventResponseFn = map[string]func(v *pb.GetEventsResponse) interface{}{
	getTypeName(&pb.GetEventsResponse_ResourceLinksPublished{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceLinksPublished()
	},
	getTypeName(&pb.GetEventsResponse_ResourceLinksUnpublished{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceLinksUnpublished()
	},
	getTypeName(&pb.GetEventsResponse_ResourceLinksSnapshotTaken{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceLinksSnapshotTaken()
	},
	getTypeName(&pb.GetEventsResponse_ResourceChanged{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceChanged()
	},
	getTypeName(&pb.GetEventsResponse_ResourceUpdatePending{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceUpdatePending()
	},
	getTypeName(&pb.GetEventsResponse_ResourceUpdated{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceUpdated()
	},
	getTypeName(&pb.GetEventsResponse_ResourceRetrievePending{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceRetrievePending()
	},
	getTypeName(&pb.GetEventsResponse_ResourceRetrieved{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceRetrieved()
	},
	getTypeName(&pb.GetEventsResponse_ResourceDeletePending{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceDeletePending()
	},
	getTypeName(&pb.GetEventsResponse_ResourceDeleted{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceDeleted()
	},
	getTypeName(&pb.GetEventsResponse_ResourceCreatePending{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceCreatePending()
	},
	getTypeName(&pb.GetEventsResponse_ResourceCreated{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceCreated()
	},
	getTypeName(&pb.GetEventsResponse_ResourceStateSnapshotTaken{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetResourceStateSnapshotTaken()
	},
	getTypeName(&pb.GetEventsResponse_DeviceMetadataUpdatePending{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetDeviceMetadataUpdatePending()
	},
	getTypeName(&pb.GetEventsResponse_DeviceMetadataUpdated{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetDeviceMetadataUpdated()
	},
	getTypeName(&pb.GetEventsResponse_DeviceMetadataSnapshotTaken{}): func(v *pb.GetEventsResponse) interface{} {
		return v.GetDeviceMetadataSnapshotTaken()
	},
}

func GetWrappedEvent(v *pb.GetEventsResponse) interface{} {
	getWrappedEventType := func(r *pb.GetEventsResponse) string {
		return fmt.Sprintf("%T", r.GetType())
	}
	getWrappedEventFn, ok := getWrappedEventFromEventResponseFn[getWrappedEventType(v)]
	if !ok {
		return nil
	}
	return getWrappedEventFn(v)
}

func CheckGetEventsResponse(t *testing.T, got []*pb.GetEventsResponse) {
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
