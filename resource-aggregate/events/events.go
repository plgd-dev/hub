package events

func (e *EventMetadata) Clone() *EventMetadata {
	if e == nil {
		return nil
	}

	return &EventMetadata{
		Version:      e.GetVersion(),
		Timestamp:    e.GetTimestamp(),
		ConnectionId: e.GetConnectionId(),
		Sequence:     e.GetSequence(),
	}
}

func (d *DeviceMetadataUpdated) Clone() *DeviceMetadataUpdated {
	if d == nil {
		return nil
	}

	return &DeviceMetadataUpdated{
		DeviceId:              d.GetDeviceId(),
		Status:                d.GetStatus().Clone(),
		ShadowSynchronization: d.GetShadowSynchronization(),
		AuditContext:          d.GetAuditContext().Clone(),
		EventMetadata:         d.GetEventMetadata().Clone(),
		Canceled:              d.GetCanceled(),
	}
}

func (d *DeviceMetadataUpdatePending) Clone() *DeviceMetadataUpdatePending {
	if d == nil {
		return nil
	}
	return &DeviceMetadataUpdatePending{
		DeviceId:      d.GetDeviceId(),
		UpdatePending: d.GetUpdatePending(),
		AuditContext:  d.GetAuditContext().Clone(),
		EventMetadata: d.GetEventMetadata().Clone(),
		ValidUntil:    d.GetValidUntil(),
	}
}

func (d *DeviceMetadataSnapshotTaken) Clone() *DeviceMetadataSnapshotTaken {
	if d == nil {
		return nil
	}
	return &DeviceMetadataSnapshotTaken{
		DeviceId:              d.GetDeviceId(),
		DeviceMetadataUpdated: d.GetDeviceMetadataUpdated().Clone(),
		UpdatePendings:        append([]*DeviceMetadataUpdatePending(nil), d.GetUpdatePendings()...),
		EventMetadata:         d.GetEventMetadata().Clone(),
	}
}
