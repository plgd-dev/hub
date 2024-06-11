package events

import (
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/stretchr/testify/assert"
)

func TestResourceStateSnapshotTakenForCommand_ValidateCancelPendingCommandsForNotExistingResource(t *testing.T) {
	e := NewResourceStateSnapshotTakenForCommand("", "", "", nil)

	// Test when there are no pending commands
	assert.False(t, e.validateCancelPendingCommandsForNotExistingResource(&commands.CancelPendingCommandsRequest{}))

	// Test when there are pending commands but no correlation ID filter
	e.ResourceUpdatePendings = []*ResourceUpdatePending{
		{
			AuditContext: &commands.AuditContext{
				CorrelationId: "1",
			},
		},
	}
	assert.True(t, e.validateCancelPendingCommandsForNotExistingResource(&commands.CancelPendingCommandsRequest{}))

	// Test when there are pending commands and a correlation ID filter
	e.ResourceCreatePendings = []*ResourceCreatePending{
		{
			AuditContext: &commands.AuditContext{
				CorrelationId: "2",
			},
		},
	}
	e.ResourceDeletePendings = []*ResourceDeletePending{
		{
			AuditContext: &commands.AuditContext{
				CorrelationId: "3",
			},
		},
	}
	assert.True(t, e.validateCancelPendingCommandsForNotExistingResource(&commands.CancelPendingCommandsRequest{
		CorrelationIdFilter: []string{"2", "4"},
	}))

	// Test when there are no pending commands matching the correlation ID filter
	assert.False(t, e.validateCancelPendingCommandsForNotExistingResource(&commands.CancelPendingCommandsRequest{
		CorrelationIdFilter: []string{"5", "6"},
	}))
}
