package pb

import (
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

func MakeServiceMetadataUpdated(servicesHeartbeat *events.ServicesHeartbeat, correlationID string) *events.ServiceMetadataUpdated {
	return &events.ServiceMetadataUpdated{
		ServicesHeartbeat: servicesHeartbeat,
		AuditContext:      commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
	}
}

func CleanUpServiceMetadataUpdated(e *events.ServiceMetadataUpdated, resetCorrelationID bool) *events.ServiceMetadataUpdated {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	if e.GetServicesHeartbeat() != nil {
		for i := range e.GetServicesHeartbeat().GetValid() {
			if e.GetServicesHeartbeat().GetValid()[i] != nil {
				e.GetServicesHeartbeat().GetValid()[i].ValidUntil = 0
			}
		}
		for i := range e.GetServicesHeartbeat().GetExpired() {
			if e.GetServicesHeartbeat().GetExpired()[i] != nil {
				e.GetServicesHeartbeat().GetExpired()[i].ValidUntil = 0
			}
		}
	}
	return e
}

func CmpServiceMetadataUpdated(t *testing.T, expected, got *events.ServiceMetadataUpdated) {
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpServiceMetadataUpdated(expected, resetCorrelationID)
	CleanUpServiceMetadataUpdated(got, resetCorrelationID)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpServiceMetadataUpdatedSlice(t *testing.T, expected, got []*events.ServiceMetadataUpdated) {
	require.Len(t, got, len(expected))
	for idx := range expected {
		resetCorrelationID := expected[idx].GetAuditContext().GetCorrelationId() == ""
		CleanUpServiceMetadataUpdated(expected[idx], resetCorrelationID)
		CleanUpServiceMetadataUpdated(got[idx], resetCorrelationID)
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

var cleanupServicesEventFn = map[string]func(ev eventstore.Event){
	getTypeName(&events.ServiceMetadataUpdated{}): func(ev eventstore.Event) {
		if v, ok := ev.(*events.ServiceMetadataUpdated); ok {
			CleanUpServiceMetadataUpdated(v, true)
		}
	},
	getTypeName(&events.ServiceMetadataSnapshotTaken{}): func(ev eventstore.Event) {
		if v, ok := ev.(*events.ServiceMetadataSnapshotTaken); ok {
			CleanUpServiceMetadataUpdated(v.GetServiceMetadataUpdated(), true)
			v.EventMetadata = nil
		}
	},
}

func CleanUpServiceEvent(t *testing.T, ev eventstore.Event) {
	handler, ok := cleanupServicesEventFn[getTypeName(ev)]
	require.True(t, ok)
	handler(ev)
}

func CmpServicesEvents(t *testing.T, expected, got []eventstore.Event) {
	require.Len(t, got, len(expected))

	// normalize
	for i := range expected {
		CleanUpServiceEvent(t, expected[i])
		CleanUpServiceEvent(t, got[i])
	}

	// compare
	for _, gotV := range got {
		test.CheckProtobufs(t, expected, gotV, test.RequireToCheckFunc(require.Contains))
	}
	for _, expectedV := range expected {
		test.CheckProtobufs(t, got, expectedV, test.RequireToCheckFunc(require.Contains))
	}
}
