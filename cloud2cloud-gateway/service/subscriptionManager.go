package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/store"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	kitSync "github.com/plgd-dev/kit/v2/sync"
)

type SubscriptionManager struct {
	ctx               context.Context
	subscriptions     *kitSync.Map
	store             store.Store
	gwClient          pb.GrpcGatewayClient
	reconnectInterval time.Duration
	emitEvent         emitEventFunc
}

func NewSubscriptionManager(ctx context.Context, store store.Store, gwClient pb.GrpcGatewayClient, reconnectInterval time.Duration, emitEvent emitEventFunc) *SubscriptionManager {
	return &SubscriptionManager{
		store:             store,
		reconnectInterval: reconnectInterval,
		subscriptions:     kitSync.NewMap(),
		gwClient:          gwClient,
		ctx:               ctx,
		emitEvent:         emitEvent,
	}
}

type subscriptionLoader struct {
	s *SubscriptionManager
}

func (l *subscriptionLoader) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	for {
		var s store.Subscription
		if !iter.Next(ctx, &s) {
			break
		}
		l.s.storeToSubs(s)
	}
	return iter.Err()
}

func (s *SubscriptionManager) LoadSubscriptions() error {
	h := subscriptionLoader{
		s: s,
	}
	err := s.store.LoadSubscriptions(s.ctx, store.SubscriptionQuery{}, &h)
	if err != nil {
		return err
	}
	return nil
}

func (s *SubscriptionManager) storeToSubs(sub store.Subscription) {
	subData := &SubscriptionData{
		data:                                sub,
		incrementSubscriptionSequenceNumber: s.store.IncrementSubscriptionSequenceNumber,
		gwClient:                            s.gwClient,
		setInitialized:                      s.store.SetInitialized,
	}
	s.subscriptions.LoadOrStore(sub.ID, subData)
}

func (s *SubscriptionManager) Connect(id string) error {
	subRaw, ok := s.subscriptions.Load(id)
	if !ok {
		return fmt.Errorf("not found")
	}
	sub := subRaw.(*SubscriptionData)
	if sub.sub != nil {
		if !ok {
			return fmt.Errorf("already connected")
		}
	}
	ctx := s.ctx
	if sub.data.AccessToken != "" {
		ctx = grpc.CtxWithToken(ctx, sub.data.AccessToken)
	}
	return sub.Connect(ctx, s.emitEvent, s.PullOut)
}

func (s *SubscriptionManager) Store(ctx context.Context, sub store.Subscription) error {
	err := s.store.SaveSubscription(ctx, sub)
	if err != nil {
		return err
	}
	s.storeToSubs(sub)
	return nil
}

func (s *SubscriptionManager) Load(id string) (store.Subscription, bool) {
	subDataRaw, ok := s.subscriptions.Load(id)
	if !ok {
		return store.Subscription{}, false
	}
	subData := subDataRaw.(*SubscriptionData)
	data := subData.Data()
	return data, true
}

func cancelSubscription(ctx context.Context, emitEvent emitEventFunc, sub store.Subscription) error {
	_, err := emitEvent(ctx, events.EventType_SubscriptionCanceled, sub, func(_ context.Context) (uint64, error) {
		return sub.SequenceNumber, nil
	}, nil)
	return err
}

func (s *SubscriptionManager) PullOut(ctx context.Context, id, href string) (store.Subscription, error) {
	subDataRaw, ok := s.subscriptions.PullOut(id)
	if !ok {
		return store.Subscription{}, fmt.Errorf("not found")
	}
	subData := subDataRaw.(*SubscriptionData)
	if href != "" && subData.data.Href != href {
		return store.Subscription{}, fmt.Errorf("invalid resource(%v) for subscription", href)
	}
	sub, err := s.store.PopSubscription(ctx, id)
	if err != nil {
		return store.Subscription{}, err
	}
	subscription := subData.Subscription()
	if subscription != nil {
		wait, err := subscription.Cancel()
		if err == nil {
			wait()
		}
	}
	return sub, nil
}

func (s *SubscriptionManager) DumpNotConnectedSubscriptionDatas() map[string]*SubscriptionData {
	out := make(map[string]*SubscriptionData)
	s.subscriptions.Range(func(key, resourceI interface{}) bool {
		subData := resourceI.(*SubscriptionData)
		if subData.Subscription() == nil {
			out[key.(string)] = resourceI.(*SubscriptionData)
		}
		return true
	})
	return out
}

func (s *SubscriptionManager) Run() {
	for {
		var wg sync.WaitGroup
		for _, task := range s.DumpNotConnectedSubscriptionDatas() {
			wg.Add(1)
			go func(subData *SubscriptionData) {
				defer wg.Done()
				err := subData.Connect(s.ctx, s.emitEvent, s.PullOut)
				if err != nil {
					log.Errorf("cannot connect %+v: %w", subData.Data(), err)
				}
			}(task)
		}
		wg.Wait()
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(s.reconnectInterval):
		}
	}
}
