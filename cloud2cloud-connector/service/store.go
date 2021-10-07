package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	"github.com/plgd-dev/kit/v2/log"
)

type Store struct {
	cache *Cache
	db    store.Store
}

type cloudLoader struct {
	cache *Cache
}

func (h *cloudLoader) Handle(ctx context.Context, iter store.LinkedCloudIter) (err error) {
	for {
		var cloud store.LinkedCloud
		if !iter.Next(ctx, &cloud) {
			break
		}
		h.cache.LoadOrCreateCloud(cloud)
	}
	return iter.Err()
}

type linkedAccountLoader struct {
	cache *Cache
}

func (h *linkedAccountLoader) Handle(ctx context.Context, iter store.LinkedAccountIter) (err error) {
	for {
		var account store.LinkedAccount
		if !iter.Next(ctx, &account) {
			break
		}
		_, _, err := h.cache.LoadOrCreateLinkedAccount(account)
		if err != nil {
			log.Errorf("cannot load linked account %+v: %w", account, err)
		}
	}
	return iter.Err()
}

func NewStore(ctx context.Context, db store.Store) (*Store, error) {
	cache := NewCache()
	hc := cloudLoader{
		cache: cache,
	}
	err := db.LoadLinkedClouds(ctx, store.Query{}, &hc)
	if err != nil {
		return nil, err
	}
	ha := linkedAccountLoader{
		cache: cache,
	}
	err = db.LoadLinkedAccounts(ctx, store.Query{}, &ha)
	if err != nil {
		return nil, err
	}
	return &Store{
		cache: cache,
		db:    db,
	}, nil
}

func (s *Store) LoadOrCreateCloud(ctx context.Context, cloud store.LinkedCloud) (store.LinkedCloud, bool, error) {
	err := s.db.InsertLinkedCloud(ctx, cloud)
	if err != nil {
		return store.LinkedCloud{}, false, err
	}
	cloud, loaded := s.cache.LoadOrCreateCloud(cloud)
	return cloud, loaded, nil
}

func (s *Store) LoadCloud(cloudID string) (store.LinkedCloud, bool) {
	return s.cache.LoadCloud(cloudID)
}

func (s *Store) LoadOrCreateLinkedAccount(ctx context.Context, linkedAccount store.LinkedAccount) (store.LinkedAccount, bool, error) {
	err := s.db.InsertLinkedAccount(ctx, linkedAccount)
	if err != nil {
		return store.LinkedAccount{}, false, err
	}
	return s.cache.LoadOrCreateLinkedAccount(linkedAccount)
}

func (s *Store) UpdateLinkedAccount(ctx context.Context, linkedAccount store.LinkedAccount) error {
	err := s.db.UpdateLinkedAccount(ctx, linkedAccount)
	if err != nil {
		return err
	}
	return s.cache.UpdateLinkedAccount(linkedAccount)
}

func (s *Store) LoadOrCreateSubscription(sub Subscription) (subscriptionData, bool, error) {
	return s.cache.LoadOrCreateSubscription(sub)
}

func (s *Store) LoadSubscription(subscripionID string) (subscriptionData, bool) {
	return s.cache.LoadSubscription(subscripionID)
}

func (s *Store) LoadDevicesSubscription(cloudID, linkedAccountID string) (subscriptionData, bool) {
	return s.cache.LoadDevicesSubscription(cloudID, linkedAccountID)
}

func (s *Store) LoadDeviceSubscription(cloudID, linkedAccountID, deviceID string) (subscriptionData, bool) {
	return s.cache.LoadDeviceSubscription(cloudID, linkedAccountID, deviceID)
}

func (s *Store) LoadResourceSubscription(cloudID, linkedAccountID, deviceID, href string) (subscriptionData, bool) {
	return s.cache.LoadResourceSubscription(cloudID, linkedAccountID, deviceID, href)
}

func (s *Store) Dump() interface{} {
	return s.cache.Dump()
}

func (s *Store) PullOutSubscription(subscripionID string) (subscriptionData, bool) {
	return s.cache.PullOutSubscription(subscripionID)
}

func (s *Store) PullOutCloud(ctx context.Context, cloudID string) (*CloudData, error) {
	cloud, ok := s.cache.PullOutCloud(cloudID)
	if !ok {
		return cloud, fmt.Errorf("not found")
	}
	return cloud, s.db.RemoveLinkedCloud(ctx, cloudID)
}

func (s *Store) PullOutLinkedAccount(ctx context.Context, cloudID, linkedAccountID string) (*LinkedAccountData, error) {
	cloud, ok := s.cache.PullOutLinkedAccount(cloudID, linkedAccountID)
	if !ok {
		return cloud, fmt.Errorf("not found")
	}
	return cloud, s.db.RemoveLinkedAccount(ctx, linkedAccountID)
}

func (s *Store) PullOutDevice(cloudID, linkedAccountID, deviceID string) (*DeviceData, bool) {
	return s.cache.PullOutDevice(cloudID, linkedAccountID, deviceID)
}

func (s *Store) PullOutResource(cloudID, linkedAccountID, deviceID, href string) (*ResourceData, bool) {
	return s.cache.PullOutResource(cloudID, linkedAccountID, deviceID, href)
}

func (s *Store) DumpLinkedAccounts() []provisionCacheData {
	return s.cache.DumpLinkedAccounts()
}

func (s *Store) DumpDevices() []subscriptionData {
	return s.cache.DumpDevices()
}

func (s *Store) DumpTasks() []Task {
	return s.cache.DumpTasks()
}
