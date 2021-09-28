package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/subscription"
	isClient "github.com/plgd-dev/cloud/identity/client"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"google.golang.org/grpc/codes"
)

type subscriptions struct {
	send                    func(e *pb.Event) error
	resourceSubscriber      *subscriber.Subscriber
	ownerCache              *isClient.OwnerCache
	resourceDirectoryClient pb.GrpcGatewayClient
	subscriptionBufferSize  int

	subs map[string]*subscription.Sub
}

func newSubscriptions(
	resourceDirectoryClient pb.GrpcGatewayClient,
	resourceSubscriber *subscriber.Subscriber,
	ownerCache *isClient.OwnerCache,
	subscriptionBufferSize int,
	send func(e *pb.Event) error) *subscriptions {
	return &subscriptions{
		subs:                    make(map[string]*subscription.Sub),
		send:                    send,
		resourceDirectoryClient: resourceDirectoryClient,
		resourceSubscriber:      resourceSubscriber,
		ownerCache:              ownerCache,
		subscriptionBufferSize:  subscriptionBufferSize,
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

func (s *subscriptions) createSubscription(ctx context.Context, req *pb.SubscribeToEvents) error {
	sub := subscription.New(ctx, s.resourceSubscriber, s.send, req.GetCorrelationId(), s.subscriptionBufferSize, func(err error) {
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

func (r *RequestHandler) SubscribeToEvents(srv pb.GrpcGateway_SubscribeToEventsServer) (errRet error) {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var sendMutex sync.Mutex
	send := func(e *pb.Event) error {
		sendMutex.Lock()
		defer sendMutex.Unlock()
		return srv.Send(e)
	}

	subs := newSubscriptions(r.resourceDirectoryClient, r.resourceSubscriber, r.ownerCache, r.config.APIs.GRPC.SubscriptionBufferSize, send)
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
