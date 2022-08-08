package commands

import (
	"sort"
)

func NewAuditContext(userID, correlationID string) *AuditContext {
	return &AuditContext{
		UserId:        userID,
		CorrelationId: correlationID,
	}
}

func (c *AuditContext) Clone() *AuditContext {
	if c == nil {
		return c
	}
	return NewAuditContext(c.GetUserId(), c.GetCorrelationId())
}

func NewConnectionStatus(value ConnectionStatus_Status, validUntil int64, connectionID string) *ConnectionStatus {
	return &ConnectionStatus{
		Value:        value,
		ValidUntil:   validUntil,
		ConnectionId: connectionID,
	}
}

func (c *ConnectionStatus) Clone() *ConnectionStatus {
	if c == nil {
		return c
	}
	return NewConnectionStatus(c.GetValue(), c.GetValidUntil(), c.GetConnectionId())
}

type Resources []*Resource

func (r Resources) Len() int      { return len(r) }
func (r Resources) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r Resources) Less(i, j int) bool {
	return r[i].GetHref() < r[j].GetHref()
}

func (r Resources) Sort() {
	sort.Sort(r)
}
