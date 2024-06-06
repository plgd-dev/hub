package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := store.ValidateAndNormalizeConfiguration(conf, false); err != nil {
		return nil, err
	}
	newConf := conf.Clone()
	if newConf.GetId() == "" {
		newConf.Id = uuid.NewString()
	}
	newConf.Timestamp = time.Now().UnixNano()
	storeConf := store.MakeFirstConfiguration2(newConf)
	_, err := s.Collection(configurationsCol).InsertOne(ctx, storeConf)
	if err != nil {
		return nil, err
	}
	return storeConf.GetLatest()
}

func (s *Store) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := store.ValidateAndNormalizeConfiguration(conf, true); err != nil {
		return nil, err
	}
	filter := bson.M{
		store.IDKey:    conf.GetId(),
		store.OwnerKey: conf.GetOwner(),
	}
	// if version is not set -> set it to latest + 1
	insertVersion := conf.GetVersion() == 0
	if !insertVersion {
		// if is set -> it must be higher than the latest.version
		filter[store.LatestKey+"."+store.VersionKey] = bson.M{"$lt": conf.GetVersion()}
	}

	ts := time.Now().UnixNano()
	latest := bson.M{
		store.VersionKey:   conf.GetVersion(),
		store.ResourcesKey: conf.GetResources(),
		store.TimestampKey: ts,
	}
	if conf.GetName() != "" {
		latest[store.NameKey] = conf.GetName()
	}
	if insertVersion {
		// if version is not set -> set it to latest + 1
		latest[store.VersionKey] = bson.M{
			"$add": bson.A{
				bson.M{"$ifNull": bson.A{"$" + store.LatestKey + "." + store.VersionKey, 0}},
				1,
			},
		}
	}

	setVersions := bson.M{
		"$concatArrays": bson.A{
			bson.M{
				"$ifNull": bson.A{
					"$versions",
					bson.A{},
				},
			},
			bson.A{
				bson.M{
					store.NameKey:      "$latest.name",
					store.VersionKey:   "$latest.version",
					store.ResourcesKey: "$latest.resources",
					store.TimestampKey: "$latest.timestamp",
				},
			},
		},
	}

	update := mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			store.LatestKey: latest,
		}}},
		bson.D{{Key: "$set", Value: bson.M{
			store.VersionsKey: setVersions,
		}}},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(
		bson.M{store.VersionsKey: false},
	)
	result := s.Collection(configurationsCol).FindOneAndUpdate(ctx, filter, update, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	updatedCfg := &store.Configuration{}
	err := result.Decode(&updatedCfg)
	if err != nil {
		return nil, err
	}
	return updatedCfg.GetLatest()
}

func (s *Store) getConfigurationsByFind(ctx context.Context, owner string, idfAlls []string, p store.ProcessConfigurations) error {
	// opts := options.Find().SetProjection(bson.M{store.LatestKey: false})
	cur, err := s.Collection(configurationsCol).Find(ctx, toIdFilterQuery(owner, idfAlls))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func getVersionsPipeline(pl mongo.Pipeline, vf pb.VersionFilter, exclude bool) mongo.Pipeline {
	versions := make([]interface{}, 0, len(vf.Versions())+1)
	for _, version := range vf.Versions() {
		versions = append(versions, version)
	}
	if vf.Latest() {
		versions = append(versions, "$latest.version")
	}
	cond := bson.M{"$in": bson.A{"$$version.version", versions}}
	if exclude {
		cond = bson.M{"$not": cond}
	}
	pl = append(pl, bson.D{{Key: "$addFields", Value: bson.M{
		"versions": bson.M{
			"$filter": bson.M{
				"input": "$versions",
				"as":    "version",
				"cond":  cond,
			},
		},
	}}})
	return pl
}

func (s *Store) getConfigurationsByAggregation(ctx context.Context, owner, id string, vf pb.VersionFilter, p store.ProcessConfigurations) error {
	pl := mongo.Pipeline{bson.D{{Key: "$match", Value: addMatchCondition(owner, id)}}}
	project := bson.M{}
	if len(vf.Versions()) == 0 {
		project[store.VersionsKey] = false
	} else {
		pl = getVersionsPipeline(pl, vf, false)
	}
	if !vf.Latest() {
		project[store.LatestKey] = false
	}
	if len(project) > 0 {
		pl = append(pl, bson.D{{Key: "$project", Value: project}})
	}
	cur, err := s.Collection(configurationsCol).Aggregate(ctx, pl)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) GetConfigurations(ctx context.Context, owner string, query *pb.GetConfigurationsRequest, p store.ProcessConfigurations) error {
	idVersionAll, idVersions := pb.PartitionIDFilter(query.GetIdFilter())
	var errors *multierror.Error
	if len(idVersionAll) > 0 || len(idVersions) == 0 {
		err := s.getConfigurationsByFind(ctx, owner, idVersionAll, p)
		errors = multierror.Append(errors, err)
	}
	if len(idVersions) > 0 {
		for id, vf := range idVersions {
			err := s.getConfigurationsByAggregation(ctx, owner, id, vf, p)
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

func (s *Store) DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) (int64, error) {
	return s.delete(ctx, configurationsCol, owner, query.GetIdFilter())
}

func (s *Store) deleteVersion(ctx context.Context, collection, owner string, id string, vf pb.VersionFilter) error {
	pl := getVersionsPipeline(mongo.Pipeline{}, vf, true)
	// unset latest
	pl = append(pl, bson.D{{Key: "$unset", Value: store.LatestKey}})
	// take last element from versions array as latest (if it exists)
	pl = append(pl, bson.D{{Key: "$set", Value: bson.M{
		store.LatestKey: bson.D{{
			Key:   "$arrayElemAt",
			Value: bson.A{"$versions", -1},
		}},
	}}})
	_, err := s.Collection(collection).UpdateMany(ctx, addMatchCondition(owner, id), pl)
	// TODO: delete document if no versions remain in the array
	return err
}

func (s *Store) deleteDocument(ctx context.Context, collection, owner string, idfAlls []string) error {
	_, err := s.Collection(collection).DeleteMany(ctx, toIdFilterQuery(owner, idfAlls))
	return err
}

func (s *Store) delete(ctx context.Context, collection, owner string, filter []*pb.IDFilter) (int64, error) {
	success := false
	idVersionAll, idVersions := pb.PartitionIDFilter(filter)
	var errors *multierror.Error
	if len(idVersionAll) > 0 || len(idVersions) == 0 {
		err := s.deleteDocument(ctx, collection, owner, idVersionAll)
		if err == nil {
			success = true
		}
		errors = multierror.Append(errors, err)
	}
	if len(idVersions) > 0 {
		for id, vf := range idVersions {
			err := s.deleteVersion(ctx, collection, owner, id, vf)
			if err == nil {
				success = true
			}
			errors = multierror.Append(errors, err)
		}
	}
	return toDeleteResult(errors.ErrorOrNil(), success)
}

func (s *Store) DeleteConfigurations2(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) (int64, error) {
	return s.delete(ctx, configurationsCol, owner, query.GetIdFilter())
}
