package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"

	oapiConStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const lastCheckKey = "lastcheck"
const lastDevicesRegisteredKey = "lastdevicesregistered"
const lastDevicesOnlineKey = "lastdevicesonline"
const lastDevicesOfflineKey = "lastdevicesoffline"

var devicesLastCheckKeySubscriptionQueryIndex = bson.D{
	{typeKey, 1},
	{lastCheckKey, 1},
}

type dbDevicesSub struct {
	ID                    string `bson:"_id"`
	URL                   string
	CorrelationID         string // uuid
	Type                  oapiConStore.Type
	ContentType           string
	EventTypes            []events.EventType
	DeviceID              string `bson:"deviceid"`
	Href                  string `bson:"href"`
	SequenceNumber        uint64 `bson:"sequencenumber"`
	UserID                string `bson:"userid"`
	SigningSecret         string
	AccessToken           string
	LastDevicesRegistered []string `bson:"lastdevicesregistered"`
	LastDevicesOnline     []string `bson:"lastdevicesonline"`
	LastDevicesOffline    []string `bson:"lastdevicesoffline"`
	LastCheck             int64    `bson:"lastcheck"`
}

func toStringArray(devices []events.Device) []string {
	d := make([]string, 0, len(devices))
	for _, v := range devices {
		d = append(d, v.ID)
	}
	return d
}

func toDeviceArray(devices []string) []events.Device {
	d := make([]events.Device, 0, len(devices))
	for _, v := range devices {
		d = append(d, events.Device{
			ID: v,
		})
	}
	return d
}

func makeDBDevicesSub(sub store.DevicesSubscription) dbDevicesSub {
	return dbDevicesSub{
		ID:                    sub.ID,
		URL:                   sub.URL,
		CorrelationID:         sub.CorrelationID,
		Type:                  sub.Type,
		ContentType:           sub.ContentType,
		EventTypes:            sub.EventTypes,
		DeviceID:              sub.DeviceID,
		Href:                  sub.Href,
		SequenceNumber:        sub.SequenceNumber,
		UserID:                sub.UserID,
		SigningSecret:         sub.SigningSecret,
		AccessToken:           sub.AccessToken,
		LastDevicesRegistered: toStringArray([]events.Device(sub.LastDevicesRegistered)),
		LastDevicesOnline:     toStringArray([]events.Device(sub.LastDevicesOnline)),
		LastDevicesOffline:    toStringArray([]events.Device(sub.LastDevicesOffline)),
		LastCheck:             sub.LastCheck.Unix(),
	}
}

func (s *Store) SaveDevicesSubscription(ctx context.Context, sub store.DevicesSubscription) error {
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
	if sub.UserID == "" {
		return fmt.Errorf("invalid UserID")
	}

	DBSub := makeDBDevicesSub(sub)
	col := s.client.Database(s.DBName()).Collection(subscriptionsCName)

	if _, err := col.InsertOne(ctx, DBSub); err != nil {
		return fmt.Errorf("cannot save devices subscription: %w", err)
	}
	return nil
}

func (s *Store) LoadDevicesSubscriptions(ctx context.Context, query store.DevicesSubscriptionQuery, h store.DevicesSubscriptionHandler) error {
	var iter *mongo.Cursor
	var err error
	col := s.client.Database(s.DBName()).Collection(subscriptionsCName)

	switch {
	case query.SubscriptionID != "":
		iter, err = col.Find(ctx, bson.M{"_id": query.SubscriptionID})
	case !query.LastCheck.IsZero():
		q := bson.M{
			typeKey: oapiConStore.Type_Devices,
			lastCheckKey: bson.M{
				"$lt": query.LastCheck.Unix(),
			},
		}
		iter, err = col.Find(ctx, q, &options.FindOptions{
			Hint: devicesLastCheckKeySubscriptionQueryIndex,
		})
		if err == mongo.ErrNilDocument {
			return nil
		}
		_, errUpd := col.UpdateMany(ctx, q, bson.M{
			"$set": bson.M{
				lastCheckKey: time.Now().Unix(),
			},
		})
		if errUpd != nil {
			iter.Close(ctx)
			return fmt.Errorf("cannot load all devices subscription - update last check: %w", errUpd)
		}
	default:
		iter, err = col.Find(ctx, bson.M{})
	}
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot load all devices subscription: %w", err)
	}

	i := devicesSubscriptionIterator{
		iter: iter,
	}
	err = h.Handle(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

func (s *Store) PopDevicesSubscription(ctx context.Context, subscriptionID string) (sub store.DevicesSubscription, err error) {
	var DBSub dbDevicesSub
	res := s.client.Database(s.DBName()).Collection(subscriptionsCName).FindOneAndDelete(ctx, bson.M{"_id": subscriptionID})
	if res.Err() != nil {
		return sub, res.Err()
	}
	err = res.Decode(&DBSub)
	if err != nil {
		return sub, err
	}
	return convertToDevicesSubscription(DBSub), nil
}

func (s *Store) UpdateDevicesSubscription(ctx context.Context, subscriptionID string, lastDevicesRegistered events.DevicesRegistered, lastDevicesOnline events.DevicesOnline, lastDevicesOffline events.DevicesOffline) error {
	col := s.client.Database(s.DBName()).Collection(subscriptionsCName)

	_, err := col.UpdateOne(ctx, bson.M{"_id": subscriptionID}, bson.M{
		"$set": bson.M{
			lastDevicesRegisteredKey: toStringArray([]events.Device(lastDevicesRegistered)),
			lastDevicesOnlineKey:     toStringArray([]events.Device(lastDevicesOnline)),
			lastDevicesOfflineKey:    toStringArray([]events.Device(lastDevicesOffline)),
		},
	})
	if err != nil {
		return fmt.Errorf("cannot update last devices hash of %v: %w", subscriptionID, err)
	}
	return err
}

type devicesSubscriptionIterator struct {
	iter *mongo.Cursor
}

func convertToDevicesSubscription(sub dbDevicesSub) (s store.DevicesSubscription) {
	s.ID = sub.ID
	s.URL = sub.URL
	s.CorrelationID = sub.CorrelationID
	s.Type = sub.Type
	s.ContentType = sub.ContentType
	s.EventTypes = sub.EventTypes
	s.DeviceID = sub.DeviceID
	s.Href = sub.Href
	s.SequenceNumber = sub.SequenceNumber
	s.UserID = sub.UserID
	s.SigningSecret = sub.SigningSecret
	s.AccessToken = sub.AccessToken
	s.LastDevicesRegistered = toDeviceArray(sub.LastDevicesRegistered)
	s.LastDevicesOnline = toDeviceArray(sub.LastDevicesOnline)
	s.LastDevicesOffline = toDeviceArray(sub.LastDevicesOffline)
	s.LastCheck = time.Unix(sub.LastCheck, 0)
	return
}

func (i *devicesSubscriptionIterator) Next(ctx context.Context, s *store.DevicesSubscription) bool {
	var sub dbDevicesSub

	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(&sub)
	if err != nil {
		return false
	}
	*s = convertToDevicesSubscription(sub)
	return true
}

func (i *devicesSubscriptionIterator) Err() error {
	return i.iter.Err()
}
