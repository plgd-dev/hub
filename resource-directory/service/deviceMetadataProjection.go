package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type deviceMetadataProjection struct {
	lock sync.Mutex
	data *events.DeviceMetadataSnapshotTaken

	subscriptions *Subscriptions
}

func NewDeviceMetadataProjection(subscriptions *Subscriptions) eventstore.Model {
	return &deviceMetadataProjection{
		subscriptions: subscriptions,
	}
}

func (p *deviceMetadataProjection) Clone() *deviceMetadataProjection {
	p.lock.Lock()
	defer p.lock.Unlock()

	data := &events.DeviceMetadataSnapshotTaken{
		DeviceId:              p.data.GetDeviceId(),
		DeviceMetadataUpdated: p.data.GetDeviceMetadataUpdated(),
		UpdatePendings:        p.data.GetUpdatePendings(),
		EventMetadata:         p.data.GetEventMetadata(),
	}

	return &deviceMetadataProjection{
		data: data,
	}
}

func (p *deviceMetadataProjection) InitialNotifyOfDeviceMetadata(ctx context.Context, subscription *subscription) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return subscription.NotifyOfUpdatedDeviceMetadata(ctx, p.data.GetDeviceMetadataUpdated())
}

func (p *deviceMetadataProjection) InitialSendDevicesMetadataPending(ctx context.Context, subscription *subscription) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.onDeviceMetadataUpdatePendingLocked(ctx, subscription.NotifyOfUpdatePendingDeviceMetadata)
}

func (p *deviceMetadataProjection) onDeviceMetadataUpdatePendingLocked(ctx context.Context, do func(ctx context.Context, createPending *events.DeviceMetadataUpdatePending) error) error {
	log.Debugf("deviceMetadataProjection.onDeviceMetadataUpdatePendingLocked %v", p.data.GetUpdatePendings())
	var errors []error
	now := time.Now()
	for _, ev := range p.data.GetUpdatePendings() {
		if ev.IsExpired(now) {
			continue
		}
		err := do(ctx, ev)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device metadata update pending event: %v", errors)
	}
	return nil
}

func (p *deviceMetadataProjection) onDeviceMetadataUpdatedLocked(ctx context.Context) error {
	log.Debugf("deviceMetadataProjection.onDeviceMetadataUpdatedLocked %v", p.data)
	return p.subscriptions.OnDeviceMetadataUpdated(ctx, p.data.GetDeviceMetadataUpdated())
}

func (p *deviceMetadataProjection) EventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.EventType()
}

func (p *deviceMetadataProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onDeviceMetadataUpdatePending bool
	var onDeviceMetadataUpdated bool

	p.lock.Lock()
	defer p.lock.Unlock()
	var anyEventProcessed bool
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		log.Debugf("deviceMetadataProjection.Handle deviceID=%v eventype%v version=%v", eu.GroupID(), eu.EventType(), eu.Version())
		anyEventProcessed = true
		if p.data == nil {
			p.data = &events.DeviceMetadataSnapshotTaken{
				DeviceId:      eu.GroupID(),
				EventMetadata: events.MakeEventMeta("", 0, eu.Version()),
			}
		}
		p.data.GetEventMetadata().Version = eu.Version()
		switch eu.EventType() {
		case (&events.DeviceMetadataSnapshotTaken{}).EventType():
			var e events.DeviceMetadataSnapshotTaken
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			if len(e.GetUpdatePendings()) > 0 {
				if len(e.GetUpdatePendings()) != len(p.data.GetUpdatePendings()) {
					onDeviceMetadataUpdatePending = true
				} else {
					for i, upd := range e.GetUpdatePendings() {
						if p.data.GetUpdatePendings()[i].GetAuditContext().GetCorrelationId() != upd.GetAuditContext().GetCorrelationId() {
							onDeviceMetadataUpdatePending = true
							break
						}
					}
				}
			}
			onDeviceMetadataUpdated = true
			p.data = &e
		case (&events.DeviceMetadataUpdatePending{}).EventType():
			var e events.DeviceMetadataUpdatePending
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			if err := p.data.HandleDeviceMetadataUpdatePending(ctx, &e); err != nil {
				continue
			}
			onDeviceMetadataUpdatePending = true
			p.data.DeviceId = e.GetDeviceId()
		case (&events.DeviceMetadataUpdated{}).EventType():
			var e events.DeviceMetadataUpdated
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			p.data.DeviceId = e.GetDeviceId()
			ok, _ := p.data.HandleDeviceMetadataUpdated(ctx, &e, false)
			if ok {
				onDeviceMetadataUpdated = true
				onDeviceMetadataUpdatePending = true
			}
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if onDeviceMetadataUpdatePending {
		err := p.onDeviceMetadataUpdatePendingLocked(ctx, p.subscriptions.OnDeviceMetadataUpdatePending)
		if err != nil {
			log.Errorf("%v", err)
		}
	}
	if onDeviceMetadataUpdated {
		if err := p.onDeviceMetadataUpdatedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	return nil
}
