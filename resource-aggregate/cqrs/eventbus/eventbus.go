package eventbus

import (
	"context"
)

// Publisher publish event to topics
type Publisher interface {
	Publish(ctx context.Context, topics []string, groupID, aggregateID string, event Event) error
	GetPublishSubject(owner string, event Event) []string
}

// Subscriber creates observation over topics. When subscriptionID is same among more Subscribers events are balanced among them.
type Subscriber interface {
	Subscribe(ctx context.Context, subscriptionID string, topics []string, eh Handler) (Observer, error)
}

// Observer handles events from observation and forward them to EventHandler.
type Observer interface {
	Close() error
	SetTopics(ctx context.Context, topics []string) error
}
