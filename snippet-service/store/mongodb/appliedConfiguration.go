package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) InsertAppliedConfigurations(ctx context.Context, confs ...*store.AppliedConfiguration) error {
	documents := make([]interface{}, 0, len(confs))
	for _, conf := range confs {
		documents = append(documents, conf)
	}
	_, err := s.Collection(appliedConfigurationsCol).InsertMany(ctx, documents)
	return err
}

func (s *Store) replaceAppliedConfiguration(ctx context.Context, newAdc *store.AppliedConfiguration) (*store.AppliedConfiguration, error) {
	var replacedAdc *store.AppliedConfiguration
	filter := bson.M{
		store.OwnerKey:                   newAdc.GetOwner(),
		store.DeviceIDKey:                newAdc.GetDeviceId(),
		store.ConfigurationRelationIDKey: newAdc.GetConfigurationId().GetId(),
	}
	opts := options.FindOneAndReplace().SetReturnDocument(options.Before).SetUpsert(true)
	result := s.Collection(appliedConfigurationsCol).FindOneAndReplace(ctx, filter, newAdc, opts)
	if result.Err() == nil {
		replacedAdc = &store.AppliedConfiguration{}
		if err := result.Decode(replacedAdc); err != nil {
			return nil, err
		}
	}
	if result.Err() != nil && !errors.Is(result.Err(), mongo.ErrNoDocuments) {
		return nil, result.Err()
	}
	return replacedAdc, nil
}

func (s *Store) CreateAppliedConfiguration(ctx context.Context, adc *pb.AppliedConfiguration, force bool) (*pb.AppliedConfiguration, *pb.AppliedConfiguration, error) {
	if err := store.ValidateAppliedConfiguration(adc, false); err != nil {
		return nil, nil, err
	}
	newAdc := store.MakeAppliedConfiguration(adc)
	if newAdc.GetId() == "" {
		newAdc.Id = uuid.NewString()
	}
	newAdc.Timestamp = time.Now().UnixNano()
	if force {
		replacedAdc, err := s.replaceAppliedConfiguration(ctx, &newAdc)
		if err != nil {
			return nil, nil, err
		}
		return newAdc.GetAppliedConfiguration(), replacedAdc.GetAppliedConfiguration(), nil
	}
	if _, err := s.Collection(appliedConfigurationsCol).InsertOne(ctx, &newAdc); err != nil {
		return nil, nil, err
	}
	return newAdc.GetAppliedConfiguration(), nil, nil
}

func (s *Store) UpdateAppliedConfiguration(ctx context.Context, adc *pb.AppliedConfiguration) (*pb.AppliedConfiguration, error) {
	err := store.ValidateAppliedConfiguration(adc, true)
	if err != nil {
		return nil, err
	}
	newAdc := store.MakeAppliedConfiguration(adc)
	filter := bson.M{
		store.IDKey:    newAdc.GetId(),
		store.OwnerKey: newAdc.GetOwner(),
		// TODO: should we also match configurationID, deviceID or is update allowed to change those?
	}
	newAdc.Timestamp = time.Now().UnixNano()
	opts := options.FindOneAndReplace().SetReturnDocument(options.After)
	result := s.Collection(appliedConfigurationsCol).FindOneAndReplace(ctx, filter, &newAdc, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}
	updatedAdc := store.AppliedConfiguration{}
	if err = result.Decode(&updatedAdc); err != nil {
		return nil, err
	}
	return updatedAdc.GetAppliedConfiguration(), nil
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

func (s *Store) GetAppliedConfigurations(ctx context.Context, owner string, query *pb.GetAppliedConfigurationsRequest, p store.ProccessAppliedConfigurations) error {
	configurationIdFilter := pb.PartitionIDFilter(query.GetConfigurationIdFilter())
	conditionIdFilter := pb.PartitionIDFilter(query.GetConditionIdFilter())
	filter := toAppliedDeviceConfigurationsQuery(owner, query.GetIdFilter(), query.GetDeviceIdFilter(), configurationIdFilter, conditionIdFilter)
	cur, err := s.Collection(appliedConfigurationsCol).Find(ctx, filter)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) DeleteAppliedConfigurations(ctx context.Context, owner string, query *pb.DeleteAppliedConfigurationsRequest) (int64, error) {
	res, err := s.Collection(appliedConfigurationsCol).DeleteMany(ctx, toAppliedDeviceConfigurationsQuery(owner, query.GetIdFilter(), nil, pb.VersionFilter{}, pb.VersionFilter{}))
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (s *Store) UpdateAppliedConfigurationResource(ctx context.Context, owner string, query store.UpdateAppliedConfigurationResourceRequest) (*pb.AppliedConfiguration, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	filter := bson.M{
		store.IDKey:                              query.AppliedConfigurationID,
		store.ResourcesKey + "." + store.HrefKey: query.Resource.GetHref(),
	}
	if owner != "" {
		filter[store.OwnerKey] = owner
	}
	statusFilter := bson.A{}
	if len(query.StatusFilter) > 0 {
		for _, status := range query.StatusFilter {
			statusFilter = append(statusFilter, status.String())
		}
	}

	matchResourceCond := func(alias string) bson.M {
		cond := bson.M{"$eq": bson.A{"$$" + alias + "." + store.HrefKey, query.Resource.GetHref()}}
		if len(statusFilter) > 0 {
			cond = bson.M{
				mongodb.And: bson.A{
					cond,
					bson.M{mongodb.In: bson.A{"$$" + alias + "." + store.StatusKey, statusFilter}},
				},
			}
		}
		return cond
	}

	const matchFoundKey = "__matchFound"
	updatedTimestamp := time.Now().UnixNano()
	update := mongo.Pipeline{
		// check if we have a resource with the given href and status
		bson.D{{Key: mongodb.Set, Value: bson.M{
			matchFoundKey: bson.M{"$gt": bson.A{
				bson.M{
					"$size": bson.M{
						"$filter": bson.M{
							"input": "$" + store.ResourcesKey,
							"as":    "elem",
							"cond":  matchResourceCond("elem"),
						},
					},
				}, 0,
			}},
		}}},
	}
	// replace the resource with the new one
	update = append(update, bson.D{{Key: mongodb.Set, Value: bson.M{
		store.ResourcesKey: bson.M{
			"$map": bson.M{
				"input": "$" + store.ResourcesKey,
				"as":    "elem",
				"in": bson.M{
					"$cond": bson.M{
						"if":   matchResourceCond("elem"),
						"then": query.Resource,
						"else": "$$elem",
					},
				},
			},
		},
	}}})

	// update the timestamp and the condition relation if we have a matched resource
	set := bson.M{
		store.TimestampKey: bson.M{
			"$cond": bson.M{
				"if":   "$" + matchFoundKey,
				"then": updatedTimestamp,
				"else": "$" + store.TimestampKey,
			},
		},
	}
	if query.AppliedCondition != nil {
		set[store.ConditionRelationIDKey] = bson.M{
			"$cond": bson.M{
				"if":   "$" + matchFoundKey,
				"then": query.AppliedCondition.GetId(),
				"else": "$" + store.ConditionRelationIDKey,
			},
		}
		set[store.ConditionRelationVersionKey] = bson.M{
			"$cond": bson.M{
				"if":   "$" + matchFoundKey,
				"then": query.AppliedCondition.GetVersion(),
				"else": "$" + store.ConditionRelationVersionKey,
			},
		}
	}

	// update the timestamp and the condition relation if we have a matched resource
	update = append(update, bson.D{{Key: mongodb.Set, Value: set}})

	// remove the __matchFound field
	update = append(update, bson.D{{Key: mongodb.Unset, Value: bson.A{matchFoundKey}}})

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := s.Collection(appliedConfigurationsCol).FindOneAndUpdate(ctx, filter, update, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: %w", store.ErrNotFound, fmt.Errorf("no applied configuration(%v) with resource(%v)", query.AppliedConfigurationID, query.Resource.GetHref()))
		}
		return nil, result.Err()
	}

	updatedAppliedCfg := store.AppliedConfiguration{}
	err := result.Decode(&updatedAppliedCfg)
	if err != nil {
		return nil, err
	}
	// check timestamp to know whether the resource was updated or not
	if updatedAppliedCfg.GetTimestamp() != updatedTimestamp {
		return nil, fmt.Errorf("%w: %w", store.ErrNotModified, fmt.Errorf("applied configuration(%v) was not updated", query.AppliedConfigurationID))
	}

	return updatedAppliedCfg.GetAppliedConfiguration(), nil
}

func (s *Store) GetExpiredAppliedConfigurationResourceUpdates(ctx context.Context, p store.ProccessAppliedConfigurations) (int64, error) {
	validUntil := time.Now().UnixNano()
	pl := mongo.Pipeline{
		// match resources that have a resource in pending state and expired
		bson.D{{Key: mongodb.Match, Value: bson.M{
			store.ResourcesKey: bson.M{
				"$elemMatch": bson.M{
					store.StatusKey:  pb.AppliedConfiguration_Resource_PENDING.String(),
					store.ValidUntil: bson.M{"$lte": validUntil},
				},
			},
		}}},
	}
	// project only the resources that are expired
	pl = append(pl, bson.D{{Key: mongodb.Set, Value: bson.M{
		store.ResourcesKey: bson.M{
			"$filter": bson.M{
				"input": "$" + store.ResourcesKey,
				"as":    "elem",
				"cond": bson.M{
					mongodb.And: bson.A{
						bson.M{"$eq": bson.A{"$$elem." + store.StatusKey, pb.AppliedConfiguration_Resource_PENDING.String()}},
						bson.M{"$gt": bson.A{"$$elem." + store.ValidUntil, 0}},
						bson.M{"$lte": bson.A{"$$elem." + store.ValidUntil, validUntil}},
					},
				},
			},
		},
	}}})

	cur, err := s.Collection(appliedConfigurationsCol).Aggregate(ctx, pl)
	if err != nil {
		return 0, err
	}
	return validUntil, processCursor(ctx, cur, p)
}
