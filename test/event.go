package test

import (
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
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

func ResourceLinkToResourceChangedEvent(deviceID string, l schema.ResourceLink) *pb.Event {
	return &pb.Event{
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: &events.ResourceChanged{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     l.Href,
				},
				Status: commands.Status_OK,
			},
		},
	}
}

func ResourceLinksToExpectedResourceChangedEvents(deviceID string, links []schema.ResourceLink) map[string]*pb.Event {
	e := make(map[string]*pb.Event)
	for _, l := range links {
		e[deviceID+l.Href] = ResourceLinkToResourceChangedEvent(deviceID, l)
	}
	return e
}

func CmpEvents(t *testing.T, expected, got []*pb.Event) {
	require.Len(t, got, len(expected))
	cleanUpResourceLinksPublished := func(ev *pb.Event) {
		if ev.GetResourcePublished() != nil {
			CleanUpResourceLinksPublished(ev.GetResourcePublished())
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
		CheckProtobufs(t, expected, gotV, RequireToCheckFunc(require.Contains))
	}
	for _, expectedV := range expected {
		CheckProtobufs(t, got, expectedV, RequireToCheckFunc(require.Contains))
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

	expectedData := DecodeCbor(t, e.GetContent().GetData())
	gotData := DecodeCbor(t, g.GetContent().GetData())
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
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

	expectedData, ok := DecodeCbor(t, e.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	gotData, ok := DecodeCbor(t, g.GetContent().GetData()).(map[interface{}]interface{})
	require.True(t, ok)
	delete(expectedData, "ins")
	delete(gotData, "ins")
	require.Equal(t, expectedData, gotData)
	e.GetContent().Data = nil
	g.GetContent().Data = nil

	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
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

	CheckProtobufs(t, expected, got, RequireToCheckFunc(require.Equal))
}
