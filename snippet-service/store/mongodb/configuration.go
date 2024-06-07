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
	storeConf := store.MakeFirstConfiguration(newConf)
	_, err := s.Collection(configurationsCol).InsertOne(ctx, storeConf)
	if err != nil {
		return nil, err
	}
	return storeConf.GetLatest()
}

func filterConfiguration(conf *pb.Configuration) bson.M {
	filter := bson.M{
		store.IDKey:    conf.GetId(),
		store.OwnerKey: conf.GetOwner(),
	}
	if conf.GetVersion() != 0 {
		// if is set -> it must be higher than the $latest.version
		filter[store.LatestKey+"."+store.VersionKey] = bson.M{"$lt": conf.GetVersion()}
	}
	return filter
}

func latestConfiguration(conf *pb.Configuration) bson.M {
	ts := time.Now().UnixNano()
	latest := bson.M{
		store.VersionKey:   conf.GetVersion(),
		store.ResourcesKey: conf.GetResources(),
		store.TimestampKey: ts,
	}
	if conf.GetName() != "" {
		latest[store.NameKey] = conf.GetName()
	}
	if conf.GetVersion() == 0 {
		// if version is not set -> set it to $latest.version + 1
		latest[store.VersionKey] = incrementLatestVersion()
	}
	return latest
}

func updateConfiguration(conf *pb.Configuration) mongo.Pipeline {
	setVersions := appendLatestToVersions([]string{store.NameKey, store.VersionKey, store.ResourcesKey, store.TimestampKey})
	return mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			store.LatestKey: latestConfiguration(conf),
		}}},
		bson.D{{Key: "$set", Value: bson.M{
			store.VersionsKey: setVersions,
		}}},
	}
}

func (s *Store) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := store.ValidateAndNormalizeConfiguration(conf, true); err != nil {
		return nil, err
	}

	filter := filterConfiguration(conf)
	update := updateConfiguration(conf)
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(bson.M{store.VersionsKey: false})
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
	cur, err := s.Collection(configurationsCol).Find(ctx, toIdFilterQuery(owner, idfAlls))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) getConfigurationsByAggregation(ctx context.Context, owner, id string, vf pb.VersionFilter, p store.ProcessConfigurations) error {
	cur, err := s.Collection(configurationsCol).Aggregate(ctx, getPipeline(owner, id, vf))
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
