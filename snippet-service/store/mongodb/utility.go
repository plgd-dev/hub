package mongodb

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func addMatchCondition(owner, id string, notEmpty bool) bson.D {
	match := bson.D{
		{Key: store.VersionsKey + ".0", Value: bson.M{"$exists": notEmpty}},
	}
	if id != "" {
		match = append(match, bson.E{Key: store.IDKey, Value: id})
	}
	if owner != "" {
		match = append(match, bson.E{Key: store.OwnerKey, Value: owner})
	}
	return match
}

func appendLatestToVersions(fields []string) bson.M {
	latest := bson.M{}
	for _, field := range fields {
		latest[field] = "$latest." + field
	}
	return bson.M{
		"$concatArrays": bson.A{
			bson.M{
				"$ifNull": bson.A{
					"$versions",
					bson.A{},
				},
			},
			bson.A{latest},
		},
	}
}

func incrementLatestVersion() bson.M {
	return bson.M{
		"$add": bson.A{
			bson.M{"$ifNull": bson.A{"$" + store.LatestKey + "." + store.VersionKey, 0}},
			1,
		},
	}
}

func getPipeline(owner, id string, vf pb.VersionFilter) mongo.Pipeline {
	pl := mongo.Pipeline{bson.D{{Key: "$match", Value: addMatchCondition(owner, id, true)}}}
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
	return pl
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

func toIdFilterQuery(owner string, idfAlls []string) interface{} {
	filters := make([]interface{}, 0, 2)
	if owner != "" {
		filters = append(filters, bson.D{{Key: store.OwnerKey, Value: owner}})
	}
	if len(idfAlls) > 0 {
		idfilter := make([]bson.D, 0, len(idfAlls))
		for _, idfall := range idfAlls {
			idfilter = append(idfilter, bson.D{{Key: "_id", Value: idfall}})
		}
		filters = append(filters, bson.M{"$or": idfilter})
	}
	if len(filters) == 0 {
		return bson.D{}
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return bson.M{"$and": filters}
}

func processCursor[T any](ctx context.Context, cr *mongo.Cursor, p store.ProcessIterator[T]) error {
	if p == nil {
		return nil
	}
	var errors *multierror.Error
	i := store.MongoIterator[T]{
		Cursor: cr,
	}
	err := p(ctx, &i)
	errors = multierror.Append(errors, err)
	errClose := cr.Close(ctx)
	errors = multierror.Append(errors, errClose)
	return errors.ErrorOrNil()
}

const (
	DeleteFailed         = 0
	DeleteSuccess        = 1
	DeletePartialSuccess = 2
)

func toDeleteResult(err error, partialSuccess bool) (int64, error) {
	if err != nil {
		if partialSuccess {
			return DeletePartialSuccess, err
		}
		return DeleteFailed, err
	}
	return DeleteSuccess, nil
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
	_, err := s.Collection(collection).UpdateMany(ctx, addMatchCondition(owner, id, true), pl)
	if err != nil {
		return err
	}
	_, err = s.Collection(collection).DeleteMany(ctx, addMatchCondition(owner, id, false))
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
