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

func (s *Store) CreateAppliedConfiguration(ctx context.Context, adc *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
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

func (s *Store) UpdateAppliedConfiguration(ctx context.Context, adc *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
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

func toAppliedDeviceConfigurationsIdFilterQuery(idFilter, deviceIdFilter []string) interface{} {
	filters := make([]interface{}, 0, 2)
	idf := inArrayQuery(store.IDKey, strings.Unique(idFilter))
	if idf != nil {
		filters = append(filters, idf)
	}
	dif := inArrayQuery(store.DeviceIDKey, strings.Unique(deviceIdFilter))
	if dif != nil {
		filters = append(filters, dif)
	}
	return toFilter(mongodb.Or, filters)
}

func toAppliedDeviceConfigurationsQuery(owner string, idFilter, deviceIdFilter []string) interface{} {
	filters := make([]interface{}, 0, 2)
	if owner != "" {
		filters = append(filters, bson.D{{Key: store.OwnerKey, Value: owner}})
	}
	idfilters := toAppliedDeviceConfigurationsIdFilterQuery(idFilter, deviceIdFilter)
	if idfilters != nil {
		filters = append(filters, idfilters)
	}
	return toFilter(mongodb.And, filters)
}

func (s *Store) GetAppliedConfigurations(ctx context.Context, owner string, query *pb.GetAppliedDeviceConfigurationsRequest, p store.ProccessAppliedDeviceConfigurations) error {
	cur, err := s.Collection(appliedConfigurationsCol).Find(ctx, toAppliedDeviceConfigurationsQuery(owner, query.GetIdFilter(), query.GetDeviceIdFilter()))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) DeleteAppliedConfigurations(ctx context.Context, owner string, query *pb.DeleteAppliedDeviceConfigurationsRequest) (int64, error) {
	res, err := s.Collection(appliedConfigurationsCol).DeleteMany(ctx, toAppliedDeviceConfigurationsQuery(owner, query.GetIdFilter(), nil))
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}
