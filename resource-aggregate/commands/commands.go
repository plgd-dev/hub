package commands

import (
	"sort"
)

func NewAuditContext(userID, correlationID, owner string) *AuditContext {
	return &AuditContext{
		UserId:        userID,
		CorrelationId: correlationID,
		Owner:         owner,
	}
}

func (c *AuditContext) Clone() *AuditContext {
	if c == nil {
		return c
	}
	return NewAuditContext(c.GetUserId(), c.GetCorrelationId(), c.GetOwner())
}

func NewConnection(status Connection_Status, connectionID string, connectedAt int64) *Connection {
	return &Connection{
		Status:      status,
		Id:          connectionID,
		ConnectedAt: connectedAt,
	}
}

func (c *Connection) Clone() *Connection {
	if c == nil {
		return c
	}
	return NewConnection(c.GetStatus(), c.GetId(), c.GetConnectedAt())
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
