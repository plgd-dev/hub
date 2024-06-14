package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *Store) InsertAppliedConfigurations(ctx context.Context, confs ...*store.AppliedDeviceConfiguration) error {
	documents := make([]interface{}, 0, len(confs))
	for _, conf := range confs {
		documents = append(documents, conf)
	}
	_, err := s.Collection(appliedConfigurationsCol).InsertMany(ctx, documents)
	return err
}

func (s *Store) CreateAppliedDeviceConfiguration(ctx context.Context, adc *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	err := store.ValidateAppliedConfiguration(adc, false)
	if err != nil {
		return nil, err
	}
	newAdc := adc.Clone()
	if newAdc.GetId() == "" {
		newAdc.Id = uuid.NewString()
	}
	newAdc.Timestamp = time.Now().UnixNano()
	_, err = s.Collection(appliedConfigurationsCol).InsertOne(ctx, newAdc)
	if err != nil {
		return nil, err
	}
	return newAdc, nil
}

func toAppliedDeviceConfigurationsQuery(owner string, query *pb.GetAppliedDeviceConfigurationsRequest) interface{} {
	filters := make([]interface{}, 0, 3)
	if owner != "" {
		filters = append(filters, bson.D{{Key: store.OwnerKey, Value: owner}})
	}
	idFilter := inArrayQuery(store.IDKey, strings.Unique(query.GetIdFilter()))
	if idFilter != nil {
		filters = append(filters, idFilter)
	}
	confIdFilter := inArrayQuery(store.ConfigurationRelationIDKey, strings.Unique(query.GetConfigurationIdFilter()))
	if confIdFilter != nil {
		filters = append(filters, confIdFilter)
	}
	if len(filters) == 0 {
		return bson.D{}
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return bson.M{"$and": filters}
}

func (s *Store) GetAppliedDeviceConfigurations(ctx context.Context, owner string, query *pb.GetAppliedDeviceConfigurationsRequest, p store.ProccessAppliedDeviceConfigurations) error {
	cur, err := s.Collection(appliedConfigurationsCol).Find(ctx, toAppliedDeviceConfigurationsQuery(owner, query))
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, p)
}
