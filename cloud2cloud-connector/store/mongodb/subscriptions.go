package mongodb

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const subscriptionCName = "Subscription"
const hrefKey = "href"
const linkedAccountIDKey = "linkedAccountID"
const deviceIDKey = "deviceid"
const signingSecretKey = "signingsecret"
const typeKey = "type"

var typeQueryIndex = bson.D{
	{typeKey, 1},
}

var subscriptionLinkAccountQueryIndex = bson.D{
	{linkedAccountIDKey, 1},
}

var subscriptionDeviceQueryIndex = bson.D{
	{deviceIDKey, 1},
	{typeKey, 1},
}

var subscriptionDeviceHrefQueryIndex = bson.D{
	{hrefKey, 1},
	{deviceIDKey, 1},
	{typeKey, 1},
}

type dbSubscription struct {
	SubscriptionID  string `bson:"_id"`
	LinkedAccountID string `bson:"linkedAccountID"`
	DeviceID        string `bson:"deviceid"`
	Href            string `bson:"href"`
	Type            string `bson:"type"`
	SigningSecret   string `bson:"signingsecret"`
}

func makeDBSubscription(sub store.Subscription) dbSubscription {
	return dbSubscription{
		SubscriptionID:  sub.SubscriptionID,
		LinkedAccountID: sub.LinkedAccountID,
		DeviceID:        sub.DeviceID,
		Href:            sub.Href,
		Type:            string(sub.Type),
		SigningSecret:   sub.SigningSecret,
	}
}

func (s *Store) LoadSubscriptions(ctx context.Context, queries []store.SubscriptionQuery, h store.SubscriptionHandler) error {
	col := s.client.Database(s.DBName()).Collection(subscriptionCName)
	opts := options.FindOptions{}
	q := bson.M{}
	bsonQueries := make([]bson.M, 0, 32)
	for _, query := range queries {
		tmp := bson.M{}
		if query.SubscriptionID != "" {
			tmp["_id"] = query.SubscriptionID
		}
		if query.LinkedAccountID != "" {
			tmp[linkedAccountIDKey] = query.LinkedAccountID
			opts.SetHint(subscriptionLinkAccountQueryIndex)
		}
		if query.Type != "" {
			tmp[typeKey] = query.Type
			opts.SetHint(typeQueryIndex)
		}
		if query.DeviceID != "" {
			if query.Type == "" {
				return fmt.Errorf("cannot load device subscription: invalid Type")
			}
			tmp[deviceIDKey] = query.DeviceID
			opts.SetHint(subscriptionDeviceQueryIndex)
		}
		if query.Href != "" {
			if query.DeviceID == "" {
				return fmt.Errorf("cannot load resource subscription: invalid DeviceID")
			}
			if query.Type == "" {
				return fmt.Errorf("cannot load resource subscription: invalid Type")
			}
			tmp[hrefKey] = query.Href
			opts.SetHint(subscriptionDeviceHrefQueryIndex)
		}
		bsonQueries = append(bsonQueries, tmp)
	}
	if len(bsonQueries) > 0 {
		q["$or"] = bsonQueries
	}

	iter, err := col.Find(ctx, q, &opts)
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

func (s *Store) FindOrCreateSubscription(ctx context.Context, sub store.Subscription) (store.Subscription, error) {
	if sub.SubscriptionID == "" {
		return store.Subscription{}, fmt.Errorf("invalid SubscriptionID")
	}
	if sub.LinkedAccountID == "" {
		return store.Subscription{}, fmt.Errorf("invalid LinkedAccountID")
	}
	q := bson.M{
		linkedAccountIDKey: sub.LinkedAccountID,
		typeKey:            sub.Type,
	}
	switch sub.Type {
	case "":
		return sub, fmt.Errorf("invalid Type")
	case store.Type_Device:
		if sub.DeviceID == "" {
			return store.Subscription{}, fmt.Errorf("invalid DeviceID")
		}
		q[deviceIDKey] = sub.DeviceID
	case store.Type_Resource:
		if sub.DeviceID == "" {
			return store.Subscription{}, fmt.Errorf("invalid DeviceID")
		}
		if sub.Href == "" {
			return store.Subscription{}, fmt.Errorf("invalid Href")
		}
		q[deviceIDKey] = sub.DeviceID
		q[hrefKey] = sub.Href
	}

	dbSub := makeDBSubscription(sub)
	col := s.client.Database(s.DBName()).Collection(subscriptionCName)

	opts := options.FindOneAndUpdateOptions{}
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.ReturnDocument(options.After))
	res := col.FindOneAndUpdate(ctx, bson.M{
		"$and": []bson.M{q},
	}, bson.M{"$setOnInsert": dbSub}, &opts)
	if res.Err() != nil {
		return store.Subscription{}, fmt.Errorf("cannot find and create for device subscription: %v", res.Err())
	}

	var storedSub dbSubscription
	err := res.Decode(&storedSub)
	if err != nil {
		return store.Subscription{}, fmt.Errorf("cannot devcode all device subscription: %v", err)
	}
	if storedSub.SubscriptionID != dbSub.SubscriptionID {
		return store.Subscription{}, fmt.Errorf("cannet create duplicit subscription of type %v:%v:%v", dbSub.Type, dbSub.DeviceID, dbSub.Href)
	}

	return sub, nil
}

func (s *Store) RemoveSubscriptions(ctx context.Context, query store.SubscriptionQuery) error {
	if query.Type != "" {
		return fmt.Errorf("remove by Type is not supported")
	}
	q := bson.M{}
	if query.SubscriptionID != "" {
		q["_id"] = query.SubscriptionID
	} else if query.LinkedAccountID != "" {
		q[linkedAccountIDKey] = query.LinkedAccountID
		if query.DeviceID != "" {
			q[deviceIDKey] = query.DeviceID
		}
		if query.DeviceID != "" && query.Href != "" {
			q[hrefKey] = query.Href
		}
	}
	if len(q) == 0 {
		return fmt.Errorf("remove all subscriptions is not supported")
	}
	_, err := s.client.Database(s.DBName()).Collection(subscriptionCName).DeleteMany(ctx, q)
	if err != nil {
		return fmt.Errorf("cannot remove subscriptions: %v", err)
	}
	return nil
}

type subscriptionIterator struct {
	iter *mongo.Cursor
}

func (i *subscriptionIterator) Next(ctx context.Context, s *store.Subscription) bool {
	var sub dbSubscription

	if i.iter == nil {
		return false
	}
	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(&sub)
	if err != nil {
		return false
	}

	s.LinkedAccountID = sub.LinkedAccountID
	s.DeviceID = sub.DeviceID
	s.SubscriptionID = sub.SubscriptionID
	s.Href = sub.Href
	s.Type = store.Type(sub.Type)
	s.SigningSecret = sub.SigningSecret

	return true
}

func (i *subscriptionIterator) Err() error {
	return i.iter.Err()
}
