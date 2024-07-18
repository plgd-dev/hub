package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

var ErrNotFound = errors.New("not found")

type subscriptions struct {
	owner              string
	send               func(e *pb.Event) error
	subscriptionsCache *subscription.SubscriptionsCache
	leadRTEnabled      bool

	subs map[string]*subscription.Sub
}

func newSubscriptions(
	owner string,
	subscriptionsCache *subscription.SubscriptionsCache,
	leadRTEnabled bool,
	send func(e *pb.Event) error,
) *subscriptions {
	return &subscriptions{
		owner:              owner,
		subs:               make(map[string]*subscription.Sub),
		send:               send,
		subscriptionsCache: subscriptionsCache,
		leadRTEnabled:      leadRTEnabled,
	}
}

func (s *subscriptions) close() {
	for _, sub := range s.subs {
		err := sub.Close()
		if err != nil {
			log.Errorf("cannot close subscription('%v'): %w", sub.Id(), err)
		}
	}
}

func NewOperationProcessed(subscriptionId, correlationId string, code pb.Event_OperationProcessed_ErrorStatus_Code, msg string) *pb.Event {
	return &pb.Event{
		SubscriptionId: subscriptionId,
		CorrelationId:  correlationId,
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code:    code,
					Message: msg,
				},
			},
		},
	}
}

func (s *subscriptions) createSubscription(req *pb.SubscribeToEvents) error {
	sub := subscription.New(s.send, req.GetCorrelationId(), s.leadRTEnabled, req.GetCreateSubscription())
	err := s.send(NewOperationProcessed(sub.Id(), req.GetCorrelationId(), pb.Event_OperationProcessed_ErrorStatus_OK, ""))
	if err != nil {
		return err
	}
	err = sub.Init(s.owner, s.subscriptionsCache)
	if err != nil {
		_ = s.send(NewOperationProcessed(sub.Id(), req.GetCorrelationId(), pb.Event_OperationProcessed_ErrorStatus_ERROR, err.Error()))
		return err
	}
	s.subs[sub.Id()] = sub
	return nil
}

func (s *subscriptions) cancelSubscription(req *pb.SubscribeToEvents) error {
	sub, ok := s.subs[req.GetCancelSubscription().GetSubscriptionId()]
	if !ok {
		err := fmt.Errorf("cannot cancel subscription('%v'): %w", req.GetCancelSubscription().GetSubscriptionId(), ErrNotFound)
		err2 := s.send(NewOperationProcessed(req.GetCancelSubscription().GetSubscriptionId(), req.GetCorrelationId(), pb.Event_OperationProcessed_ErrorStatus_NOT_FOUND, err.Error()))
		var errors *multierror.Error
		errors = multierror.Append(errors, err)
		if err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot send operation processed event for subscription('%v'): %w", req.GetCancelSubscription().GetSubscriptionId(), err2))
		}
		return errors.ErrorOrNil()
	}
	delete(s.subs, req.GetCancelSubscription().GetSubscriptionId())
	err := sub.Close()
	err2 := s.send(&pb.Event{
		SubscriptionId: sub.Id(),
		CorrelationId:  req.GetCorrelationId(),
		Type: &pb.Event_SubscriptionCanceled_{
			SubscriptionCanceled: &pb.Event_SubscriptionCanceled{},
		},
	})
	if err2 != nil {
		return fmt.Errorf("cannot send subscription canceled for subscription('%v'): %w", sub.Id(), err2)
	}
	err2 = s.send(NewOperationProcessed(sub.Id(), req.GetCorrelationId(), pb.Event_OperationProcessed_ErrorStatus_OK, ""))
	if err2 != nil {
		return fmt.Errorf("cannot send operation processed event for subscription('%v'): %w", sub.Id(), err2)
	}
	return err
}

type subscribeToEventsHandler struct {
	srv      pb.GrpcGateway_SubscribeToEventsServer
	sendChan chan *pb.Event
}

func makeSubscribeToEventsHandler(srv pb.GrpcGateway_SubscribeToEventsServer, sendChan chan *pb.Event) subscribeToEventsHandler {
	return subscribeToEventsHandler{
		srv:      srv,
		sendChan: sendChan,
	}
}

func (h *subscribeToEventsHandler) send(e *pb.Event) error {
	// sending event go grpc goroutine
	select {
	case <-h.srv.Context().Done():
		return nil
	case h.sendChan <- e:
	default:
		return fmt.Errorf("event('%v') was dropped because client is exhausted", e)
	}
	return nil
}

func (h *subscribeToEventsHandler) processNextRequest(subs *subscriptions) (bool, error) {
	req, err := h.srv.Recv()
	if errors.Is(err, io.EOF) {
		return false, nil
	}
	if err != nil {
		return false, grpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err)
	}
	switch v := req.GetAction().(type) {
	case (*pb.SubscribeToEvents_CreateSubscription_):
		err := subs.createSubscription(req)
		if err != nil {
			log.Errorf("cannot create subscription: %w", err)
		}
	case (*pb.SubscribeToEvents_CancelSubscription_):
		err := subs.cancelSubscription(req)
		if err != nil {
			log.Errorf("cannot cancel subscription: %w", err)
		}
	case nil:
		err := fmt.Errorf("invalid action('%T')", v)
		_ = h.send(NewOperationProcessed("", req.GetCorrelationId(), pb.Event_OperationProcessed_ErrorStatus_ERROR, err.Error()))
		log.Errorf("%w", err)
	default:
		err := fmt.Errorf("unknown action %T", v)
		_ = h.send(NewOperationProcessed("", req.GetCorrelationId(), pb.Event_OperationProcessed_ErrorStatus_ERROR, err.Error()))
		log.Errorf("%w", err)
	}
	return true, nil
}

func (r *RequestHandler) SubscribeToEvents(srv pb.GrpcGateway_SubscribeToEventsServer) (errRet error) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	// sending event to grpc client
	sendChan := make(chan *pb.Event, r.config.APIs.GRPC.SubscriptionBufferSize)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-sendChan:
				if err := srv.Send(e); err != nil {
					log.Errorf("cannot send event('%v'): %w", e, err)
					return
				}
			}
		}
	}()

	h := makeSubscribeToEventsHandler(srv, sendChan)

	owner, err := grpc.OwnerFromTokenMD(ctx, r.ownerCache.OwnerClaim())
	if err != nil {
		return err
	}

	subs := newSubscriptions(owner, r.subscriptionsCache, r.config.Clients.Eventbus.NATS.LeadResourceType.IsEnabled(), h.send)
	defer subs.close()

	for {
		ok, err := h.processNextRequest(subs)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}
