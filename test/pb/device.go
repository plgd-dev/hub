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
	}

	for idx := range expected {
		cleanUp(expected[idx])
		cleanUp(got[idx])
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func MakeDeviceMetadataUpdated(deviceID string, shadowSynchronization commands.ShadowSynchronization, correlationId string) *events.DeviceMetadataUpdated {
	return &events.DeviceMetadataUpdated{
		DeviceId: deviceID,
		Status: &commands.ConnectionStatus{
			Value: commands.ConnectionStatus_ONLINE,
		},
		ShadowSynchronization: shadowSynchronization,
		AuditContext:          commands.NewAuditContext(oauthService.DeviceUserID, correlationId),
	}
}

func CleanUpDeviceMetadataUpdated(e *events.DeviceMetadataUpdated, resetCorrelationId bool) *events.DeviceMetadataUpdated {
	if e.GetAuditContext() != nil && resetCorrelationId {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	if e.GetStatus() != nil {
		e.GetStatus().ValidUntil = 0
	}
	return e
}

func CmpDeviceMetadataUpdated(t *testing.T, expected, got *events.DeviceMetadataUpdated) {
	resetCorrelationId := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpDeviceMetadataUpdated(expected, resetCorrelationId)
	CleanUpDeviceMetadataUpdated(got, resetCorrelationId)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpDeviceMetadataUpdatedSlice(t *testing.T, expected, got []*events.DeviceMetadataUpdated) {
	require.Len(t, got, len(expected))
	for idx := range expected {
		resetCorrelationId := expected[idx].GetAuditContext().GetCorrelationId() == ""
		CleanUpDeviceMetadataUpdated(expected[idx], resetCorrelationId)
		CleanUpDeviceMetadataUpdated(got[idx], resetCorrelationId)
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}
