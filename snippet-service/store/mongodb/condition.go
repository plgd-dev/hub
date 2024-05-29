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

func toCondition(c *store.Condition) *pb.Condition {
	cond := &pb.Condition{
		Id:              c.Id,
		Name:            c.Name,
		Enabled:         c.Enabled,
		Owner:           c.Owner,
		ConfigurationId: c.ConfigurationId,
		ApiAccessToken:  c.ApiAccessToken,
		Timestamp:       c.Timestamp,
	}
	if len(c.Versions) > 0 {
		cond.Version = c.Versions[0].Version
		cond.DeviceIdFilter = c.Versions[0].DeviceIdFilter
		cond.ResourceTypeFilter = c.Versions[0].ResourceTypeFilter
		cond.ResourceHrefFilter = c.Versions[0].ResourceHrefFilter
		cond.JqExpressionFilter = c.Versions[0].JqExpressionFilter
	}
	return cond
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
	storeCond := store.MakeCondition(newCond)
	_, err := s.Collection(conditionsCol).InsertOne(ctx, storeCond)
	if err != nil {
		return nil, err
	}
	return toCondition(&storeCond), nil
}

func (s *Store) UpdateCondition(ctx context.Context, cond *pb.Condition) (*pb.Condition, error) {
	if err := store.ValidateAndNormalizeCondition(cond, true); err != nil {
		return nil, err
	}

	filter := bson.M{
		store.IDKey:    cond.GetId(),
		store.OwnerKey: cond.GetOwner(),
		store.VersionsKey + "." + store.VersionKey: bson.M{"$ne": cond.GetVersion()},
	}
	if cond.GetConfigurationId() != "" {
		filter[store.ConfigurationIDKey] = cond.GetConfigurationId()
	}

	ts := time.Now().UnixNano()
	set := bson.M{
		store.EnabledKey:        cond.GetEnabled(),
		store.TimestampKey:      ts,
		store.ApiAccessTokenKey: cond.GetApiAccessToken(),
	}
	if cond.GetName() != "" {
		set[store.NameKey] = cond.GetName()
	}
	update := bson.M{
		"$push": bson.M{
			"versions": store.ConditionVersion{
				Version:            cond.GetVersion(),
				DeviceIdFilter:     cond.GetDeviceIdFilter(),
				ResourceTypeFilter: cond.GetResourceTypeFilter(),
				ResourceHrefFilter: cond.GetResourceHrefFilter(),
				JqExpressionFilter: cond.GetJqExpressionFilter(),
			},
		},
		"$set": set,
	}

	res, err := s.Collection(conditionsCol).UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, store.ErrNotFound
	}
	cond.Timestamp = ts
	return cond, nil
}

// func toDeviceIDQueryFilter(deviceID string) bson.M {
// 	return bson.M{"$or": []bson.D{
// 		{{Key: store.DeviceIDFilterKey, Value: bson.M{"$exists": false}}},
// 		{{Key: store.DeviceIDFilterKey, Value: deviceID}},
// 	}}
// }

// func toResourceHrefQueryFilter(resourceHref string) bson.M {
// 	return bson.M{"$or": []bson.D{
// 		{{Key: store.ResourceHrefFilterKey, Value: bson.M{"$exists": false}}},
// 		{{Key: store.ResourceHrefFilterKey, Value: resourceHref}},
// 	}}
// }

// func toResouceTypeQueryFilter(resourceTypeFilter []string) bson.M {
// 	slices.Sort(resourceTypeFilter)
// 	return bson.M{"$or": []bson.D{
// 		{{Key: store.ResourceTypeFilterKey, Value: bson.M{"$exists": false}}},
// 		{{Key: store.ResourceTypeFilterKey, Value: resourceTypeFilter}},
// 	}}
// }

// func toConditionsQueryFilter(owner string, queries *store.ConditionsQuery) interface{} {
// 	filter := make([]interface{}, 0, 4)
// 	if owner != "" {
// 		filter = append(filter, bson.D{{Key: store.OwnerKey, Value: owner}})
// 	}
// 	if queries.DeviceID != "" {
// 		filter = append(filter, toDeviceIDQueryFilter(queries.DeviceID))
// 	}
// 	if queries.ResourceHref != "" {
// 		filter = append(filter, toResourceHrefQueryFilter(queries.ResourceHref))
// 	}
// 	if len(queries.ResourceTypeFilter) > 0 {
// 		filter = append(filter, toResouceTypeQueryFilter(queries.ResourceTypeFilter))
// 	}
// 	if len(filter) == 0 {
// 		return bson.D{}
// 	}
// 	if len(filter) == 1 {
// 		return filter[0]
// 	}
// 	return bson.M{"$and": filter}
// }

// func (s *Store) LoadConditions(ctx context.Context, owner string, query *store.ConditionsQuery, h store.LoadConditionsFunc) error {
// 	col := s.Collection(conditionsCol)
// 	cr, err := col.Find(ctx, toConditionsQueryFilter(owner, query))
// 	if errors.Is(err, mongo.ErrNilDocument) {
// 		return nil
// 	}
// 	if h == nil || err != nil {
// 		return err
// 	}
// 	var errors *multierror.Error
// 	i := store.MongoIterator[store.Condition]{
// 		Cursor: cr,
// 	}
// 	err = h(ctx, &i)
// 	errors = multierror.Append(errors, err)
// 	errClose := cr.Close(ctx)
// 	errors = multierror.Append(errors, errClose)
// 	return errors.ErrorOrNil()
// }

func (s *Store) getConditionsByFind(ctx context.Context, owner string, idfAlls []string, p store.ProcessConditions) error {
	cur, err := s.Collection(conditionsCol).Find(ctx, toIdFilterQuery(owner, idfAlls))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}

func (s *Store) getConditionsByAggregation(ctx context.Context, owner, id string, vf pb.VersionFilter, p store.ProcessConditions) error {
	pl := mongo.Pipeline{bson.D{{Key: "$match", Value: addMatchCondition(owner, id)}}}
	pl = getVersionsPipeline(pl, vf, false)
	cur, err := s.Collection(conditionsCol).Aggregate(ctx, pl)
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
