package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

type subscriptions struct {
	owner              string
	send               func(e *pb.Event) error
	subscriptionsCache *subscription.SubscriptionsCache

	subs map[string]*subscription.Sub
}

func newSubscriptions(
	owner string,
	subscriptionsCache *subscription.SubscriptionsCache,
	send func(e *pb.Event) error) *subscriptions {
	return &subscriptions{
		owner:              owner,
		subs:               make(map[string]*subscription.Sub),
		send:               send,
		subscriptionsCache: subscriptionsCache,
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

func (s *subscriptions) createSubscription(req *pb.SubscribeToEvents) error {
	sub := subscription.New(s.send, req.GetCorrelationId(), req.GetCreateSubscription())
	err := s.send(&pb.Event{
		SubscriptionId: sub.Id(),
		CorrelationId:  req.GetCorrelationId(),
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	err = sub.Init(s.owner, s.subscriptionsCache)
	if err != nil {
		_ = s.send(&pb.Event{
			SubscriptionId: sub.Id(),
			CorrelationId:  req.GetCorrelationId(),
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code:    pb.Event_OperationProcessed_ErrorStatus_ERROR,
						Message: err.Error(),
					},
				},
			},
		})
		return err
	}
	s.subs[sub.Id()] = sub
	return nil
}

func (s *subscriptions) cancelSubscription(ctx context.Context, req *pb.SubscribeToEvents) error {
	sub, ok := s.subs[req.GetCancelSubscription().GetSubscriptionId()]
	if !ok {
		err := fmt.Errorf("cannot cancel subscription('%v'): not found", req.GetCancelSubscription().GetSubscriptionId())
		err2 := s.send(&pb.Event{
			SubscriptionId: req.GetCancelSubscription().GetSubscriptionId(),
			CorrelationId:  req.GetCorrelationId(),
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code:    pb.Event_OperationProcessed_ErrorStatus_NOT_FOUND,
						Message: err.Error(),
					},
				},
			},
		})
		if err2 != nil {
			return fmt.Errorf("cannot send operation processed event for subscription('%v'): %v: %w", req.GetCancelSubscription().GetSubscriptionId(), err, err2)
		}
		return err
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
	err2 = s.send(&pb.Event{
		SubscriptionId: sub.Id(),
		CorrelationId:  req.GetCorrelationId(),
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	})
	if err2 != nil {
		return fmt.Errorf("cannot send operation processed event for subscription('%v'): %w", sub.Id(), err2)
	}
	return err
}

func processNextRequest(ctx context.Context, srv pb.GrpcGateway_SubscribeToEventsServer, subs *subscriptions, send func(e *pb.Event) error) (bool, error) {
	req, err := srv.Recv()
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err))
	}
	switch v := req.GetAction().(type) {
	case (*pb.SubscribeToEvents_CreateSubscription_):
		err := subs.createSubscription(req)
		if err != nil {
			log.Errorf("cannot create subscription: %w", err)
		}
	case (*pb.SubscribeToEvents_CancelSubscription_):
		err := subs.cancelSubscription(ctx, req)
		if err != nil {
			log.Errorf("cannot cancel subscription: %w", err)
		}
	case nil:
		err := fmt.Errorf("invalid action('%T')", v)
		_ = send(&pb.Event{
			CorrelationId: req.GetCorrelationId(),
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code:    pb.Event_OperationProcessed_ErrorStatus_ERROR,
						Message: err.Error(),
					},
				},
			},
		})
		log.Errorf("%w", err)
	default:
		err := fmt.Errorf("unknown action %T", v)
		_ = send(&pb.Event{
			CorrelationId: req.GetCorrelationId(),
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code:    pb.Event_OperationProcessed_ErrorStatus_ERROR,
						Message: err.Error(),
					},
				},
			},
		})
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

	// sending event go grpc goroutine
	send := func(e *pb.Event) error {
		select {
		case <-srv.Context().Done():
			return nil
		case sendChan <- e:
		default:
			return fmt.Errorf("event('%v') was dropped because client is exhausted", e)
		}
		return nil
	}

	owner, err := grpc.OwnerFromTokenMD(ctx, r.ownerCache.OwnerClaim())
	if err != nil {
		return err
	}

	subs := newSubscriptions(owner, r.subscriptionsCache, send)
	defer subs.close()

	for {
		ok, err := processNextRequest(ctx, srv, subs, send)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}
