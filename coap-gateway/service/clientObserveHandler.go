package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	coapMessage "github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

const errFmtObserveResource = "cannot handle observe resource%v: %w"

func getObserveResourceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtObserveResource, "", err)
}

func clientObserveHandler(req *mux.Message, client *session, observe uint32) (*pool.Message, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getObserveResourceErr(err))
	}
	deviceID, href, err := message.URIToDeviceIDHref(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getObserveResourceErr(err))
	}

	switch observe {
	case 0:
		return startResourceObservation(req, client, authCtx, deviceID, href)
	case 1:
		return stopResourceObservation(req, client, deviceID, href)
	default:
		return nil, statusErrorf(coapCodes.BadRequest, errFmtObserveResource, fmt.Sprintf(" /%v%v", deviceID, href), fmt.Errorf("invalid Observe value(%v)", observe))
	}
}

func CreateResourceContentToObserver(client *session, resourceChanged *events.ResourceChanged, observe uint32, token coapMessage.Token) (*pool.Message, error) {
	msg := client.server.messagePool.AcquireMessage(client.coapConn.Context())
	msg.SetCode(coapCodes.Content)
	msg.SetObserve(observe)
	msg.SetToken(token)
	if resourceChanged.GetContent() != nil {
		mediaType, err := coapconv.MakeMediaType(-1, resourceChanged.GetContent().GetContentType())
		if err != nil {
			return nil, statusErrorf(coapCodes.BadRequest, "cannot set content format for observer: %v", err)
		}
		msg.SetContentFormat(mediaType)
		msg.SetBody(bytes.NewReader(resourceChanged.GetContent().GetData()))
	}
	return msg, nil
}

type resourceSubscription struct {
	client   *session
	token    coapMessage.Token
	authCtx  *authorizationContext
	deviceID string
	href     string

	seqNum uint32
	sub    *subscription.Sub

	mutex              sync.Mutex
	version            uint64
	versionInitialized bool
}

func (s *resourceSubscription) cancelSubscription(code coapCodes.Code) {
	err := s.client.server.taskQueue.Submit(func() {
		if _, err := s.client.cancelResourceSubscription(s.token.String()); err != nil {
			s.client.Errorf("failed to cancel resource /%v%v subscription: %w", s.deviceID, s.href, err)
		}
		err := statusErrorf(code, "cannot observe resource /%v%v, device response: %v", s.deviceID, s.href, code)
		resp := s.client.createErrorResponse(err, s.token)
		defer s.client.ReleaseMessage(resp)
		s.client.WriteMessage(resp)
		s.client.logRequestResponse(nil, resp, err)
	})
	if err != nil {
		s.client.Errorf("failed to cancel resource /%v%v subscription: %w", s.deviceID, s.href, err)
	}
}

func (s *resourceSubscription) isDuplicateEvent(ev *events.ResourceChanged) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.versionInitialized && s.version >= ev.GetEventMetadata().GetVersion() {
		return true
	}
	s.versionInitialized = true
	s.version = ev.GetEventMetadata().GetVersion()
	return false
}

func (s *resourceSubscription) eventHandler(e *pb.Event) error {
	switch {
	case e.GetResourceUnpublished() != nil:
		if !strings.Contains(e.GetResourceUnpublished().GetHrefs(), s.href) {
			return nil
		}
		s.cancelSubscription(coapCodes.ServiceUnavailable)
	case e.GetDeviceUnregistered() != nil:
		s.cancelSubscription(coapCodes.ServiceUnavailable)
	case e.GetResourceChanged() != nil:
		// deduplicate events
		if s.isDuplicateEvent(e.GetResourceChanged()) {
			return nil
		}
		if e.GetResourceChanged().GetStatus() != commands.Status_OK {
			s.cancelSubscription(coapconv.StatusToCoapCode(e.GetResourceChanged().GetStatus(), coapconv.Retrieve))
			return nil
		}
		seqNum := atomic.AddUint32(&s.seqNum, 1)
		err := s.client.server.taskQueue.Submit(func() {
			msg, err := CreateResourceContentToObserver(s.client, e.GetResourceChanged(), seqNum, s.token)
			if err != nil {
				s.client.Errorf("failed to create resource content for observer: %w", err)
			}
			defer s.client.ReleaseMessage(msg)
			s.client.WriteMessage(msg)
			s.client.logNotificationToClient(e.GetResourceChanged().GetResourceId().GetHref(), msg)
		})
		if err != nil {
			s.client.Errorf("failed to send event resource /%v%v to observer: %w", s.deviceID, s.href, err)
		}
		return nil
	}
	return nil
}

func (s *resourceSubscription) Init(ctx context.Context) error {
	res := &commands.ResourceId{DeviceId: s.deviceID, Href: s.href}
	client, err := s.client.server.rdClient.GetResources(ctx, &pb.GetResourcesRequest{
		ResourceIdFilter: []string{res.ToString()},
	})
	if err != nil {
		return err
	}

	var d *events.ResourceChanged
	for {
		resource, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		d = resource.Data
	}
	if err != nil {
		return err
	}
	authCtx, err := s.client.GetAuthorizationContext()
	if err != nil {
		return err
	}

	err = s.sub.Init(authCtx.GetUserID(), s.client.server.subscriptionsCache)
	if err != nil {
		return err
	}

	if d == nil {
		return nil
	}
	return s.eventHandler(&pb.Event{
		SubscriptionId: s.sub.Id(),
		CorrelationId:  s.sub.CorrelationId(),
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: d,
		},
	})
}

func (s *resourceSubscription) Close() error {
	return s.sub.Close()
}

func newResourceSubscription(req *mux.Message, client *session, authCtx *authorizationContext, deviceID string, href string) *resourceSubscription {
	r := &resourceSubscription{
		client:   client,
		token:    req.Token(),
		authCtx:  authCtx,
		deviceID: deviceID,
		href:     href,
		seqNum:   2,
	}

	res := &commands.ResourceId{DeviceId: deviceID, Href: href}
	sub := subscription.New(r.eventHandler, req.Token().String(), &pb.SubscribeToEvents_CreateSubscription{
		ResourceIdFilter: []string{res.ToString()},
		EventFilter:      []pb.SubscribeToEvents_CreateSubscription_Event{pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED},
	})
	r.sub = sub

	return r
}

const errFmtStartObserveResource = "cannot start resource observation /%v%v: %w"

func getStartObserveResourceErr(deviceID, href string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtStartObserveResource, deviceID, href, err)
}

func startResourceObservation(req *mux.Message, client *session, authCtx *authorizationContext, deviceID, href string) (*pool.Message, error) {
	ok, err := client.server.ownerCache.OwnsDevice(req.Context(), deviceID)
	if err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve), "%w", getStartObserveResourceErr(deviceID, href, err))
	}
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getStartObserveResourceErr(deviceID, href, fmt.Errorf("unauthorized access")))
	}
	token := req.Token().String()
	sub := newResourceSubscription(req, client, authCtx, deviceID, href)
	_, loaded := client.resourceSubscriptions.LoadOrStore(token, sub)
	if loaded {
		if err := sub.Close(); err != nil {
			client.Errorf("failed to close resource /%v%v subscription: %w", deviceID, href, err)
		}
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getStartObserveResourceErr(deviceID, href, fmt.Errorf("resource subscription with token %v already exist", token)))
	}
	err = sub.Init(req.Context())
	if err != nil {
		_, _ = client.resourceSubscriptions.PullOut(token)
		if errClose := sub.Close(); errClose != nil {
			client.Errorf("failed to close resource /%v%v subscription: %w", deviceID, href, errClose)
		}
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getStartObserveResourceErr(deviceID, href, err))
	}

	// response will be send from projection
	return nil, nil
}

const errFmtStopObserveResource = "cannot stop resource observation /%v%v: %w"

func getStopObserveResourceErr(deviceID, href string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtStopObserveResource, deviceID, href, err)
}

func stopResourceObservation(req *mux.Message, client *session, deviceID, href string) (*pool.Message, error) {
	token := req.Token().String()
	canceled, err := client.cancelResourceSubscription(token)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getStopObserveResourceErr(deviceID, href, err))
	}
	if !canceled {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getStopObserveResourceErr(deviceID, href, fmt.Errorf("subscription not found")))
	}
	return CreateResourceContentToObserver(client, nil, 1, req.Token())
}

func clientResetObservationHandler(req *mux.Message, client *session) (*pool.Message, error) {
	token := req.Token().String()
	canceled, err := client.cancelResourceSubscription(token)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("cannot reset resource observation: %w", err))
	}
	if !canceled {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("cannot reset resource observation: not found"))
	}
	// reset does not send response
	return nil, nil
}
