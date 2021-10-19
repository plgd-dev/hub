package test

import (
	"sort"
	"testing"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/stretchr/testify/require"
)

type SortPendingCommand []*pb.PendingCommand

func (a SortPendingCommand) Len() int      { return len(a) }
func (a SortPendingCommand) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortPendingCommand) Less(i, j int) bool {
	toKey := func(v *pb.PendingCommand) string {
		switch {
		case v.GetResourceCreatePending() != nil:
			return v.GetResourceCreatePending().GetResourceId().GetDeviceId() + v.GetResourceCreatePending().GetResourceId().GetHref()
		case v.GetResourceRetrievePending() != nil:
			return v.GetResourceRetrievePending().GetResourceId().GetDeviceId() + v.GetResourceRetrievePending().GetResourceId().GetHref()
		case v.GetResourceUpdatePending() != nil:
			return v.GetResourceUpdatePending().GetResourceId().GetDeviceId() + v.GetResourceUpdatePending().GetResourceId().GetHref()
		case v.GetResourceDeletePending() != nil:
			return v.GetResourceDeletePending().GetResourceId().GetDeviceId() + v.GetResourceDeletePending().GetResourceId().GetHref()
		case v.GetDeviceMetadataUpdatePending() != nil:
			return v.GetDeviceMetadataUpdatePending().GetDeviceId()
		}
		return ""
	}

	return toKey(a[i]) < toKey(a[j])
}

func CmpPendingCmds(t *testing.T, want []*pb.PendingCommand, got []*pb.PendingCommand) {
	require.Len(t, got, len(want))

	sort.Sort(SortPendingCommand(want))
	sort.Sort(SortPendingCommand(got))

	for idx := range want {
		switch {
		case got[idx].GetResourceCreatePending() != nil:
			got[idx].GetResourceCreatePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceCreatePending().EventMetadata = nil
		case got[idx].GetResourceRetrievePending() != nil:
			got[idx].GetResourceRetrievePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceRetrievePending().EventMetadata = nil
		case got[idx].GetResourceUpdatePending() != nil:
			got[idx].GetResourceUpdatePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceUpdatePending().EventMetadata = nil
		case got[idx].GetResourceDeletePending() != nil:
			got[idx].GetResourceDeletePending().AuditContext.CorrelationId = ""
			got[idx].GetResourceDeletePending().EventMetadata = nil
		case got[idx].GetDeviceMetadataUpdatePending() != nil:
			got[idx].GetDeviceMetadataUpdatePending().AuditContext.CorrelationId = ""
			got[idx].GetDeviceMetadataUpdatePending().EventMetadata = nil
		}
		CheckProtobufs(t, want[idx], got[idx], RequireToCheckFunc(require.Equal))
	}
}
