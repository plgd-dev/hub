package mongodb

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func addMatchCondition(owner, id string) bson.D {
	match := bson.D{
		{Key: store.VersionsKey + ".0", Value: bson.M{"$exists": true}},
	}
	if id != "" {
		match = append(match, bson.E{Key: store.IDKey, Value: id})
	}
	if owner != "" {
		match = append(match, bson.E{Key: store.OwnerKey, Value: owner})
	}
	return match
}

func addLatestVersionField() bson.D {
	return bson.D{{Key: "$addFields", Value: bson.M{
		"latestVersion": bson.M{"$reduce": bson.M{
			"input":        "$versions",
			"initialValue": bson.M{"version": 0},
			"in": bson.M{
				"$cond": bson.A{
					bson.M{"$gte": bson.A{"$$this.version", "$$value.version"}},
					"$$this.version",
					"$$value.version",
				},
			},
		}},
	}}}
}

func getVersionsPipeline(pl mongo.Pipeline, vf pb.VersionFilter, exclude bool) mongo.Pipeline {
	versions := make([]interface{}, 0, len(vf.Versions())+1)
	for _, version := range vf.Versions() {
		versions = append(versions, version)
	}

	if vf.Latest() {
		pl = append(pl, addLatestVersionField())
		versions = append(versions, "$latestVersion")
	}

	if len(versions) > 0 {
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
	}
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
	if vf.Latest() {
		pl = append(pl, bson.D{{Key: "$unset", Value: "latestVersion"}})
	}
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
