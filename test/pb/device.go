package pb

import (
	"testing"

	pbGrpc "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

func CmpDeviceValues(t *testing.T, expected, got []*pbGrpc.Device) {
	require.Len(t, got, len(expected))

	cleanUp := func(dev *pbGrpc.Device) {
		dev.ProtocolIndependentId = ""
		dev.Metadata.Connection.Id = ""
		dev.Metadata.Connection.ConnectedAt = 0
		dev.Metadata.Connection.LocalEndpoints = nil
		dev.Metadata.Connection.ServiceId = ""
		if dev.Metadata.TwinSynchronization != nil {
			dev.Metadata.TwinSynchronization.SyncingAt = 0
			dev.Metadata.TwinSynchronization.InSyncAt = 0
			dev.Metadata.TwinSynchronization.CommandMetadata = nil
		}
		dev.Data = nil
	}

	for idx := range expected {
		cleanUp(expected[idx])
		cleanUp(got[idx])
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeDeviceMetadataUpdated(deviceID string, connectionStatus commands.Connection_Status, connectionProtocol commands.Connection_Protocol, twinEnabled bool, twinSynchronizationState commands.TwinSynchronization_State, correlationID string) *events.DeviceMetadataUpdated {
	return &events.DeviceMetadataUpdated{
		DeviceId: deviceID,
		Connection: &commands.Connection{
			Status:   connectionStatus,
			Protocol: connectionProtocol,
		},
		TwinSynchronization: &commands.TwinSynchronization{
			State: twinSynchronizationState,
		},
		TwinEnabled:  twinEnabled,
		AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, correlationID, oauthService.DeviceUserID),
	}
}

func CleanUpDeviceMetadataUpdated(e *events.DeviceMetadataUpdated, resetCorrelationID bool) *events.DeviceMetadataUpdated {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	e.OpenTelemetryCarrier = nil
	if e.GetConnection() != nil {
		if e.GetConnection().IsOnline() {
			e.GetConnection().ServiceId = ""
		}
		e.GetConnection().Id = ""
		e.GetConnection().ConnectedAt = 0
		e.GetConnection().LocalEndpoints = nil
	}
	if e.GetTwinSynchronization() != nil {
		e.GetTwinSynchronization().CommandMetadata = nil
		e.GetTwinSynchronization().SyncingAt = 0
		e.GetTwinSynchronization().InSyncAt = 0
	}
	return e
}

func CleanUpDeviceMetadataSnapshotTaken(e *events.DeviceMetadataSnapshotTaken, resetCorrelationID bool) {
	CleanUpDeviceMetadataUpdated(e.DeviceMetadataUpdated, resetCorrelationID)
	e.EventMetadata = nil
}

func CmpDeviceMetadataUpdated(t *testing.T, expected, got *events.DeviceMetadataUpdated) {
	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpDeviceMetadataUpdated(expected, resetCorrelationID)
	CleanUpDeviceMetadataUpdated(got, resetCorrelationID)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpDeviceMetadataUpdatedSlice(t *testing.T, expected, got []*events.DeviceMetadataUpdated) {
	require.Len(t, got, len(expected))
	for idx := range expected {
		resetCorrelationID := expected[idx].GetAuditContext().GetCorrelationId() == ""
		CleanUpDeviceMetadataUpdated(expected[idx], resetCorrelationID)
		CleanUpDeviceMetadataUpdated(got[idx], resetCorrelationID)
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}
