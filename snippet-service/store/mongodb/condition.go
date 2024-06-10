package mongodb

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Condition -> podmienka za akych okolnosti sa aplikuje
	- id (identifikator) (user nevie menit)
	- verzia (pri update sa verzia inkremente)
		- pri ukladani checknem v DB ze predchadzajuca verzia je o 1 mensie
			- t.j. check nenastala mi medzi tym zmena
	- name - user-friendly meno
	- enabled
	- configuration id
	- device_id_filter - OR - pride event a chcecknem ci device_id_filter obsahuje ID device
		- ak je prazdny tak vsetko pustit
	- resource_type_filter - AND - ked mam viac musia sa vsetky matchnut
			- ak je prazdny tak vsetko pustit
	- resource_href_filer - OR - musim matchnut aspon jeden href
			- ak je prazdny tak vsetko pustit- resource_href_filter
        - jq_expression - expression pustim nad obsahom ResourceChanged eventom, mal by vratit true/false (ci hodnota existuje)
          - dalsia podmienka
          https://github.com/itchyny/gojq
        - api_access_token - TODO, zatial nechame otvorene; ked sa ide aplikovat konfiguraciu tak potrebujes token na autorizaciu
        - owner -> musi sediet s tym co je v DB
*/

func (s *Store) InsertConditions(ctx context.Context, conds ...*store.Condition) error {
	documents := make([]interface{}, 0, len(conds))
	for _, cond := range conds {
		documents = append(documents, cond)
	}
	_, err := s.Collection(conditionsCol).InsertMany(ctx, documents)
	return err
}

func (s *Store) CreateCondition(ctx context.Context, cond *pb.Condition) (*pb.Condition, error) {
	if err := store.ValidateAndNormalizeCondition(cond, false); err != nil {
		return nil, err
	}
	newCond := cond.Clone()
	if newCond.GetId() == "" {
		newCond.Id = uuid.NewString()
	}
	newCond.Timestamp = time.Now().UnixNano()
	storeCond := store.MakeFirstCondition(newCond)
	_, err := s.Collection(conditionsCol).InsertOne(ctx, storeCond)
	if err != nil {
		return nil, err
	}
	return storeCond.GetLatest()
}

func filterCondition(cond *pb.Condition) bson.M {
	filter := bson.M{
		store.IDKey:    cond.GetId(),
		store.OwnerKey: cond.GetOwner(),
	}
	if cond.GetConfigurationId() != "" {
		filter[store.ConfigurationIDKey] = cond.GetConfigurationId()
	}
	if cond.GetVersion() != 0 {
		// if is set -> it must be higher than the $latest.version
		filter[store.LatestKey+"."+store.VersionKey] = bson.M{"$lt": cond.GetVersion()}
	}
	return filter
}

func latestCondition(cond *pb.Condition) bson.M {
	ts := time.Now().UnixNano()
	latest := bson.M{
		store.VersionKey:            cond.GetVersion(),
		store.EnabledKey:            cond.GetEnabled(),
		store.TimestampKey:          ts,
		store.DeviceIDFilterKey:     cond.GetDeviceIdFilter(),
		store.ResourceTypeFilterKey: cond.GetResourceTypeFilter(),
		store.ResourceHrefFilterKey: cond.GetResourceHrefFilter(),
		store.JqExpressionFilterKey: cond.GetJqExpressionFilter(),
		store.ApiAccessTokenKey:     cond.GetApiAccessToken(),
	}
	if cond.GetName() != "" {
		latest[store.NameKey] = cond.GetName()
	}
	if cond.GetVersion() == 0 {
		// if version is not set -> set it to $latest.version + 1
		latest[store.VersionKey] = incrementLatestVersion()
	}
	return latest
}

func updateCondition(cond *pb.Condition) mongo.Pipeline {
	setVersions := appendLatestToVersions([]string{
		store.NameKey,
		store.VersionKey,
		store.EnabledKey,
		store.TimestampKey,
		store.DeviceIDFilterKey,
		store.ResourceTypeFilterKey,
		store.ResourceHrefFilterKey,
		store.JqExpressionFilterKey,
		store.ApiAccessTokenKey,
	})
	return mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			store.LatestKey: latestCondition(cond),
		}}},
		bson.D{{Key: "$set", Value: bson.M{
			store.VersionsKey: setVersions,
		}}},
	}
}

func (s *Store) UpdateCondition(ctx context.Context, cond *pb.Condition) (*pb.Condition, error) {
	if err := store.ValidateAndNormalizeCondition(cond, true); err != nil {
		return nil, err
	}

	filter := filterCondition(cond)
	update := updateCondition(cond)
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(bson.M{store.VersionsKey: false})
	result := s.Collection(conditionsCol).FindOneAndUpdate(ctx, filter, update, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	updatedCond := &store.Condition{}
	err := result.Decode(&updatedCond)
	if err != nil {
		return nil, err
	}
	return updatedCond.GetLatest()
}

// getConditionsByID returns full condition documents matched by ID
func (s *Store) getConditionsByID(ctx context.Context, owner string, ids []string, p store.ProcessConditions) error {
	cur, err := s.Collection(conditionsCol).Find(ctx, toFilterQuery(owner, toIdQuery(ids), false))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func toIdOrConfIdQuery(ids, confIds []string) bson.M {
	filter := make([]bson.M, 0, 2)
	idFilter := inArrayQuery(store.IDKey, ids)
	if len(idFilter) > 0 {
		filter = append(filter, idFilter)
	}
	confIdFilter := inArrayQuery(store.ConfigurationIDKey, confIds)
	if len(confIdFilter) > 0 {
		filter = append(filter, confIdFilter)
	}
	if len(filter) == 0 {
		return nil
	}
	if len(filter) == 1 {
		return filter[0]
	}
	return bson.M{"$or": filter}
}

// getLatestConditions returns the latest condition from document matched by condition ID or configuration ID
func (s *Store) getLatestConditions(ctx context.Context, owner string, ids, confIds []string, p store.ProcessConditions) error {
	opt := options.Find().SetProjection(bson.M{store.VersionsKey: false})
	cur, err := s.Collection(conditionsCol).Find(ctx, toFilterQuery(owner, toIdOrConfIdQuery(ids, confIds), false), opt)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

// getConditionsByAggregation returns conditions matched by ID and versions
func (s *Store) getConditionsByAggregation(ctx context.Context, owner, id string, versions []uint64, p store.ProcessConditions) error {
	cur, err := s.Collection(conditionsCol).Aggregate(ctx, getPipeline(owner, id, versions))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func normalizeSlice(s []string) []string {
	slices.Sort(s)
	return slices.Compact(s)
}

func (s *Store) GetConditions(ctx context.Context, owner string, query *pb.GetConditionsRequest, p store.Process[store.Condition]) error {
	vf := pb.PartitionIDFilter(query.GetIdFilter())
	confIdLatestFilter := normalizeSlice(query.GetConfigurationIdFilter())
	var errors *multierror.Error
	if len(vf.All) > 0 || vf.IsEmpty() && len(confIdLatestFilter) == 0 {
		err := s.getConditionsByID(ctx, owner, vf.All, p)
		errors = multierror.Append(errors, err)
	}

	if len(vf.Latest) > 0 || len(confIdLatestFilter) > 0 {
		err := s.getLatestConditions(ctx, owner, vf.Latest, query.GetConfigurationIdFilter(), p)
		errors = multierror.Append(errors, err)
	}

	for id, vf := range vf.Versions {
		err := s.getConditionsByAggregation(ctx, owner, id, vf, p)
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}

func (s *Store) DeleteConditions(ctx context.Context, owner string, query *pb.DeleteConditionsRequest) (int64, error) {
	return s.delete(ctx, conditionsCol, owner, query.GetIdFilter())
}

func toLatestDeviceIDQueryFilter(deviceID string) bson.M {
	key := store.LatestKey + "." + store.DeviceIDFilterKey
	return bson.M{"$or": bson.A{
		bson.M{key: bson.M{"$exists": false}},
		bson.M{key: deviceID},
	}}
}

func toLatestConditionsQueryFilter(owner string, queries *store.GetLatestConditionsQuery) interface{} {
	filter := make([]interface{}, 0, 4)
	if owner != "" {
		filter = append(filter, bson.D{{Key: store.OwnerKey, Value: owner}})
	}
	if queries.DeviceID != "" {
		filter = append(filter, toLatestDeviceIDQueryFilter(queries.DeviceID))
	}
	// if queries.ResourceHref != "" {
	// 	filter = append(filter, toResourceHrefQueryFilter(queries.ResourceHref))
	// }
	// if len(queries.ResourceTypeFilter) > 0 {
	// 	filter = append(filter, toResouceTypeQueryFilter(queries.ResourceTypeFilter))
	// }
	if len(filter) == 0 {
		return bson.D{}
	}
	if len(filter) == 1 {
		return filter[0]
	}
	return bson.M{"$and": filter}
}

func (s *Store) GetLatestConditions(ctx context.Context, owner string, query *store.GetLatestConditionsQuery, p store.ProcessConditions) error {
	if err := store.ValidateAndNormalizeConditionsQuery(query); err != nil {
		return err
	}
	col := s.Collection(conditionsCol)
	cur, err := col.Find(ctx, toLatestConditionsQueryFilter(owner, query))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}
