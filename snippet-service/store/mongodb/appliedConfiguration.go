package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) InsertAppliedConfigurations(ctx context.Context, confs ...*store.AppliedDeviceConfiguration) error {
	documents := make([]interface{}, 0, len(confs))
	for _, conf := range confs {
		documents = append(documents, conf)
	}
	_, err := s.Collection(appliedConfigurationsCol).InsertMany(ctx, documents)
	return err
}

func (s *Store) CreateAppliedDeviceConfiguration(ctx context.Context, adc *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	if err := store.ValidateAppliedConfiguration(adc, false); err != nil {
		return nil, err
	}
	newAdc := adc.Clone()
	if newAdc.GetId() == "" {
		newAdc.Id = uuid.NewString()
	}
	newAdc.Timestamp = time.Now().UnixNano()
	if _, err := s.Collection(appliedConfigurationsCol).InsertOne(ctx, newAdc); err != nil {
		return nil, err
	}
	return newAdc, nil
}

func (s *Store) UpdateAppliedDeviceConfiguration(ctx context.Context, adc *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	err := store.ValidateAppliedConfiguration(adc, true)
	if err != nil {
		return nil, err
	}
	newAdc := adc.Clone()
	filter := bson.M{
		store.IDKey:    newAdc.GetId(),
		store.OwnerKey: newAdc.GetOwner(),
	}
	newAdc.Timestamp = time.Now().UnixNano()
	opts := options.FindOneAndReplace().SetReturnDocument(options.After)
	result := s.Collection(appliedConfigurationsCol).FindOneAndReplace(ctx, filter, newAdc, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}
	updatedAdc := pb.AppliedDeviceConfiguration{}
	if err = result.Decode(&updatedAdc); err != nil {
		return nil, err
	}
	return &updatedAdc, nil
}

func toAppliedDeviceConfigurationsIdFilterQuery(query *pb.GetAppliedDeviceConfigurationsRequest) interface{} {
	idfilters := make([]interface{}, 0, 2)
	idFilter := inArrayQuery(store.IDKey, strings.Unique(query.GetIdFilter()))
	if idFilter != nil {
		idfilters = append(idfilters, idFilter)
	}
	deviceIdFilter := inArrayQuery(store.DeviceIDKey, strings.Unique(query.GetDeviceIdFilter()))
	if deviceIdFilter != nil {
		idfilters = append(idfilters, deviceIdFilter)
	}
	if len(idfilters) == 0 {
		return nil
	}
	if len(idfilters) == 1 {
		return idfilters[0]
	}
	return bson.M{mongodb.Or: idfilters}
}

func toAppliedDeviceConfigurationsQuery(owner string, query *pb.GetAppliedDeviceConfigurationsRequest) interface{} {
	filters := make([]interface{}, 0, 2)
	if owner != "" {
		filters = append(filters, bson.D{{Key: store.OwnerKey, Value: owner}})
	}
	idfilters := toAppliedDeviceConfigurationsIdFilterQuery(query)
	if idfilters != nil {
		filters = append(filters, idfilters)
	}
	return toFilter(mongodb.And, filters)
}

func (s *Store) GetAppliedDeviceConfigurations(ctx context.Context, owner string, query *pb.GetAppliedDeviceConfigurationsRequest, p store.ProccessAppliedDeviceConfigurations) error {
	cur, err := s.Collection(appliedConfigurationsCol).Find(ctx, toAppliedDeviceConfigurationsQuery(owner, query))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}
