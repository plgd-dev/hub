package mongodb

import (
	"context"
	"fmt"
	"github.com/go-ocf/cloud/authorization/persistence"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const (
	deviceIDKey     = "_id"
	userIDKey       = "userid"
	accessTokenKey  = "accesstoken"
	refreshTokenKey = "refreshtoken"
	expiryKey       = "expiry"
)

// PersistenceTx prevents data race for a sequence of read and write operations.
type PersistenceTx struct {
	tx     mongo.Session
	dbname string
	err    error
	ctx    context.Context
}

// NewTransaction creates a new transaction.
// A transaction must always be closed:
//  tx := s.persistence.NewTransaction()
//  defer tx.Close()
func (p *Store) NewTransaction(ctx context.Context) persistence.PersistenceTx {
	tx, err := p.client.StartSession()
	if err == nil {
		err = tx.StartTransaction()
		if err != nil {
			tx.EndSession(ctx)
		}
	}
	return &PersistenceTx{tx: tx, dbname: p.DBName(), err: err, ctx: ctx}
}

// Retrieve device's authorization details.
func (p *PersistenceTx) Retrieve(deviceID, userID string) (_ *persistence.AuthorizedDevice, ok bool, err error) {
	if p.err != nil {
		err = p.err
		return
	}

	col := p.tx.Client().Database(p.dbname).Collection(userDevicesCName)
	iter, err := col.Find(p.ctx, bson.M{deviceIDKey: deviceID, userIDKey: userID}, &options.FindOptions{
		Hint: userDeviceQueryIndex,
	})

	if err == mongo.ErrNilDocument {
		err = nil
		return
	}
	if err != nil {
		return
	}

	it := iterator{
		iter: iter,
		ctx:  p.ctx,
	}
	defer it.Close()
	var d persistence.AuthorizedDevice
	ok = it.Next(&d)
	if it.Err() != nil {
		err = it.Err()
		return
	}

	return &d, ok, nil
}

// RetrieveByDevice device's authorization details.
func (p *PersistenceTx) RetrieveByDevice(deviceID string) (_ *persistence.AuthorizedDevice, ok bool, err error) {
	if p.err != nil {
		err = p.err
		return
	}

	col := p.tx.Client().Database(p.dbname).Collection(userDevicesCName)
	iter, err := col.Find(p.ctx, bson.M{deviceIDKey: deviceID})

	if err == mongo.ErrNilDocument {
		err = nil
		return
	}
	if err != nil {
		return
	}

	it := iterator{
		iter: iter,
		ctx:  p.ctx,
	}
	defer it.Close()
	var d persistence.AuthorizedDevice
	ok = it.Next(&d)
	if it.Err() != nil {
		err = it.Err()
		return
	}

	return &d, ok, nil
}

// RetrieveAll retrieves all user's authorized devices.
func (p *PersistenceTx) RetrieveAll(userID string) persistence.Iterator {
	if p.err != nil {
		return &iterator{err: p.err}
	}

	col := p.tx.Client().Database(p.dbname).Collection(userDevicesCName)
	iter, err := col.Find(p.ctx, bson.M{userIDKey: userID}, &options.FindOptions{
		Hint: userDevicesQueryIndex,
	})

	if err == mongo.ErrNilDocument {
		return &iterator{}
	}
	if err != nil {
		return &iterator{err: fmt.Errorf("cannot load all devices subscription: %v", err)}
	}

	return &iterator{
		iter: iter,
		ctx:  p.ctx,
	}
}

type iterator struct {
	err  error
	iter *mongo.Cursor
	ctx  context.Context
}

func (i *iterator) Next(s *persistence.AuthorizedDevice) bool {
	if i.err != nil {
		return false
	}

	var sub bson.M

	if !i.iter.Next(i.ctx) {
		return false
	}

	err := i.iter.Decode(&sub)
	if err != nil {
		return false
	}
	s.DeviceID = sub[deviceIDKey].(string)
	s.UserID = sub[userIDKey].(string)
	s.AccessToken = sub[accessTokenKey].(string)
	s.Expiry = time.Unix(sub[expiryKey].(int64), 0)
	s.RefreshToken = sub[refreshTokenKey].(string)

	return true
}

func (i *iterator) Err() error {
	return i.iter.Err()
}

func (i *iterator) Close() {
	i.err = i.iter.Close(i.ctx)
}

func makeRecord(d *persistence.AuthorizedDevice) bson.M {
	return bson.M{
		deviceIDKey:     d.DeviceID,
		userIDKey:       d.UserID,
		accessTokenKey:  d.AccessToken,
		refreshTokenKey: d.RefreshToken,
		expiryKey:       d.Expiry.Unix(),
	}
}

// Persist device's authorization details.
func (p *PersistenceTx) Persist(d *persistence.AuthorizedDevice) error {
	if p.err != nil {
		return p.err
	}

	record := makeRecord(d)
	col := p.tx.Client().Database(p.dbname).Collection(userDevicesCName)
	upsert := true
	if _, err := col.UpdateOne(p.ctx, bson.M{deviceIDKey: d.DeviceID}, bson.M{"$set": record}, &options.UpdateOptions{
		Upsert: &upsert,
	}); err != nil {
		return err
	}

	if err := p.tx.CommitTransaction(p.ctx); err != nil {
		return fmt.Errorf("cannot commit transaction: %w", err)
	}
	return nil
}

// Delete removes the device's authorization record.
func (p *PersistenceTx) Delete(deviceID, userID string) error {
	if p.err != nil {
		return p.err
	}
	col := p.tx.Client().Database(p.dbname).Collection(userDevicesCName)
	res, err := col.DeleteOne(p.ctx, bson.M{
		deviceIDKey: deviceID,
		userIDKey:   userID,
	})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("not found")
	}
	if err := p.tx.CommitTransaction(p.ctx); err != nil {
		return fmt.Errorf("cannot commit transaction: %w", err)
	}

	return nil
}

// Close must always be called (use defer immediately after NewTransaction).
func (p *PersistenceTx) Close() {
	if p.tx != nil {
		p.tx.EndSession(p.ctx)
	}
}
