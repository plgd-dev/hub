package service

import (
	"context"
	"fmt"
	"sync"

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
		Status:                p.data.GetStatus(),
		ShadowSynchronization: p.data.GetShadowSynchronization(),
		UpdatePendings:        p.data.GetUpdatePendings(),
		EventMetadata:         p.data.GetEventMetadata(),
	}

	return &deviceMetadataProjection{
		data: data,
	}
}

func appendOrdered(a []*events.DeviceMetadataUpdated, ev *events.DeviceMetadataUpdated) []*events.DeviceMetadataUpdated {
	if ev == nil {
		return a
	}
	if len(a) == 0 {
		return []*events.DeviceMetadataUpdated{ev}
	}
	if a[0].GetEventMetadata().GetVersion() < ev.GetEventMetadata().GetVersion() {
		return append(a, ev)
	}
	return append([]*events.DeviceMetadataUpdated{ev}, a...)
}

func (p *deviceMetadataProjection) InitialNotifyOfDeviceMetadata(ctx context.Context, subscription *devicesSubscription) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var onDeviceMetadataUpdated []*events.DeviceMetadataUpdated
	onDeviceMetadataUpdated = appendOrdered(onDeviceMetadataUpdated, p.data.GetStatus())
	onDeviceMetadataUpdated = appendOrdered(onDeviceMetadataUpdated, p.data.GetShadowSynchronization())

	var errors []error
	for _, r := range onDeviceMetadataUpdated {
		err := subscription.NotifyOfUpdatedDeviceMetadata(ctx, r)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func (p *deviceMetadataProjection) onDeviceMetadataUpdatePendingLocked(ctx context.Context) error {
	log.Debugf("deviceMetadataProjection.onDeviceMetadataUpdatePendingLocked %v", p.data.GetUpdatePendings())
	var errors []error
	for _, u := range p.data.GetUpdatePendings() {
		err := p.subscriptions.OnDeviceMetadataUpdatePending(ctx, u)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device metadata update pending event: %v", errors)
	}
	return nil
}

func (p *deviceMetadataProjection) onDeviceMetadataUpdatedLocked(ctx context.Context, updated []*events.DeviceMetadataUpdated) error {
	log.Debugf("deviceMetadataProjection.onDeviceMetadataUpdatedLocked %v", p.data)
	var errors []error
	for _, u := range updated {
		err := p.subscriptions.OnDeviceMetadataUpdated(ctx, u)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device metadata updated event: %v", errors)
	}
	return nil
}

func (p *deviceMetadataProjection) EventType() string {
	s := &events.ResourceLinksSnapshotTaken{}
	return s.EventType()
}

func (p *deviceMetadataProjection) Handle(ctx context.Context, iter eventstore.Iter) error {
	var onDeviceMetadataUpdatePending bool
	var onDeviceMetadataUpdated []*events.DeviceMetadataUpdated

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
			if p.data.GetShadowSynchronization().GetShadowSynchronization().GetDisabled() != e.GetShadowSynchronization().GetShadowSynchronization().GetDisabled() {
				onDeviceMetadataUpdated = appendOrdered(onDeviceMetadataUpdated, e.GetShadowSynchronization())
			}
			if p.data.GetStatus().GetStatus().GetValidUntil() != e.GetStatus().GetStatus().GetValidUntil() {
				onDeviceMetadataUpdated = appendOrdered(onDeviceMetadataUpdated, e.GetStatus())
			} else if p.data.GetStatus().GetStatus().GetValue() != e.GetStatus().GetStatus().GetValue() {
				onDeviceMetadataUpdated = appendOrdered(onDeviceMetadataUpdated, e.GetStatus())
			}
			p.data = &e
		case (&events.DeviceMetadataUpdatePending{}).EventType():
			var e events.DeviceMetadataUpdatePending
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}
			if err := p.data.HandleDeviceMetadataUpdatePending(ctx, &e); err != nil {
				return err
			}
			onDeviceMetadataUpdatePending = true
			p.data.DeviceId = e.GetDeviceId()
		case (&events.DeviceMetadataUpdated{}).EventType():
			var e events.DeviceMetadataUpdated
			if err := eu.Unmarshal(&e); err != nil {
				return err
			}

			p.data.DeviceId = e.GetDeviceId()
			err := p.data.HandleDeviceMetadataUpdated(ctx, &e)
			if err == nil {
				onDeviceMetadataUpdated = appendOrdered(onDeviceMetadataUpdated, &e)
				onDeviceMetadataUpdatePending = true
			}
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if onDeviceMetadataUpdatePending {
		err := p.onDeviceMetadataUpdatePendingLocked(ctx)
		if err != nil {
			log.Errorf("%v", err)
		}
	}
	if len(onDeviceMetadataUpdated) > 0 {
		if err := p.onDeviceMetadataUpdatedLocked(ctx, onDeviceMetadataUpdated); err != nil {
			log.Errorf("%v", err)
		}
	}

	return nil
}
