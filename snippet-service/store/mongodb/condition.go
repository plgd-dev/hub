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

func (s *Store) getConditionsByFind(ctx context.Context, owner string, idfAlls []string, p store.ProcessConditions) error {
	cur, err := s.Collection(conditionsCol).Find(ctx, toIdFilterQuery(owner, idfAlls))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) getConditionsByAggregation(ctx context.Context, owner, id string, vf pb.VersionFilter, p store.ProcessConditions) error {
	cur, err := s.Collection(conditionsCol).Aggregate(ctx, getPipeline(owner, id, vf))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) GetConditions(ctx context.Context, owner string, query *pb.GetConditionsRequest, p store.ProcessIterator[store.Condition]) error {
	idVersionAll, idVersions := pb.PartitionIDFilter(query.GetIdFilter())
	var errors *multierror.Error
	if len(idVersionAll) > 0 || len(idVersions) == 0 {
		err := s.getConditionsByFind(ctx, owner, idVersionAll, p)
		errors = multierror.Append(errors, err)
	}
	if len(idVersions) > 0 {
		for id, vf := range idVersions {
			err := s.getConditionsByAggregation(ctx, owner, id, vf, p)
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

func (s *Store) DeleteConditions(ctx context.Context, owner string, query *pb.DeleteConditionsRequest) (int64, error) {
	return s.delete(ctx, conditionsCol, owner, query.GetIdFilter())
}
