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

func MakeServicesMetadataUpdated(status *events.ServicesStatus, correlationID string) *events.ServicesMetadataUpdated {
	return &events.ServicesMetadataUpdated{
		Status:       status,
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
	}
}

func CleanUpServicesMetadataUpdated(e *events.ServicesMetadataUpdated, resetCorrelationID bool) *events.ServicesMetadataUpdated {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	if e.GetStatus() != nil {
		for i := range e.GetStatus().GetOnline() {
			if e.GetStatus().GetOnline()[i] != nil {
				e.GetStatus().GetOnline()[i].OnlineValidUntil = 0
			}
		}
		for i := range e.GetStatus().GetOffline() {
			if e.GetStatus().GetOffline()[i] != nil {
				e.GetStatus().GetOffline()[i].OnlineValidUntil = 0
			}
		}
	}
	return e
}

func CmpServicesMetadataUpdated(t *testing.T, expected, got *events.ServicesMetadataUpdated) {
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpServicesMetadataUpdated(expected, resetCorrelationID)
	CleanUpServicesMetadataUpdated(got, resetCorrelationID)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpServicesMetadataUpdatedSlice(t *testing.T, expected, got []*events.ServicesMetadataUpdated) {
	require.Len(t, got, len(expected))
	for idx := range expected {
		resetCorrelationID := expected[idx].GetAuditContext().GetCorrelationId() == ""
		CleanUpServicesMetadataUpdated(expected[idx], resetCorrelationID)
		CleanUpServicesMetadataUpdated(got[idx], resetCorrelationID)
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

var cleanupServicesEventFn = map[string]func(ev eventstore.Event){
	getTypeName(&events.ServicesMetadataUpdated{}): func(ev eventstore.Event) {
		if v, ok := ev.(*events.ServicesMetadataUpdated); ok {
			CleanUpServicesMetadataUpdated(v, true)
		}
	},
	getTypeName(&events.ServicesMetadataSnapshotTaken{}): func(ev eventstore.Event) {
		if v, ok := ev.(*events.ServicesMetadataSnapshotTaken); ok {
			CleanUpServicesMetadataUpdated(v.GetServicesMetadataUpdated(), true)
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
