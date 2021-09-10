package service

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/subscription"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/strings"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func clientObserveHandler(req *mux.Message, client *Client, observe uint32) {
	t := time.Now()
	defer func() {
		log.Debugf("clientObserveHandler takes %v", time.Since(t))
	}()

	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	switch observe {
	case 0:
		startResourceObservation(req, client, authCtx, deviceID, href)
	case 1:
		stopResourceObservation(req, client, authCtx, deviceID, href)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: invalid Observe value", authCtx.GetDeviceID(), deviceID, href), coapCodes.BadRequest, req.Token)
		return
	}

}

func SendResourceContentToObserver(client *Client, resourceChanged *events.ResourceChanged, observe uint32, token message.Token) {
	msg := pool.AcquireMessage(client.coapConn.Context())
	msg.SetCode(coapCodes.Content)
	msg.SetObserve(observe)
	msg.SetToken(token)
	if resourceChanged.GetContent() != nil {
		mediaType, err := coapconv.MakeMediaType(-1, resourceChanged.GetContent().GetContentType())
		if err != nil {
			log.Errorf("cannot set content format for observer: %v", err)
			return
		}
		msg.SetContentFormat(mediaType)
		msg.SetBody(bytes.NewReader(resourceChanged.GetContent().GetData()))
	}
	err := client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send observe notification to %v: %w", client.remoteAddrString(), err)
	}
	decodeMsgToDebug(client, msg, "SEND-NOTIFICATION")
}

type resourceSubscription struct {
	client   *Client
	token    message.Token
	authCtx  *authorizationContext
	deviceID string
	href     string

	seqNum uint32
	sub    *subscription.Sub
}

func (s *resourceSubscription) cancelSubscription(code coapCodes.Code) {
	err := s.client.server.taskQueue.Submit(func() {
		if _, err := s.client.cancelResourceSubscription(s.token.String()); err != nil {
			log.Errorf("failed to cancel resource /%v%v subscription: %w", s.deviceID, s.href, err)
		}
		s.client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v, device response: %v", s.authCtx.GetDeviceID(), s.deviceID, s.href, code), code, s.token)
	})
	if err != nil {
		log.Errorf("failed to cancel resource /%v%v subscription: %w", s.deviceID, s.href, err)
	}
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
		if e.GetResourceChanged().GetStatus() != commands.Status_OK {
			s.cancelSubscription(coapconv.StatusToCoapCode(e.GetResourceChanged().GetStatus(), coapconv.Retrieve))
			return nil
		}
		SendResourceContentToObserver(s.client, e.GetResourceChanged(), atomic.AddUint32(&s.seqNum, 1), s.token)
		return nil
	}
	return nil
}

func (s *resourceSubscription) Init() error {
	return s.sub.Init(s.client.server.ownerCache)
}

func (s *resourceSubscription) Close() error {
	return s.sub.Close()
}

func newResourceSubscription(req *mux.Message, client *Client, authCtx *authorizationContext, deviceID string, href string) *resourceSubscription {
	r := &resourceSubscription{
		client:   client,
		token:    req.Token,
		authCtx:  authCtx,
		deviceID: deviceID,
		href:     href,
		seqNum:   2,
	}
	res := &commands.ResourceId{DeviceId: deviceID, Href: href}
	sub := subscription.New(req.Context, client.server.resourceSubscriber, client.server.rdClient, r.eventHandler, req.Token.String(), client.server.config.APIs.COAP.SubscriptionBufferSize, client.server.config.APIs.COAP.SubscriptionCacheExpiration, func(err error) {
		log.Errorf("error occurs during processing event for /%v%v by subscription: %w", deviceID, href, err)
	}, &pb.SubscribeToEvents_CreateSubscription{
		ResourceIdFilter:    []string{res.ToString()},
		EventFilter:         []pb.SubscribeToEvents_CreateSubscription_Event{pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED},
		IncludeCurrentState: true,
	})
	r.sub = sub
	return r
}

func startResourceObservation(req *mux.Message, client *Client, authCtx *authorizationContext, deviceID, href string) {
	ok, err := client.server.ownerCache.OwnsDevice(req.Context, deviceID)
	if err != nil {
		code := codes.InvalidArgument
		s, ok := status.FromError(err)
		if ok {
			code = s.Code()
		}
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapconv.GrpcCode2CoapCode(code, coapconv.Retrieve), req.Token)
		return
	}
	if !ok {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: unauthorized access", authCtx.GetDeviceID(), deviceID, href), coapCodes.Unauthorized, req.Token)
		return
	}
	token := req.Token.String()
	sub := newResourceSubscription(req, client, authCtx, deviceID, href)
	_, loaded := client.resourceSubscriptions.LoadOrStore(token, sub)
	if loaded {
		if err := sub.Close(); err != nil {
			log.Errorf("failed to close resource /%v%v subscription: %w", deviceID, href, err)
		}
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: resource subscription with token %v already exist", authCtx.GetDeviceID(), deviceID, href, token), coapCodes.BadRequest, req.Token)
		return
	}
	err = sub.Init()
	if err != nil {
		_, _ = client.resourceSubscriptions.PullOut(token)
		if err := sub.Close(); err != nil {
			log.Errorf("failed to close resource /%v%v subscription: %w", deviceID, href, err)
		}
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
	}

	// response will be send from projection
}

func stopResourceObservation(req *mux.Message, client *Client, authCtx *authorizationContext, deviceID, href string) {
	token := req.Token.String()
	cancelled, err := client.cancelResourceSubscription(token)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}
	if !cancelled {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation /%v%v: subscription not found", authCtx.GetDeviceID(), deviceID, href), coapCodes.BadRequest, req.Token)
		return
	}
	SendResourceContentToObserver(client, nil, 1, req.Token)
}

func clientResetObservationHandler(req *mux.Message, client *Client) {
	token := req.Token.String()
	cancelled, err := client.cancelResourceSubscription(token)
	if err != nil {
		log.Errorf("cannot reset resource observation: %v", err)
		return
	}
	if !cancelled {
		log.Errorf("cannot reset resource observation: not found")
		return
	}
}
