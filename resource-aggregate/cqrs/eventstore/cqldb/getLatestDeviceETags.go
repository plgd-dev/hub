package cqldb

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
)

type etagTimestamp struct {
	timestamp int64
	etag      []byte
}

type etagTimestamps []etagTimestamp

func (a etagTimestamps) sort() {
	sort.Slice(a, func(i, j int) bool {
		// sort by timestamp descending
		return a[i].timestamp > a[j].timestamp
	})
}

func (a etagTimestamps) toETags(limit int) [][]byte {
	a.sort()
	if limit > 0 && len(a) > limit {
		a = a[:limit]
	}
	etags := make([][]byte, 0, len(a))
	for _, v := range a {
		etags = append(etags, v.etag)
	}
	return etags
}

// Get latest ETags for device resources from event store for batch observing
func (s *EventStore) GetLatestDeviceETags(ctx context.Context, deviceID string, limit uint32) ([][]byte, error) {
	if deviceID == "" {
		return nil, errors.New("deviceID is invalid")
	}
	var q strings.Builder
	q.WriteString(cqldb.SelectCommand)
	q.WriteString(" " + etagKey + "," + etagTimestampKey)
	q.WriteString(" " + cqldb.FromClause)
	q.WriteString(" " + s.Table())
	q.WriteString(" " + cqldb.WhereClause)
	q.WriteString(" " + deviceIDKey + "=" + deviceID)

	iter := s.client.Session().Query(q.String()).WithContext(ctx).Iter()
	if iter == nil {
		return nil, errors.New("cannot create iterator")
	}

	ret := make(etagTimestamps, 0, iter.NumRows())
	for {
		var v etagTimestamp
		if !iter.Scan(&v.etag, &v.timestamp) {
			break
		}
		if v.etag == nil || v.timestamp == 0 {
			continue
		}
		ret = append(ret, v)
	}
	err := iter.Close()
	if err == nil {
		return ret.toETags(int(limit)), nil
	}
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, nil
	}
	return nil, err
}
