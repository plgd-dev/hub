package pb

import (
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

func CleanUpResourceStateSnapshotTaken(e *events.ResourceStateSnapshotTaken, resetCorrelationID bool) *events.ResourceStateSnapshotTaken {
	if e.GetAuditContext() != nil && resetCorrelationID {
		e.GetAuditContext().CorrelationId = ""
	}
	e.EventMetadata = nil
	return e
}

func CmpResourceStateSnapshotTaken(t *testing.T, expected, got *events.ResourceStateSnapshotTaken) {
	require.NotEmpty(t, expected)
	require.NotEmpty(t, got)

	CmpResourceChanged(t, expected.GetLatestResourceChange(), got.GetLatestResourceChange(), "")
	expected.LatestResourceChange = nil
	got.LatestResourceChange = nil

	require.Len(t, got.GetResourceCreatePendings(), len(expected.GetResourceCreatePendings()))
	for i := range expected.GetResourceCreatePendings() {
		CmpResourceCreatePending(t, expected.GetResourceCreatePendings()[i], got.GetResourceCreatePendings()[i])
	}
	got.ResourceCreatePendings = nil
	expected.ResourceCreatePendings = nil

	require.Len(t, got.GetResourceRetrievePendings(), len(expected.GetResourceRetrievePendings()))
	for i := range expected.GetResourceRetrievePendings() {
		CmpResourceRetrievePending(t, expected.GetResourceRetrievePendings()[i], got.GetResourceRetrievePendings()[i])
	}
	got.ResourceRetrievePendings = nil
	expected.ResourceRetrievePendings = nil

	require.Len(t, got.GetResourceUpdatePendings(), len(expected.GetResourceUpdatePendings()))
	for i := range expected.GetResourceUpdatePendings() {
		CmpResourceUpdatePending(t, expected.GetResourceUpdatePendings()[i], got.GetResourceUpdatePendings()[i])
	}
	got.ResourceUpdatePendings = nil
	expected.ResourceUpdatePendings = nil

	require.Len(t, got.GetResourceDeletePendings(), len(expected.GetResourceDeletePendings()))
	for i := range expected.GetResourceDeletePendings() {
		CmpResourceDeletePending(t, expected.GetResourceDeletePendings()[i], got.GetResourceDeletePendings()[i])
	}
	got.ResourceDeletePendings = nil
	expected.ResourceDeletePendings = nil

	resetCorrelationID := expected.GetAuditContext().GetCorrelationId() == ""
	CleanUpResourceStateSnapshotTaken(expected, resetCorrelationID)
	CleanUpResourceStateSnapshotTaken(got, resetCorrelationID)

	test.CheckProtobufs(t, expected, got, test.RequireToCheckFunc(require.Equal))
}
