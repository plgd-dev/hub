package cqldb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	pkgStrings "github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *EventStore) LoadDeviceMetadataByServiceIDs(ctx context.Context, serviceIDs []string, limit int64) ([]eventstore.DeviceDocumentMetadata, error) {
	if len(serviceIDs) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid serviceIDs")
	}
	serviceIDs = pkgStrings.Unique(serviceIDs)
	q := cqldb.SelectCommand + " " + deviceIDKey + "," + serviceIDKey + " " + cqldb.FromClause + " " + s.Table() + " " + cqldb.WhereClause + " " + serviceIDKey + " in (" + strings.Join(serviceIDs, ",") + ") LIMIT " + strconv.FormatInt(limit, 10) + ";"
	iter := s.Session().Query(q).WithContext(ctx).Iter()
	if iter == nil {
		return nil, errors.New("cannot create iterator")
	}
	ret := make([]eventstore.DeviceDocumentMetadata, 0, iter.NumRows())
	values := make(map[string]interface{}, 2)
	for iter.MapScan(values) {
		deviceID, ok := values[deviceIDKey].(gocql.UUID)
		if !ok {
			_ = iter.Close()
			return nil, fmt.Errorf(errFmtDataIsNotUUIDType, deviceIDKey, values[deviceIDKey])
		}
		serviceID, ok := values[serviceIDKey].(gocql.UUID)
		if !ok {
			_ = iter.Close()
			return nil, fmt.Errorf(errFmtDataIsNotUUIDType, serviceIDKey, values[serviceIDKey])
		}
		ret = append(ret, eventstore.DeviceDocumentMetadata{
			DeviceID:  deviceID.String(),
			ServiceID: serviceID.String(),
		})
		delete(values, deviceIDKey)
		delete(values, serviceIDKey)
	}
	err := iter.Close()
	if err == nil {
		return ret, nil
	}
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, nil
	}
	return nil, err
}
