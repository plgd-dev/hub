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

func CmpDeviceMetadataUpdated(t *testing.T, expected, got []*events.DeviceMetadataUpdated) {
	require.Len(t, got, len(expected))

	cleanUp := func(evt *events.DeviceMetadataUpdated) {
		evt.EventMetadata = nil
		evt.AuditContext = nil
		if evt.GetStatus() != nil {
			evt.GetStatus().ValidUntil = 0
		}
	}

	for idx := range expected {
		cleanUp(expected[idx])
		cleanUp(got[idx])
	}
	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}
