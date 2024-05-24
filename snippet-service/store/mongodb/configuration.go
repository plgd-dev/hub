package mongodb

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Store) CreateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := conf.ValidateAndNormalize(); err != nil {
		return nil, err
	}

	_, err := s.Collection(configurationsCol).InsertOne(ctx, store.MakeConfiguration(conf))
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (s *Store) UpdateConfiguration(ctx context.Context, conf *pb.Configuration) (*pb.Configuration, error) {
	if err := conf.ValidateAndNormalize(); err != nil {
		return nil, err
	}
	upd, err := s.Collection(configurationsCol).UpdateOne(ctx,
		bson.M{
			"_id":          conf.GetId(),
			store.OwnerKey: conf.GetOwner(),
			store.VersionsKey + "." + store.VersionKey: bson.M{"$ne": conf.GetVersion()},
		},
		bson.M{"$push": bson.M{
			"versions": store.ConfigurationVersion{
				Version:   conf.GetVersion(),
				Resources: conf.GetResources(),
			},
		}},
	)
	if err != nil {
		return nil, err
	}
	if upd.MatchedCount == 0 {
		return nil, store.ErrNotFound
	}
	return conf, nil
}

func compareIdFilter(i, j *pb.IDFilter) int {
	if i.GetId() != j.GetId() {
		return strings.Compare(i.GetId(), j.GetId())
	}
	if i.GetAll() {
		if j.GetAll() {
			return 0
		}
		return -1
	}
	if i.GetLatest() {
		if j.GetLatest() {
			return 0
		}
		if j.GetAll() {
			return 1
		}
		return -1
	}
	if j.GetAll() || j.GetLatest() {
		return 1
	}
	return cmp.Compare(i.GetValue(), j.GetValue())
}

func checkEmptyIdFilter(idfilter []*pb.IDFilter) []*pb.IDFilter {
	// if an empty query is provided, return all
	if len(idfilter) == 0 {
		return nil
	}
	slices.SortFunc(idfilter, compareIdFilter)
	// if the first filter is All, we can ignore all other filters
	first := idfilter[0]
	if first.GetId() == "" && first.GetAll() {
		return nil
	}
	return idfilter
}

func normalizeIdFilter(idfilter []*pb.IDFilter) []*pb.IDFilter {
	idfilter = checkEmptyIdFilter(idfilter)
	if len(idfilter) == 0 {
		return nil
	}

	updatedFilter := make([]*pb.IDFilter, 0)
	var idAll bool
	var idLatest bool
	var idValue bool
	var idValueVersion uint64
	setNextLatest := func(idf *pb.IDFilter) {
		// we already have the latest filter
		if idLatest {
			// skip
			return
		}
		idLatest = true
		updatedFilter = append(updatedFilter, idf)
	}
	setNextValue := func(idf *pb.IDFilter) {
		value := idf.GetValue()
		if idValue && value == idValueVersion {
			// skip
			return
		}
		idValue = true
		idValueVersion = value
		updatedFilter = append(updatedFilter, idf)
	}
	prevID := ""
	for _, idf := range idfilter {
		if idf.GetId() != prevID {
			idAll = idf.GetAll()
			idLatest = idf.GetLatest()
			idValue = !idAll && !idLatest
			idValueVersion = idf.GetValue()
			updatedFilter = append(updatedFilter, idf)
		}

		if idAll {
			goto next
		}

		if idf.GetLatest() {
			setNextLatest(idf)
			goto next
		}

		setNextValue(idf)

	next:
		prevID = idf.GetId()
	}
	return updatedFilter
}

type versionFilter struct {
	latest   bool
	versions []uint64
}

func partitionQuery(idfilter []*pb.IDFilter) ([]string, map[string]versionFilter) {
	idFilter := normalizeIdFilter(idfilter)
	if len(idFilter) == 0 {
		return nil, nil
	}
	idVersionAll := make([]string, 0)
	idVersions := make(map[string]versionFilter, 0)
	hasAllIdsLatest := func() bool {
		vf, ok := idVersions[""]
		return ok && vf.latest
	}
	hasAllIdsVersion := func(version uint64) bool {
		vf, ok := idVersions[""]
		return ok && slices.Contains(vf.versions, version)
	}
	for _, idf := range idFilter {
		if idf.GetAll() {
			idVersionAll = append(idVersionAll, idf.GetId())
			continue
		}
		vf := idVersions[idf.GetId()]
		if idf.GetLatest() {
			if hasAllIdsLatest() {
				continue
			}
			vf.latest = true
			idVersions[idf.GetId()] = vf
			continue
		}
		version := idf.GetValue()
		if hasAllIdsVersion(version) {
			continue
		}
		vf.versions = append(vf.versions, version)
		idVersions[idf.GetId()] = vf
	}
	return idVersionAll, idVersions
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

func processCursor(ctx context.Context, cr *mongo.Cursor, h store.GetConfigurationsFunc) error {
	if h == nil {
		return nil
	}
	var errors *multierror.Error
	i := store.MongoIterator[store.Configuration]{
		Cursor: cr,
	}
	err := h(ctx, &i)
	errors = multierror.Append(errors, err)
	errClose := cr.Close(ctx)
	errors = multierror.Append(errors, errClose)
	return errors.ErrorOrNil()
}

func (s *Store) getConfigurationsByFind(ctx context.Context, owner string, idfAlls []string, h store.GetConfigurationsFunc) error {
	cur, err := s.Collection(configurationsCol).Find(ctx, toIdFilterQuery(owner, idfAlls))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, h)
}

func addMatchCondition(owner, id string) bson.D {
	match := bson.D{
		{Key: store.VersionsKey + ".0", Value: bson.M{"$exists": true}},
	}
	if id != "" {
		match = append(match, bson.E{Key: "_id", Value: id})
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

func getVersionsPipeline(pl mongo.Pipeline, vf versionFilter, exclude bool) mongo.Pipeline {
	versions := make([]interface{}, 0, len(vf.versions)+1)
	for _, version := range vf.versions {
		versions = append(versions, version)
	}

	if vf.latest {
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

func (s *Store) getConfigurationsByAggregation(ctx context.Context, owner, id string, vf versionFilter, h store.GetConfigurationsFunc) error {
	pl := mongo.Pipeline{bson.D{{Key: "$match", Value: addMatchCondition(owner, id)}}}
	pl = getVersionsPipeline(pl, vf, false)
	cur, err := s.Collection(configurationsCol).Aggregate(ctx, pl)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, h)
}

func (s *Store) GetConfigurations(ctx context.Context, owner string, query *pb.GetConfigurationsRequest, h store.GetConfigurationsFunc) error {
	idVersionAll, idVersions := partitionQuery(query.GetIdFilter())
	var errors *multierror.Error
	if len(idVersionAll) > 0 || len(idVersions) == 0 {
		err := s.getConfigurationsByFind(ctx, owner, idVersionAll, h)
		errors = multierror.Append(errors, err)
	}
	if len(idVersions) > 0 {
		for id, vf := range idVersions {
			err := s.getConfigurationsByAggregation(ctx, owner, id, vf, h)
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

func (s *Store) removeDocument(ctx context.Context, owner string, idfAlls []string) error {
	_, err := s.Collection(configurationsCol).DeleteMany(ctx, toIdFilterQuery(owner, idfAlls))
	return err
}

func (s *Store) removeVersion(ctx context.Context, owner string, id string, vf versionFilter) error {
	pl := getVersionsPipeline(mongo.Pipeline{}, vf, true)
	if vf.latest {
		pl = append(pl, bson.D{{Key: "$unset", Value: "latestVersion"}})
	}
	_, err := s.Collection(configurationsCol).UpdateMany(ctx, addMatchCondition(owner, id), pl)
	// TODO: delete document if no versions remain in the array
	return err
}

func (s *Store) DeleteConfigurations(ctx context.Context, owner string, query *pb.DeleteConfigurationsRequest) (int64, error) {
	success := false
	idVersionAll, idVersions := partitionQuery(query.GetIdFilter())
	var errors *multierror.Error
	if len(idVersionAll) > 0 || len(idVersions) == 0 {
		err := s.removeDocument(ctx, owner, idVersionAll)
		if err == nil {
			success = true
		}
		errors = multierror.Append(errors, err)
	}
	if len(idVersions) > 0 {
		for id, vf := range idVersions {
			err := s.removeVersion(ctx, owner, id, vf)
			if err == nil {
				success = true
			}
			errors = multierror.Append(errors, err)
		}
	}
	err := errors.ErrorOrNil()
	if err != nil {
		if success {
			return 2, err
		}
		return 0, err
	}
	return 1, nil
}
