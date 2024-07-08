package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	temporaryLatestKey = "__latest"
)

func addMatchCondition(owner string, id string, notEmpty bool) bson.D {
	match := bson.D{
		{Key: pb.VersionsKey + ".0", Value: bson.M{mongodb.Exists: notEmpty}},
	}
	if id != "" {
		match = append(match, bson.E{Key: pb.RecordIDKey, Value: id})
	}
	if owner != "" {
		match = append(match, bson.E{Key: pb.OwnerKey, Value: owner})
	}
	return match
}

func appendLatestToVersions() bson.M {
	return bson.M{
		"$concatArrays": bson.A{
			bson.M{
				"$ifNull": bson.A{
					"$" + pb.VersionsKey,
					bson.A{},
				},
			},
			bson.A{"$" + pb.LatestKey},
		},
	}
}

func incrementLatestVersion(key string) bson.M {
	return bson.M{
		key: bson.M{
			"$add": bson.A{
				bson.M{"$ifNull": bson.A{"$" + pb.LatestKey + "." + pb.VersionKey, 0}},
				1,
			},
		},
	}
}

func getVersionsPipeline(pl mongo.Pipeline, versions []uint64, latest, exclude bool) mongo.Pipeline {
	vfilter := make([]interface{}, 0, len(versions)+1)
	for _, version := range versions {
		vfilter = append(vfilter, version)
	}
	if latest {
		vfilter = append(vfilter, "$"+pb.LatestKey+"."+pb.VersionKey)
	}
	if len(vfilter) == 0 {
		return pl
	}
	cond := bson.M{mongodb.In: bson.A{"$$version." + pb.VersionKey, vfilter}}
	if exclude {
		cond = bson.M{"$not": cond}
	}
	pl = append(pl, bson.D{{Key: "$addFields", Value: bson.M{
		pb.VersionsKey: bson.M{
			"$filter": bson.M{
				"input": "$" + pb.VersionsKey,
				"as":    "version",
				"cond":  cond,
			},
		},
	}}})
	return pl
}

func getPipeline(owner, id string, versions []uint64) mongo.Pipeline {
	pl := mongo.Pipeline{bson.D{{Key: mongodb.Match, Value: addMatchCondition(owner, id, true)}}}
	project := bson.M{
		pb.LatestKey: false,
	}
	pl = getVersionsPipeline(pl, versions, false, false)
	pl = append(pl, bson.D{{Key: "$project", Value: project}})
	return pl
}

func inArrayQuery(key string, values []string) bson.M {
	filter := bson.A{}
	for _, v := range values {
		if v == "" {
			continue
		}
		filter = append(filter, v)
	}
	if len(filter) == 0 {
		return nil
	}
	return bson.M{key: bson.D{{Key: mongodb.In, Value: filter}}}
}

func toIdQuery(ids []string) bson.M {
	return inArrayQuery(pb.RecordIDKey, ids)
}

func toFilter(op string, filters []interface{}) interface{} {
	if len(filters) == 0 {
		return nil
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return bson.M{op: filters}
}

func toFilterQuery(op string, filters []interface{}) interface{} {
	filter := toFilter(op, filters)
	if filter == nil {
		return bson.M{}
	}
	return filter
}

func toIdFilterQuery(owner string, idfilter bson.M, emptyVersions bool) interface{} {
	filters := make([]interface{}, 0, 3)
	if owner != "" {
		filters = append(filters, bson.D{{Key: pb.OwnerKey, Value: owner}})
	}
	if idfilter != nil {
		filters = append(filters, idfilter)
	}
	if emptyVersions {
		filters = append(filters, bson.D{{Key: pb.VersionsKey + ".0", Value: bson.M{mongodb.Exists: false}}})
	}
	return toFilterQuery(mongodb.And, filters)
}

func processCursor[T any](ctx context.Context, cr *mongo.Cursor, process store.Process[T]) error {
	var errors *multierror.Error
	iter := store.MongoIterator[T]{
		Cursor: cr,
	}
	for {
		var stored T
		if !iter.Next(ctx, &stored) {
			break
		}
		err := process(&stored)
		if err != nil {
			errors = multierror.Append(errors, err)
			break
		}
	}
	errors = multierror.Append(errors, iter.Err())
	errClose := cr.Close(ctx)
	errors = multierror.Append(errors, errClose)
	return errors.ErrorOrNil()
}

func toDeleteResult(err error, partialSuccess bool) error {
	if err != nil {
		if partialSuccess {
			return fmt.Errorf("%w: %w", store.ErrPartialDelete, err)
		}
		return err
	}
	return nil
}

func (s *Store) deleteVersion(ctx context.Context, collection, owner string, id string, versions []uint64) error {
	pl := getVersionsPipeline(mongo.Pipeline{}, versions, false, true)
	// take last element from versions array as latest (if it exists)
	pl = append(pl, bson.D{{Key: mongodb.Set, Value: bson.M{
		pb.LatestKey: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$" + pb.VersionsKey, -1}}},
	}}})
	_, err := s.Collection(collection).UpdateMany(ctx, toIdFilterQuery(owner, toIdQuery([]string{id}), false), pl)
	return err
}

func (s *Store) deleteLatestVersion(ctx context.Context, collection, owner string, ids []string) error {
	pl := getVersionsPipeline(mongo.Pipeline{}, nil, true, true)
	// take last element from versions array as latest (if it exists)
	pl = append(pl, bson.D{{Key: mongodb.Set, Value: bson.M{
		pb.LatestKey: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$" + pb.VersionsKey, -1}}},
	}}})
	_, err := s.Collection(collection).UpdateMany(ctx, toIdFilterQuery(owner, toIdQuery(ids), false), pl)
	return err
}

func (s *Store) deleteDocuments(ctx context.Context, collection, owner string, ids []string) error {
	_, err := s.Collection(collection).DeleteMany(ctx, toIdFilterQuery(owner, toIdQuery(ids), false))
	return err
}

func (s *Store) delete(ctx context.Context, collection, owner string, idfilter []*pb.IDFilter) error {
	success := false
	vf := pb.PartitionIDFilter(idfilter)
	var errors *multierror.Error
	if len(vf.All) > 0 || vf.IsEmpty() {
		err := s.deleteDocuments(ctx, collection, owner, vf.All)
		success = success || err == nil
		errors = multierror.Append(errors, err)
	}

	if len(vf.Latest) > 0 {
		err := s.deleteLatestVersion(ctx, collection, owner, vf.Latest)
		success = success || err == nil
		errors = multierror.Append(errors, err)
	}

	for id, versions := range vf.Versions {
		err := s.deleteVersion(ctx, collection, owner, id, versions)
		success = success || err == nil
		errors = multierror.Append(errors, err)
	}

	// delete documents with empty versions
	if len(vf.Latest) > 0 || len(vf.Versions) > 0 {
		_, err := s.Collection(collection).DeleteMany(ctx, toIdFilterQuery(owner, nil, true))
		errors = multierror.Append(errors, err)
	}
	return toDeleteResult(errors.ErrorOrNil(), success)
}
