package service

import (
	"context"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

// protected by lock in Projection struct in resource-aggregate/cqrs/eventstore/projection.go
type deviceMetadataProjection struct {
	deviceID string

	private struct {
		lock     sync.RWMutex // protects snapshot
		snapshot *events.DeviceMetadataSnapshotTaken
	}
}

func NewDeviceMetadataProjection(deviceID string) eventstore.Model {
	return &deviceMetadataProjection{deviceID: deviceID}
}

func (p *deviceMetadataProjection) GetDeviceID() string {
	return p.deviceID
}

func (p *deviceMetadataProjection) GetDeviceMetadataUpdated() *events.DeviceMetadataUpdated {
	p.private.lock.RLock()
	defer p.private.lock.RUnlock()
	return p.private.snapshot.GetDeviceMetadataUpdated()
}

func (p *deviceMetadataProjection) GetDeviceUpdatePendings(now time.Time) []*events.DeviceMetadataUpdatePending {
	updatePendings := make([]*events.DeviceMetadataUpdatePending, 0, 4)
	p.private.lock.RLock()
	defer p.private.lock.RUnlock()
	for _, pendingCmd := range p.private.snapshot.GetUpdatePendings() {
		if pendingCmd.IsExpired(now) {
			continue
		}
		updatePendings = append(updatePendings, pendingCmd)
	}
	return updatePendings
}

func (p *deviceMetadataProjection) IsInitialized() bool {
	p.private.lock.RLock()
	defer p.private.lock.RUnlock()
	return p.private.snapshot != nil
}

func (p *deviceMetadataProjection) EventType() string {
	s := &events.DeviceMetadataSnapshotTaken{}
	return s.EventType()
}

func (p *deviceMetadataProjection) handleEventLocked(ctx context.Context, eu eventstore.EventUnmarshaler) error {
	if p.private.snapshot == nil {
		p.private.snapshot = &events.DeviceMetadataSnapshotTaken{
			DeviceId:      eu.GroupID(),
			EventMetadata: events.MakeEventMeta("", 0, eu.Version()),
		}
	}
	eventMetadata := p.private.snapshot.GetEventMetadata().Clone()
	eventMetadata.Version = eu.Version()
	p.private.snapshot.EventMetadata = eventMetadata
	switch eu.EventType() {
	case (&events.DeviceMetadataSnapshotTaken{}).EventType():
		var e events.DeviceMetadataSnapshotTaken
		if err := eu.Unmarshal(&e); err != nil {
			return err
		}
		p.private.snapshot = &e
	case (&events.DeviceMetadataUpdatePending{}).EventType():
		var e events.DeviceMetadataUpdatePending
		if err := eu.Unmarshal(&e); err != nil {
			return err
		}
		if err := p.private.snapshot.HandleDeviceMetadataUpdatePending(ctx, &e); err != nil {
			return nil //nolint:nilerr
		}
		p.private.snapshot.DeviceId = e.GetDeviceId()
	case (&events.DeviceMetadataUpdated{}).EventType():
		var e events.DeviceMetadataUpdated
		if err := eu.Unmarshal(&e); err != nil {
			return err
		}
		p.private.snapshot.DeviceId = e.GetDeviceId()
		if _, err := p.private.snapshot.HandleDeviceMetadataUpdated(ctx, &e, false); err != nil {
			return nil //nolint:nilerr
		}
	}
	return nil
}

func (p *deviceMetadataProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	p.private.lock.Lock()
	defer p.private.lock.Unlock()
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("deviceMetadataProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		if err := p.handleEventLocked(ctx, eu); err != nil {
			return err
		}
	}
	return nil
}
