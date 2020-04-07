package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/cqrs/eventstore"
	coap "github.com/go-ocf/go-coap"

	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/cloud/openapi-connector/events"
	"github.com/go-ocf/cloud/openapi-connector/store"

	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

type resourceCtx struct {
	lock                 sync.Mutex
	resource             *pbRA.Resource
	isPublished          bool
	content              *pbRA.ResourceChanged
	pendingContentUpdate []raEvents.ResourceUpdatePending
	store                store.Store
	raClient             pbRA.ResourceAggregateClient
}

func newResourceCtx(store store.Store, raClient pbRA.ResourceAggregateClient) func(context.Context) (eventstore.Model, error) {
	return func(context.Context) (eventstore.Model, error) {
		return &resourceCtx{
			store:                store,
			raClient:             raClient,
			pendingContentUpdate: make([]raEvents.ResourceUpdatePending, 0, 8),
		}, nil
	}
}

func (m *resourceCtx) cloneLocked() *resourceCtx {
	return &resourceCtx{
		resource:    m.resource,
		isPublished: m.isPublished,
		content:     m.content,
	}
}

func (m *resourceCtx) Clone() *resourceCtx {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.cloneLocked()
}

func (m *resourceCtx) onResourcePublishedLocked(ctx context.Context) error {
	return nil
}

func (m *resourceCtx) onResourceUnpublishedLocked(ctx context.Context) error {
	return nil
}

func (m *resourceCtx) onResourceChangedLocked(ctx context.Context) error {
	return nil
}

func makeUpdateHref(url, deviceID, href string) string {
	return url + kitHttp.CanonicalHref("devices/"+deviceID+"/"+href)
}

func updateDeviceResource(deviceID, href, contentType string, content []byte, l store.LinkedAccount) (string, []byte, pbRA.Status, error) {
	client := http.Client{}

	r, w := io.Pipe()

	req, err := http.NewRequest("POST", makeUpdateHref(l.TargetURL, deviceID, href), r)
	if err != nil {
		return "", nil, pbRA.Status_BAD_REQUEST, fmt.Errorf("cannot create post request: %v", err)
	}
	req.Header.Set(AcceptHeader, events.ContentType_JSON+","+events.ContentType_CBOR+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(events.ContentTypeKey, contentType)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(l.TargetCloud.AccessToken))

	go func() {
		defer w.Close()
		_, err := w.Write(content)
		if err != nil {
			log.Errorf("cannot update content of device %v resource %v: %v", deviceID, href, err)
		}
	}()
	httpResp, err := client.Do(req)
	if err != nil {
		return "", nil, pbRA.Status_UNAVAILABLE, fmt.Errorf("cannot post: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		status := pbRA.Status_UNKNOWN
		switch status {
		case http.StatusAccepted:
			status = pbRA.Status_ACCEPTED
		case http.StatusOK:
			status = pbRA.Status_OK
		case http.StatusBadRequest:
			status = pbRA.Status_BAD_REQUEST
		case http.StatusNotFound:
			status = pbRA.Status_NOT_FOUND
		case http.StatusNotImplemented:
			status = pbRA.Status_NOT_IMPLEMENTED
		case http.StatusForbidden:
			status = pbRA.Status_FORBIDDEN
		case http.StatusUnauthorized:
			status = pbRA.Status_UNAUTHORIZED
		}
		return "", nil, status, fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	respContentType := httpResp.Header.Get(events.ContentTypeKey)
	respContent := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = respContent.ReadFrom(httpResp.Body)
	if err != nil {
		return "", nil, pbRA.Status_UNAVAILABLE, fmt.Errorf("cannot read update response: %v", err)
	}

	return respContentType, respContent.Bytes(), pbRA.Status_OK, nil
}

func (m *resourceCtx) onPendingContentUpdate(ctx context.Context) error {
	var h SubscriptionHandler
	err := m.store.LoadSubscriptions(ctx, []store.SubscriptionQuery{store.SubscriptionQuery{Type: store.Type_Resource, DeviceID: m.resource.DeviceId, Href: m.resource.Href}}, &h)
	if err != nil {
		return err
	}
	if !h.ok {
		return fmt.Errorf("subscription not found")
	}

	var lah LinkedAccountHandler
	err = m.store.LoadLinkedAccounts(ctx, store.Query{ID: h.subscription.LinkedAccountID}, &lah)
	if err != nil {
		return err
	}
	if !h.ok {
		return fmt.Errorf("linked account not found")
	}

	linkedAccount, err := lah.linkedAccount.RefreshTokens(ctx, m.store)
	if err != nil {
		return err
	}

	userID, err := linkedAccount.OriginCloud.AccessToken.GetSubject()
	if err != nil {
		return fmt.Errorf("cannot get userID: %v", err)
	}

	for {
		if len(m.pendingContentUpdate) == 0 {
			break
		}
		contentType, content, status, err := updateDeviceResource(m.resource.DeviceId, m.resource.Href, m.pendingContentUpdate[0].Content.ContentType, m.pendingContentUpdate[0].Content.Data, linkedAccount)
		if err != nil {
			log.Errorf("cannot update content of device %v resource %v: %v", m.resource.DeviceId, m.resource.Href, err)
		}
		coapContentFormat := int32(-1)

		switch contentType {
		case coap.AppCBOR.String():
			coapContentFormat = int32(coap.AppCBOR)
		case coap.AppOcfCbor.String():
			coapContentFormat = int32(coap.AppOcfCbor)
		case coap.AppJSON.String():
			coapContentFormat = int32(coap.AppJSON)
		}

		_, err = m.raClient.ConfirmResourceUpdate(kitNetGrpc.CtxWithToken(ctx, linkedAccount.OriginCloud.AccessToken.String()), &pbRA.ConfirmResourceUpdateRequest{
			AuthorizationContext: &pbCQRS.AuthorizationContext{
				UserId: userID,
			},
			ResourceId:    m.resource.Id,
			CorrelationId: m.pendingContentUpdate[0].AuditContext.CorrelationId,
			CommandMetadata: &pbCQRS.CommandMetadata{
				ConnectionId: OpenapiConnectorConnectionId,
				//Sequence:     header.SequenceNumber,
			},
			Content: &pbRA.Content{
				Data:              content,
				ContentType:       contentType,
				CoapContentFormat: coapContentFormat,
			},
			Status: status,
		})
		if err != nil {
			log.Errorf("cannot update content of device %v resource %v: %v", m.resource.DeviceId, m.resource.Href, err)
		}
		m.pendingContentUpdate = m.pendingContentUpdate[1:]
	}
	return nil
}

func (m *resourceCtx) SnapshotEventType() string {
	s := &raEvents.ResourceStateSnapshotTaken{}
	return s.SnapshotEventType()
}

func (m *resourceCtx) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler
	var onResourcePublished, onResourceUnpublished, onResourceChanged bool
	m.lock.Lock()
	defer m.lock.Unlock()

	var anyEventProcessed bool
	for iter.Next(ctx, &eu) {
		anyEventProcessed = true
		log.Debugf("resourceCtx.Handle: DeviceId: %v, ResourceId: %v, Version: %v, EventType: %v", eu.GroupId, eu.AggregateId, eu.Version, eu.EventType)
		switch eu.EventType {
		case kitHttp.ProtobufContentType(&pbRA.ResourceStateSnapshotTaken{}):
			var s raEvents.ResourceStateSnapshotTaken
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if !m.isPublished {
				onResourcePublished = s.IsPublished
				onResourceUnpublished = !s.IsPublished
			}
			if m.content == nil {
				onResourceChanged = true
			} else {
				onResourceChanged = s.GetLatestResourceChange().GetEventMetadata().GetVersion() > m.content.GetEventMetadata().GetVersion()
			}
			m.content = s.LatestResourceChange
			m.resource = s.Resource
			m.isPublished = s.IsPublished
		case kitHttp.ProtobufContentType(&pbRA.ResourcePublished{}):
			var s raEvents.ResourcePublished
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if !m.isPublished {
				onResourcePublished = true
				onResourceUnpublished = false
			}
			m.isPublished = true
			m.resource = s.Resource
		case kitHttp.ProtobufContentType(&pbRA.ResourceUnpublished{}):
			if m.isPublished {
				onResourcePublished = false
				onResourceUnpublished = true
			}
			m.isPublished = false
		case kitHttp.ProtobufContentType(&pbRA.ResourceChanged{}):
			var s raEvents.ResourceChanged
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			if m.content == nil {
				onResourceChanged = true
			} else {
				onResourceChanged = s.GetEventMetadata().GetVersion() > m.content.GetEventMetadata().GetVersion()
			}
			m.content = &s.ResourceChanged
		case kitHttp.ProtobufContentType(&pbRA.ResourceUpdatePending{}):
			var s raEvents.ResourceUpdatePending
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			m.pendingContentUpdate = append(m.pendingContentUpdate, s)
		case kitHttp.ProtobufContentType(&pbRA.ResourceUpdated{}):
			var s raEvents.ResourceUpdated
			if err := eu.Unmarshal(&s); err != nil {
				return err
			}
			tmp := make([]raEvents.ResourceUpdatePending, 0, len(m.pendingContentUpdate))
			for _, cu := range m.pendingContentUpdate {
				if cu.AuditContext.CorrelationId != s.AuditContext.CorrelationId {
					tmp = append(tmp, cu)
				}
			}
			m.pendingContentUpdate = tmp
		}
	}

	if !anyEventProcessed {
		// if event event not processed, it means that the projection will be reloaded.
		return nil
	}

	if m.resource == nil {
		return fmt.Errorf("DeviceId: %v, ResourceId: %v: invalid resource is stored in eventstore: Resource attribute is not set", eu.GroupId, eu.AggregateId)
	}

	if onResourcePublished {
		if err := m.onResourcePublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	} else if onResourceUnpublished {
		if err := m.onResourceUnpublishedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	if onResourceChanged && m.isPublished {
		if err := m.onResourceChangedLocked(ctx); err != nil {
			log.Errorf("%v", err)
		}
	}

	if len(m.pendingContentUpdate) > 0 && m.isPublished {
		if err := m.onPendingContentUpdate(ctx); err != nil {
			log.Errorf("cannot update device %v resource %v: %v", m.resource.DeviceId, m.resource.Href, err)
		}
	}

	return nil
}
