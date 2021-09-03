package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/patrickmn/go-cache"
	asClient "github.com/plgd-dev/cloud/authorization/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/subscription"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"

	"google.golang.org/grpc/codes"
)

type subscriptions struct {
	send                     func(e *pb.Event) error
	resourceSubscriber       *subscriber.Subscriber
	ownerCache               *asClient.OwnerCache
	resourceDirectoryClient  pb.GrpcGatewayClient
	subscriptionCleanUpCache *cache.Cache

	subs map[string]*subscription.Sub
}

func newSubscriptions(
	resourceDirectoryClient pb.GrpcGatewayClient,
	resourceSubscriber *subscriber.Subscriber,
	ownerCache *asClient.OwnerCache,
	subscriptionCleanUpCache *cache.Cache,
	send func(e *pb.Event) error) *subscriptions {
	return &subscriptions{
		subs:                     make(map[string]*subscription.Sub),
		send:                     send,
		resourceDirectoryClient:  resourceDirectoryClient,
		resourceSubscriber:       resourceSubscriber,
		ownerCache:               ownerCache,
		subscriptionCleanUpCache: subscriptionCleanUpCache,
	}
}

func (s *subscriptions) close() {
	for _, sub := range s.subs {
		s.subscriptionCleanUpCache.Delete(sub.Id())
		err := sub.Close()
		if err != nil {
			log.Errorf("cannot close subscription('%v'): %w", sub.Id(), err)
		}
	}
}

func (s *subscriptions) createSubscription(ctx context.Context, req *pb.SubscribeToEvents) error {
	sub := subscription.New(ctx, s.resourceSubscriber, s.resourceDirectoryClient, s.send, req.GetCorrelationId(), func(err error) {
		log.Errorf("error occurs during processing event by subscription: %v", err)
	}, req.GetCreateSubscription())
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
	err = sub.Init(s.ownerCache)
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
	s.subscriptionCleanUpCache.SetDefault(sub.Id(), sub)
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
	s.subscriptionCleanUpCache.Delete(sub.Id())
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

func (r *RequestHandler) SubscribeToEvents(srv pb.GrpcGateway_SubscribeToEventsServer) (errRet error) {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var sendMutex sync.Mutex
	send := func(e *pb.Event) error {
		sendMutex.Lock()
		defer sendMutex.Unlock()
		return srv.Send(e)
	}

	subs := newSubscriptions(r.resourceDirectoryClient, r.resourceSubscriber, r.ownerCache, r.subscriptionCleanUpCache, send)
	defer subs.close()

	for {
		req, err := srv.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err))
		}
		switch v := req.GetAction().(type) {
		case (*pb.SubscribeToEvents_CreateSubscription_):
			err := subs.createSubscription(ctx, req)
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
	}
	return nil
}
