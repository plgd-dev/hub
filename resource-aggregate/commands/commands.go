package commands

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

func NewConnectionStatus(value ConnectionStatus_Status, validUntil int64) *ConnectionStatus {
	return &ConnectionStatus{
		Value:      value,
		ValidUntil: validUntil,
	}
}

func (c *ConnectionStatus) Clone() *ConnectionStatus {
	if c == nil {
		return c
	}
	return NewConnectionStatus(c.GetValue(), c.GetValidUntil())
}
