package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) InsertAppliedConfigurations(ctx context.Context, confs ...*pb.AppliedDeviceConfiguration) error {
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

func toAppliedDeviceConfigurationsVersionFilter(idKey, versionsKey string, vf pb.VersionFilter) interface{} {
	filters := make([]interface{}, 0, 2)
	if len(vf.All) > 0 {
		// all ids
		if vf.All[0] == "" {
			return bson.M{idKey: bson.M{mongodb.Exists: true}}
		}
		cidFilter := inArrayQuery(idKey, vf.All)
		if cidFilter != nil {
			filters = append(filters, cidFilter)
		}
	}
	versionFilters := make([]interface{}, 0, len(vf.Versions))
	for id, versions := range vf.Versions {
		version := bson.M{
			versionsKey: bson.M{mongodb.In: versions},
		}
		if id != "" {
			version[idKey] = id
		}
		// id must match and version must be in the list of versions
		versionFilters = append(versionFilters, version)
	}
	if len(versionFilters) > 0 {
		filters = append(filters, toFilter(mongodb.Or, versionFilters))
	}
	return toFilter(mongodb.Or, filters)
}

func toAppliedDeviceConfigurationsIdFilter(idFilter, deviceIdFilter []string, configurationIdFilter, conditionIdFilter pb.VersionFilter) interface{} {
	filters := make([]interface{}, 0, 4)
	idf := inArrayQuery(store.IDKey, strings.Unique(idFilter))
	if idf != nil {
		filters = append(filters, idf)
	}
	dif := inArrayQuery(store.DeviceIDKey, strings.Unique(deviceIdFilter))
	if dif != nil {
		filters = append(filters, dif)
	}
	confif := toAppliedDeviceConfigurationsVersionFilter(store.ConfigurationRelationIDKey, store.ConfigurationRelationVersionKey, configurationIdFilter)
	if confif != nil {
		filters = append(filters, confif)
	}
	condif := toAppliedDeviceConfigurationsVersionFilter(store.ConditionRelationIDKey, store.ConditionRelationVersionKey, conditionIdFilter)
	if condif != nil {
		filters = append(filters, condif)
	}
	return toFilter(mongodb.Or, filters)
}

func toAppliedDeviceConfigurationsQuery(owner string, idFilter, deviceIdFilter []string, configurationIdFilter, conditionIdFilter pb.VersionFilter) interface{} {
	filters := make([]interface{}, 0, 2)
	if owner != "" {
		filters = append(filters, bson.D{{Key: store.OwnerKey, Value: owner}})
	}
	idfilters := toAppliedDeviceConfigurationsIdFilter(idFilter, deviceIdFilter, configurationIdFilter, conditionIdFilter)
	if idfilters != nil {
		filters = append(filters, idfilters)
	}
	return toFilterQuery(mongodb.And, filters)
}

func (s *Store) GetAppliedConfigurations(ctx context.Context, owner string, query *pb.GetAppliedDeviceConfigurationsRequest, p store.ProccessAppliedDeviceConfigurations) error {
	configurationIdFilter := pb.PartitionIDFilter(query.GetConfigurationIdFilter())
	conditionIdFilter := pb.PartitionIDFilter(query.GetConditionIdFilter())
	filter := toAppliedDeviceConfigurationsQuery(owner, query.GetIdFilter(), query.GetDeviceIdFilter(), configurationIdFilter, conditionIdFilter)
	cur, err := s.Collection(appliedConfigurationsCol).Find(ctx, filter)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) DeleteAppliedConfigurations(ctx context.Context, owner string, query *pb.DeleteAppliedDeviceConfigurationsRequest) (int64, error) {
	res, err := s.Collection(appliedConfigurationsCol).DeleteMany(ctx, toAppliedDeviceConfigurationsQuery(owner, query.GetIdFilter(), nil, pb.VersionFilter{}, pb.VersionFilter{}))
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (s *Store) UpdateAppliedConfigurationPendingResource(ctx context.Context, owner string, query store.UpdateAppliedConfigurationPendingResourceRequest) error {
	filter := bson.M{
		store.IDKey: query.AppliedConfigurationID,
		store.ResourcesKey + "." + store.StatusKey: pb.AppliedDeviceConfiguration_Resource_PENDING.String(),
	}
	if owner != "" {
		filter[store.OwnerKey] = owner
	}

	update := bson.M{
		mongodb.Set: bson.M{
			store.ResourcesKey + ".$[elem]": query.Resource,
		},
	}
	optFilters := bson.M{
		"elem." + store.HrefKey:   query.Resource.GetHref(),
		"elem." + store.StatusKey: pb.AppliedDeviceConfiguration_Resource_PENDING.String(),
	}
	res, err := s.Collection(appliedConfigurationsCol).UpdateOne(ctx, filter, update, &options.UpdateOptions{
		ArrayFilters: &options.ArrayFilters{
			Filters: bson.A{optFilters},
		},
	})
	if err == nil && res.ModifiedCount == 0 {
		return fmt.Errorf("%w: %w", store.ErrNotFound, fmt.Errorf("no applied configuration(%v) with resource(%v) in pending status", query.AppliedConfigurationID, query.Resource.GetHref()))
	}
	return err
}
