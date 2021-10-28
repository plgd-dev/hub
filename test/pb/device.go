package pb

import (
	"testing"

	pbGrpc "github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
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

func CleanUpDeviceMetadataUpdated(e *events.DeviceMetadataUpdated) *events.DeviceMetadataUpdated {
	if e.GetAuditContext() != nil {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	if e.GetStatus() != nil {
		e.GetStatus().ValidUntil = 0
	}
	return e
}

func CmpDeviceMetadataUpdated(t *testing.T, expected, got *events.DeviceMetadataUpdated) {
	CleanUpDeviceMetadataUpdated(expected)
	CleanUpDeviceMetadataUpdated(got)
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}

func CmpDeviceMetadataUpdatedSlice(t *testing.T, expected, got []*events.DeviceMetadataUpdated) {
	require.Len(t, got, len(expected))
	for idx := range expected {
		CleanUpDeviceMetadataUpdated(expected[idx])
		CleanUpDeviceMetadataUpdated(got[idx])
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}
