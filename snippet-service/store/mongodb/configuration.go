package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) InsertConfigurations(ctx context.Context, confs ...*store.Configuration) error {
	documents := make([]interface{}, 0, len(confs))
	for _, conf := range confs {
		documents = append(documents, conf)
	}
	_, err := s.Collection(configurationsCol).InsertMany(ctx, documents)
	return err
}

func (s *Store) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	newConf, err := store.ValidateAndNormalizeConfiguration(conf, false)
	if err != nil {
		return nil, err
	}
	if newConf.GetId() == "" {
		newConf.Id = uuid.NewString()
	}
	newConf.Timestamp = time.Now().UnixNano()
	storeConf := store.MakeFirstConfiguration(newConf)
	_, err = s.Collection(configurationsCol).InsertOne(ctx, storeConf)
	if err != nil {
		return nil, err
	}
	return storeConf.GetLatest()
}

func filterConfiguration(conf *pb.Configuration) bson.M {
	filter := bson.M{
		pb.RecordIDKey: conf.GetId(),
		pb.OwnerKey:    conf.GetOwner(),
	}
	if conf.GetVersion() != 0 {
		// if is set -> it must be higher than the $latest.version
		filter[pb.LatestKey+"."+pb.VersionKey] = bson.M{"$lt": conf.GetVersion()}
	}
	return filter
}

func updateConfiguration(conf *pb.Configuration) mongo.Pipeline {
	pl := mongo.Pipeline{
		bson.D{{Key: mongodb.Set, Value: bson.M{
			temporaryLatestKey: store.MakeConfigurationVersion(conf),
		}}},
	}
	// if the version is not forced then look at the version of the last latest configuration
	// and increment it by 1
	if conf.GetVersion() == 0 {
		pl = append(pl,
			bson.D{{Key: mongodb.Set, Value: incrementLatestVersion(temporaryLatestKey + "." + pb.VersionKey)}})
	}
	pl = append(pl,
		bson.D{{Key: mongodb.Set, Value: bson.M{
			pb.LatestKey: "$" + temporaryLatestKey,
		}}},
		bson.D{{Key: mongodb.Unset, Value: bson.A{temporaryLatestKey}}},
		bson.D{{Key: mongodb.Set, Value: bson.M{
			pb.VersionsKey: appendLatestToVersions(),
		}}})
	return pl
}

func (s *Store) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	newConf, err := store.ValidateAndNormalizeConfiguration(conf, true)
	if err != nil {
		return nil, err
	}

	filter := filterConfiguration(newConf)
	newConf.Timestamp = time.Now().UnixNano()
	update := updateConfiguration(newConf)
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(bson.M{pb.VersionsKey: false})
	result := s.Collection(configurationsCol).FindOneAndUpdate(ctx, filter, update, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	updatedCfg := store.Configuration{}
	err = result.Decode(&updatedCfg)
	if err != nil {
		return nil, err
	}
	return updatedCfg.GetLatest()
}

// getConfigurationsByID returns all configurations from documents matched by ID
func (s *Store) getConfigurationsByID(ctx context.Context, owner string, ids []string, p store.ProcessConfigurations) error {
	cur, err := s.Collection(configurationsCol).Find(ctx, toIdFilterQuery(owner, toIdQuery(ids), false))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

// GetLatestConfigurationsByID returns the latest configuration from documents matched by ID
func (s *Store) GetLatestConfigurationsByID(ctx context.Context, owner string, ids []string, p store.ProcessConfigurations) error {
	opt := options.Find().SetProjection(bson.M{pb.VersionsKey: false})
	cur, err := s.Collection(configurationsCol).Find(ctx, toIdFilterQuery(owner, toIdQuery(ids), false), opt)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

// getConfigurationsByAggregation returns conditions matched by ID and versions
func (s *Store) getConfigurationsByAggregation(ctx context.Context, owner, id string, versions []uint64, p store.ProcessConfigurations) error {
	cur, err := s.Collection(configurationsCol).Aggregate(ctx, getPipeline(owner, id, versions))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) GetConfigurations(ctx context.Context, owner string, query *pb.GetConfigurationsRequest, p store.Process[store.Configuration]) error {
	vf := pb.PartitionIDFilter(query.GetIdFilter())
	var errors *multierror.Error
	if len(vf.All) > 0 || vf.IsEmpty() {
		err := s.getConfigurationsByID(ctx, owner, vf.All, p)
		errors = multierror.Append(errors, err)
	}

	if len(vf.Latest) > 0 {
		err := s.GetLatestConfigurationsByID(ctx, owner, vf.Latest, p)
		errors = multierror.Append(errors, err)
	}

	for id, versions := range vf.Versions {
		err := s.getConfigurationsByAggregation(ctx, owner, id, versions, p)
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}

func (s *Store) DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) error {
	return s.delete(ctx, configurationsCol, owner, query.GetIdFilter())
}
