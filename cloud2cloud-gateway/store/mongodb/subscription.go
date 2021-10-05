package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/store"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const subscriptionsCName = "subscriptions"
const typeKey = "type"
const hrefKey = "href"
const sequenceNumberKey = "sequencenumber"
const deviceIDKey = "deviceid"
const initializedKey = "initialized"

var typeQueryIndex = bson.D{
	{Key: typeKey, Value: 1},
}

var typeDeviceIDQueryIndex = bson.D{
	{Key: typeKey, Value: 1},
	{Key: deviceIDKey, Value: 1},
}

var typeResourceIDQueryIndex = bson.D{
	{Key: typeKey, Value: 1},
	{Key: deviceIDKey, Value: 1},
	{Key: hrefKey, Value: 1},
}

var typeInitializedIDQueryIndex = bson.D{
	{Key: "_id", Value: 1},
	{Key: initializedKey, Value: 1},
}

type DBSub struct {
	ID             string `bson:"_id"`
	URL            string
	CorrelationID  string // uuid
	Type           store.Type
	Accept         []string
	EventTypes     events.EventTypes
	DeviceID       string `bson:"deviceid"`
	Href           string `bson:"href"`
	SequenceNumber uint64 `bson:"sequencenumber"`
	SigningSecret  string
	Initialized    bool `bson:"initialized"`
	AccessToken    string
}

func makeDBSub(sub store.Subscription) DBSub {
	return DBSub{
		ID:             sub.ID,
		URL:            sub.URL,
		CorrelationID:  sub.CorrelationID,
		Type:           sub.Type,
		Accept:         sub.Accept,
		EventTypes:     sub.EventTypes,
		DeviceID:       sub.DeviceID,
		Href:           sub.Href,
		SequenceNumber: sub.SequenceNumber,
		SigningSecret:  sub.SigningSecret,
		Initialized:    sub.Initialized,
		AccessToken:    sub.AccessToken,
	}
}

func validateSubscription(sub store.Subscription) error {
	if sub.ID == "" {
		return fmt.Errorf("invalid ID")
	}
	if len(sub.EventTypes) == 0 {
		return fmt.Errorf("invalid EventTypes")
	}
	if sub.URL == "" {
		return fmt.Errorf("invalid URL")
	}
	if sub.SigningSecret == "" {
		return fmt.Errorf("invalid SigningSecret")
	}
	if sub.AccessToken == "" {
		return fmt.Errorf("invalid AccessToken")
	}

	switch sub.Type {
	case store.Type_Devices:
		if sub.DeviceID != "" {
			return fmt.Errorf("invalid DeviceID for devices subscription type")
		}
		if sub.Href != "" {
			return fmt.Errorf("invalid Href for devices subscription type")
		}
	case store.Type_Device:
		if sub.DeviceID == "" {
			return fmt.Errorf("invalid DeviceID for device subscription type")
		}
		if sub.Href != "" {
			return fmt.Errorf("invalid Href for device subscription type")
		}
	case store.Type_Resource:
		if sub.DeviceID == "" {
			return fmt.Errorf("invalid DeviceID for resource subscription type")
		}
		if sub.Href == "" {
			return fmt.Errorf("invalid Href for resource subscription type")
		}
	default:
		return fmt.Errorf("not supported Type %v", sub.Type)
	}

	return nil
}

func (s *Store) SaveSubscription(ctx context.Context, sub store.Subscription) error {
	if err := validateSubscription(sub); err != nil {
		return fmt.Errorf("cannot save resource subscription: %w", err)
	}
	DBSub := makeDBSub(sub)
	col := s.Collection(subscriptionsCName)
	if _, err := col.InsertOne(ctx, DBSub); err != nil {
		return fmt.Errorf("cannot save resource subscription: %w", err)
	}
	return nil
}

func (s *Store) IncrementSubscriptionSequenceNumber(ctx context.Context, subscriptionID string) (uint64, error) {
	col := s.Collection(subscriptionsCName)
	res, err := incrementSubscriptionSequenceNumber(ctx, col, subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("cannot increment sequence number for subscription: %w", err)
	}
	return res, err
}

func (s *Store) SetInitialized(ctx context.Context, subscriptionID string) error {
	col := s.Collection(subscriptionsCName)
	if subscriptionID == "" {
		return fmt.Errorf("cannot set initialized: invalid subscriptionId")
	}

	opts := &options.UpdateOptions{}
	opts.SetHint(typeInitializedIDQueryIndex)
	_, err := col.UpdateOne(ctx, bson.D{{Key: "_id", Value: subscriptionID}, {Key: initializedKey, Value: false}}, bson.M{"$set": bson.M{initializedKey: true}}, opts)
	if err != nil {
		return fmt.Errorf("cannot set initialized for %v: %w", subscriptionID, err)
	}
	return nil
}

func (s *Store) PopSubscription(ctx context.Context, subscriptionID string) (sub store.Subscription, err error) {
	var DBSub DBSub
	col := s.Collection(subscriptionsCName)
	res := col.FindOneAndDelete(ctx, bson.M{"_id": subscriptionID})
	if res.Err() != nil {
		return sub, res.Err()
	}
	err = res.Decode(&DBSub)
	if err != nil {
		return sub, err
	}
	return convertToSubscription(DBSub), nil
}

func (s *Store) LoadSubscriptions(ctx context.Context, query store.SubscriptionQuery, h store.SubscriptionHandler) error {
	var iter *mongo.Cursor
	var err error

	col := s.Collection(subscriptionsCName)
	switch {
	case query.SubscriptionID != "":
		iter, err = col.Find(ctx, bson.M{"_id": query.SubscriptionID})
	case query.Type == "" && query.DeviceID == "" && query.Href == "":
		iter, err = col.Find(ctx, bson.M{})
	case query.Type == "":
		return fmt.Errorf("invalid Type")
	case query.DeviceID != "" && query.Href != "":
		q := bson.M{
			typeKey:     query.Type,
			deviceIDKey: query.DeviceID,
			hrefKey:     query.Href,
		}
		iter, err = col.Find(ctx, q, &options.FindOptions{
			Hint: typeResourceIDQueryIndex,
		})
	case query.DeviceID != "":
		q := bson.M{
			typeKey:     query.Type,
			deviceIDKey: query.DeviceID,
		}
		iter, err = col.Find(ctx, q, &options.FindOptions{
			Hint: typeDeviceIDQueryIndex,
		})
	default:
		q := bson.M{
			typeKey: query.Type,
		}
		iter, err = col.Find(ctx, q, &options.FindOptions{
			Hint: typeQueryIndex,
		})
	}
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}

	i := subscriptionIterator{
		iter: iter,
	}
	err = h.Handle(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type subscriptionIterator struct {
	iter *mongo.Cursor
}

func convertToSubscription(sub DBSub) (s store.Subscription) {
	s.ID = sub.ID
	s.URL = sub.URL
	s.CorrelationID = sub.CorrelationID
	s.Type = sub.Type
	s.Accept = sub.Accept
	s.EventTypes = sub.EventTypes
	s.DeviceID = sub.DeviceID
	s.Href = sub.Href
	s.SequenceNumber = sub.SequenceNumber
	s.SigningSecret = sub.SigningSecret
	s.AccessToken = sub.AccessToken
	return
}

func (i *subscriptionIterator) Next(ctx context.Context, s *store.Subscription) bool {
	var sub DBSub

	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(&sub)
	if err != nil {
		return false
	}
	*s = convertToSubscription(sub)
	return true
}

func (i *subscriptionIterator) Err() error {
	return i.iter.Err()
}
