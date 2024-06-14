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
	newConf, err := store.ValidateAndNormalizeConfiguration(conf, true)
	if err != nil {
		return nil, err
	}

	filter := filterConfiguration(newConf)
	update := updateConfiguration(newConf)
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(bson.M{store.VersionsKey: false})
	result := s.Collection(configurationsCol).FindOneAndUpdate(ctx, filter, update, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	updatedCfg := &store.Configuration{}
	err = result.Decode(&updatedCfg)
	if err != nil {
		return nil, err
	}
	return updatedCfg.GetLatest()
}

// getConfigurationsByID returns all configurations from documents matched by ID
func (s *Store) getConfigurationsByID(ctx context.Context, owner string, ids []string, p store.ProcessConfigurations) error {
	cur, err := s.Collection(configurationsCol).Find(ctx, toFilterQuery(owner, toIdQuery(ids), false))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

// getLatestConfigurationsByID returns the latest configuration from documents matched by ID
func (s *Store) getLatestConfigurationsByID(ctx context.Context, owner string, ids []string, p store.ProcessConfigurations) error {
	opt := options.Find().SetProjection(bson.M{store.VersionsKey: false})
	cur, err := s.Collection(configurationsCol).Find(ctx, toFilterQuery(owner, toIdQuery(ids), false), opt)
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
		err := s.getLatestConfigurationsByID(ctx, owner, vf.Latest, p)
		errors = multierror.Append(errors, err)
	}

	// TODO: check with Jozef if this acceptable, we can duplicates if we have the same version by number and as latest
	for id, versions := range vf.Versions {
		err := s.getConfigurationsByAggregation(ctx, owner, id, versions, p)
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}

func (s *Store) DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) (int64, error) {
	return s.delete(ctx, configurationsCol, owner, query.GetIdFilter())
}
