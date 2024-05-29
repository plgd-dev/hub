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
)

func toConfiguration(c *store.Configuration) *pb.Configuration {
	conf := &pb.Configuration{
		Id:        c.Id,
		Name:      c.Name,
		Owner:     c.Owner,
		Timestamp: c.Timestamp,
	}
	if len(c.Versions) > 0 {
		conf.Version = c.Versions[0].Version
		conf.Resources = c.Versions[0].Resources
	}
	return conf
}

func (s *Store) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := store.ValidateAndNormalizeConfiguration(conf, false); err != nil {
		return nil, err
	}
	newConf := conf.Clone()
	if newConf.GetId() == "" {
		newConf.Id = uuid.NewString()
	}
	newConf.Timestamp = time.Now().UnixNano()
	storeConf := store.MakeConfiguration(newConf)
	_, err := s.Collection(configurationsCol).InsertOne(ctx, storeConf)
	if err != nil {
		return nil, err
	}
	return toConfiguration(&storeConf), nil
}

func (s *Store) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := store.ValidateAndNormalizeConfiguration(conf, true); err != nil {
		return nil, err
	}
	filter := bson.M{
		store.IDKey:    conf.GetId(),
		store.OwnerKey: conf.GetOwner(),
		store.VersionsKey + "." + store.VersionKey: bson.M{"$ne": conf.GetVersion()},
	}
	ts := time.Now().UnixNano()
	set := bson.M{
		store.TimestampKey: ts,
	}
	update := bson.M{
		"$push": bson.M{
			"versions": store.ConfigurationVersion{
				Version:   conf.GetVersion(),
				Resources: conf.GetResources(),
			},
		},
		"$set": set,
	}
	upd, err := s.Collection(configurationsCol).UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if upd.MatchedCount == 0 {
		return nil, store.ErrNotFound
	}
	return conf, nil
}

func (s *Store) getConfigurationsByFind(ctx context.Context, owner string, idfAlls []string, p store.ProcessConfigurations) error {
	cur, err := s.Collection(configurationsCol).Find(ctx, toIdFilterQuery(owner, idfAlls))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) getConfigurationsByAggregation(ctx context.Context, owner, id string, vf pb.VersionFilter, p store.ProcessConfigurations) error {
	pl := mongo.Pipeline{bson.D{{Key: "$match", Value: addMatchCondition(owner, id)}}}
	pl = getVersionsPipeline(pl, vf, false)
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
