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
		dev.Metadata.Status.ValidUntil = 0
		dev.Metadata.Status.ConnectionId = ""
	}

	for idx := range expected {
		cleanUp(expected[idx])
		cleanUp(got[idx])
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeDeviceMetadataUpdated(deviceID string, shadowSynchronization commands.ShadowSynchronization, correlationID string) *events.DeviceMetadataUpdated {
	return &events.DeviceMetadataUpdated{
		DeviceId: deviceID,
		Status: &commands.ConnectionStatus{
			Value: commands.ConnectionStatus_ONLINE,
		},
		ShadowSynchronization: shadowSynchronization,
		AuditContext:          commands.NewAuditContext(oauthService.DeviceUserID, correlationID),
	}
}

func CleanUpDeviceMetadataUpdated(e *events.DeviceMetadataUpdated, resetCorrelationID bool) *events.DeviceMetadataUpdated {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	if e.GetStatus() != nil {
		e.GetStatus().ValidUntil = 0
		e.GetStatus().ConnectionId = ""
	}
	return e
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
