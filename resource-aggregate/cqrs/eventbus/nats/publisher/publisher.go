package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	nats "github.com/nats-io/nats.go"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/protobuf/proto"
)

// MarshalerFunc marshal struct to bytes.
type MarshalerFunc = func(v interface{}) ([]byte, error)

type leadResourceType struct {
	filter      client.LeadResourceTypeFilter
	regexFilter []*regexp.Regexp
	useUUID     bool
}

// Publisher implements a eventbus.Publisher interface.
type Publisher struct {
	dataMarshaler    MarshalerFunc
	conn             *nats.Conn
	closeFunc        fn.FuncList
	publish          func(subj string, data []byte) error
	flusherTimeout   time.Duration
	leadResourceType *leadResourceType
}

func (p *Publisher) AddCloseFunc(f func()) {
	p.closeFunc.AddFunc(f)
}

type options struct {
	dataMarshaler    MarshalerFunc
	flusherTimeout   time.Duration
	leadResourceType *leadResourceType
}

type Option interface {
	apply(o *options)
}

type MarshalerOpt struct {
	dataMarshaler MarshalerFunc
}

func (o MarshalerOpt) apply(opts *options) {
	opts.dataMarshaler = o.dataMarshaler
}

func WithMarshaler(dataMarshaler MarshalerFunc) MarshalerOpt {
	return MarshalerOpt{
		dataMarshaler: dataMarshaler,
	}
}

type FlusherTimeoutOpt struct {
	flusherTimeout time.Duration
}

func (o FlusherTimeoutOpt) apply(opts *options) {
	if o.flusherTimeout > 0 {
		opts.flusherTimeout = o.flusherTimeout
	}
}

func WithFlusherTimeout(flusherTimeout time.Duration) FlusherTimeoutOpt {
	return FlusherTimeoutOpt{
		flusherTimeout: flusherTimeout,
	}
}

type LeadResourceTypeOpt struct {
	filter      client.LeadResourceTypeFilter
	regexFilter []*regexp.Regexp
	useUUID     bool
}

func (o LeadResourceTypeOpt) apply(opts *options) {
	opts.leadResourceType = &leadResourceType{
		filter:      o.filter,
		regexFilter: o.regexFilter,
		useUUID:     o.useUUID,
	}
}

func WithLeadResourceType(regexFilter []*regexp.Regexp, filter client.LeadResourceTypeFilter, useUUID bool) LeadResourceTypeOpt {
	return LeadResourceTypeOpt{
		regexFilter: regexFilter,
		filter:      filter,
		useUUID:     useUUID,
	}
}

// Create publisher with existing NATS connection and proto marshaller
func New(conn *nats.Conn, jetstream bool, opts ...Option) (*Publisher, error) {
	cfg := options{
		dataMarshaler:  json.Marshal,
		flusherTimeout: time.Second * 10,
	}
	for _, o := range opts {
		o.apply(&cfg)
	}

	publish := conn.Publish
	if jetstream {
		js, err := conn.JetStream()
		if err != nil {
			return nil, fmt.Errorf("cannot get jetstream context: %w", err)
		}
		publish = func(subj string, data []byte) error {
			_, err := js.Publish(subj, data)
			return err
		}
	}

	return &Publisher{
		dataMarshaler:    cfg.dataMarshaler,
		conn:             conn,
		publish:          publish,
		flusherTimeout:   cfg.flusherTimeout,
		leadResourceType: cfg.leadResourceType,
	}, nil
}

func matchType(t string, filter []*regexp.Regexp) bool {
	for _, f := range filter {
		if f.MatchString(t) {
			return true
		}
	}
	return false
}

func (p *Publisher) getLeadResourceTypeByFilter(event eventbus.Event) string {
	types := event.Types()
	if p.leadResourceType.regexFilter != nil {
		for _, t := range types {
			if matchType(t, p.leadResourceType.regexFilter) {
				return t
			}
		}
	}

	switch p.leadResourceType.filter {
	case client.LeadResourceTypeFilter_First:
		return types[0]
	case client.LeadResourceTypeFilter_Last:
		return types[len(types)-1]
	}
	return ""
}

func replaceSpecialCharacters(s string) string {
	// "*", ">" and "$" are reserved characters in NATS
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, "*", "_"), ">", "_"), "$", "_")
}

func ResourceTypeToUUID(resourceType string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(resourceType)).String()
}

func (p *Publisher) GetLeadResourceType(event eventbus.Event) string {
	if p.leadResourceType == nil || len(event.Types()) == 0 {
		return ""
	}

	leadResourceType := replaceSpecialCharacters(p.getLeadResourceTypeByFilter(event))
	if p.leadResourceType.useUUID && leadResourceType != "" {
		return ResourceTypeToUUID(leadResourceType)
	}
	return leadResourceType
}

func (p *Publisher) getPublishResourceEventSubject(owner string, resourceID *commands.ResourceId, event eventbus.Event) string {
	template := utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent
	opts := []func(values map[string]string){
		isEvents.WithOwner(owner), utils.WithDeviceID(event.GroupID()),
		utils.WithHrefId(utils.HrefToID(resourceID.GetHref()).String()), isEvents.WithEventType(event.EventType()),
	}
	if p.leadResourceType == nil {
		return isEvents.ToSubject(template, opts...)
	}
	// if leadResourceType is set, then the feature is enabled
	lrt := p.GetLeadResourceType(event)
	if lrt != "" {
		template = utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEventLeadResourceType
		opts = append(opts, utils.WithLeadResourceType(lrt))
	} else {
		// no lead resource type, but we append the suffix so we can subscribe to all events using a single
		// utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent + ".>" subject
		template = utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent + "." + utils.LeadResourcePrefix
	}
	return isEvents.ToSubject(template, opts...)
}

func (p *Publisher) GetPublishSubject(owner string, event eventbus.Event) []string {
	switch event.EventType() {
	case (&events.ResourceLinksPublished{}).EventType(), (&events.ResourceLinksUnpublished{}).EventType(), (&events.ResourceLinksSnapshotTaken{}).EventType():
		return []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourceLinksEvent, isEvents.WithOwner(owner), utils.WithDeviceID(event.GroupID()), isEvents.WithEventType(event.EventType()))}
	case (&events.DeviceMetadataUpdatePending{}).EventType(), (&events.DeviceMetadataUpdated{}).EventType(), (&events.DeviceMetadataSnapshotTaken{}).EventType():
		return []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithOwner(owner), utils.WithDeviceID(event.GroupID()), isEvents.WithEventType(event.EventType()))}
	}
	if ev, ok := event.(interface{ GetResourceId() *commands.ResourceId }); ok {
		return []string{p.getPublishResourceEventSubject(owner, ev.GetResourceId(), event)}
	}
	return nil
}

// Publish publishes an event to topics.
func (p *Publisher) Publish(ctx context.Context, topics []string, groupID, aggregateID string, event eventbus.Event) error {
	data, err := p.dataMarshaler(event)
	if err != nil {
		return errors.New("could not marshal data for event: " + err.Error())
	}

	e := pb.Event{
		EventType:   event.EventType(),
		Data:        data,
		Version:     event.Version(),
		GroupId:     groupID,
		AggregateId: aggregateID,
	}

	eData, err := proto.Marshal(&e)
	if err != nil {
		return errors.New("could not marshal event: " + err.Error())
	}

	var errors *multierror.Error
	for _, t := range topics {
		err = p.PublishData(t, eData)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	err = p.Flush(ctx)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	return errors.ErrorOrNil()
}

func (p *Publisher) PublishData(subj string, data []byte) error {
	return p.publish(subj, data)
}

func (p *Publisher) Flush(ctx context.Context) error {
	flushCtx := ctx
	_, ok := ctx.Deadline()
	if !ok {
		fctx, cancel := context.WithTimeout(ctx, p.flusherTimeout)
		defer cancel()
		flushCtx = fctx
	}
	return p.conn.FlushWithContext(flushCtx)
}

func (p *Publisher) Close() {
	p.closeFunc.Execute()
}
