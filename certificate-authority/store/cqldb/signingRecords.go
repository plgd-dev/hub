package cqldb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
)

var ErrCannotRemoveSigningRecord = errors.New("cannot remove signing record")

func deviceIDToValue(deviceID string) string {
	if deviceID == "" {
		return "null"
	}
	return deviceID
}

func setTTLbyDeviceID(deviceID string, validUntil int64) string {
	if deviceID != "" {
		// records for devices should be removed by DeleteSigningRecord on-demand
		return " USING TTL 0"
	}
	validUntil /= int64(time.Second)
	return fmt.Sprintf(" USING TTL %v", validUntil-time.Now().Unix())
}

func (s *Store) getInsertQuery(signingRecord *store.SigningRecord, upsert bool) (string, error) {
	if err := signingRecord.Validate(); err != nil {
		return "", err
	}
	data, err := utils.Marshal(signingRecord)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(s.Table())
	b.WriteString(" (")
	b.WriteString(idKey)
	b.WriteString(",")
	b.WriteString(ownerKey)
	b.WriteString(",")
	b.WriteString(deviceIDKey)
	b.WriteString(",")
	b.WriteString(commonNameKey)
	b.WriteString(",")
	b.WriteString(dataKey)
	b.WriteString(") VALUES (")
	b.WriteString(signingRecord.GetId())
	b.WriteString(",'")
	b.WriteString(signingRecord.GetOwner())
	b.WriteString("',")
	b.WriteString(deviceIDToValue(signingRecord.GetDeviceId()))
	b.WriteString(",'")
	b.WriteString(signingRecord.GetCommonName())
	b.WriteString("',")
	cqldb.EncodeToBlob(data, &b)
	b.WriteString(")")
	if !upsert {
		b.WriteString(" IF NOT EXISTS")
	}
	b.WriteString(setTTLbyDeviceID(signingRecord.GetDeviceId(), signingRecord.GetCredential().GetValidUntilDate()))
	return b.String(), nil
}

func (s *Store) CreateSigningRecord(ctx context.Context, signingRecord *store.SigningRecord) error {
	insertQuery, err := s.getInsertQuery(signingRecord, false)
	if err != nil {
		return err
	}
	applied, err := s.Session().Query(insertQuery).WithContext(ctx).ScanCAS(nil, nil, nil, nil, nil)
	if err != nil {
		return err
	}
	if !applied {
		return errors.New("cannot insert signing record: already exists")
	}
	return nil
}

func (s *Store) UpdateSigningRecord(ctx context.Context, signingRecord *store.SigningRecord) error {
	// To set TTL for whole row, we need to reinsert(update with upsert) the row,
	// because Cassandra does not allow to update TTL to primary key columns
	// More info https://opensource.docs.scylladb.com/stable/cql/time-to-live.html
	insertQuery, err := s.getInsertQuery(signingRecord, true)
	if err != nil {
		return err
	}

	return s.Session().Query(insertQuery).WithContext(ctx).Exec()
}

func stringsToCQLSet(filter []string, str bool) string {
	var b strings.Builder
	b.WriteString("(")
	for _, f := range filter {
		if f == "" {
			continue
		}
		if b.Len() > 1 {
			b.WriteString(",")
		}
		if str {
			b.WriteString("'")
		}
		b.WriteString(f)
		if str {
			b.WriteString("'")
		}
	}
	b.WriteString(")")
	return b.String()
}

func toCommonNameQueryFilter(owner string, commonNames []string) string {
	return fmt.Sprintf("%v='%v' AND %v IN %v", ownerKey, owner, commonNameKey, stringsToCQLSet(commonNames, true))
}

func toDeviceIDQueryFilter(owner string, deviceIDs []string) string {
	return fmt.Sprintf("%v='%v' AND %v IN %v", ownerKey, owner, deviceIDKey, stringsToCQLSet(deviceIDs, false))
}

func toIDQueryFilter(owner string, ids []string) string {
	return fmt.Sprintf("%v IN %v AND %v='%v'", idKey, stringsToCQLSet(ids, false), ownerKey, owner)
}

func toSigningRecordsQueryFilter(owner string, queries *store.SigningRecordsQuery, allowFiltering bool) []string {
	or := make([]string, 0, 16)
	if len(queries.GetIdFilter()) > 0 {
		or = append(or, toIDQueryFilter(owner, queries.GetIdFilter()))
	}
	if len(queries.GetCommonNameFilter()) > 0 {
		q := toCommonNameQueryFilter(owner, queries.GetCommonNameFilter())
		or = append(or, q)
	}
	if len(queries.GetDeviceIdFilter()) > 0 {
		q := toDeviceIDQueryFilter(owner, queries.GetDeviceIdFilter())
		if allowFiltering {
			q += " ALLOW FILTERING"
		}
		or = append(or, q)
	}
	if len(or) == 0 {
		return []string{ownerKey + "='" + owner + "'"}
	}
	return or
}

type primaryKeyValues struct {
	id         string
	owner      string
	commonName string
}

type primaryKeysValues []primaryKeyValues

func (p primaryKeysValues) sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].id < p[j].id
	})
}

func (p primaryKeysValues) unique() primaryKeysValues {
	p.sort()
	for i := 1; i < len(p); {
		if p[i-1].id == p[i].id {
			p = append(p[:i], p[i+1:]...)
		} else {
			i++
		}
	}
	return p
}

func (p primaryKeysValues) toCqlFilterWithoutOwner() string {
	var b strings.Builder
	b.WriteString(idKey)
	b.WriteString(" IN ")
	b.WriteString("(")
	for i, pk := range p {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(pk.id)
	}
	b.WriteString(")")
	b.WriteString(" AND ")
	b.WriteString(commonNameKey)
	b.WriteString(" IN ")
	b.WriteString("(")
	for i, pk := range p {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("'")
		b.WriteString(pk.commonName)
		b.WriteString("'")
	}
	b.WriteString(")")
	return b.String()
}

func readPrimaryKeys(iter *gocql.Iter) ([]primaryKeyValues, error) {
	var pk primaryKeyValues
	pks := make([]primaryKeyValues, 0, 32)
	for iter.Scan(&pk.id, &pk.owner, &pk.commonName) {
		pks = append(pks, pk)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return pks, nil
}

func (s *Store) readPrimaryKeys(ctx context.Context, where string) (primaryKeysValues, error) {
	var b strings.Builder
	b.WriteString(cqldb.SelectCommand + " ")
	b.WriteString(idKey)
	b.WriteString(",")
	b.WriteString(ownerKey)
	b.WriteString(",")
	b.WriteString(commonNameKey)
	b.WriteString(" " + cqldb.FromClause + " ")
	b.WriteString(s.Table())
	b.WriteString(" " + cqldb.WhereClause + " ")
	b.WriteString(where)
	iter := s.Session().Query(b.String()).WithContext(ctx).Iter()
	defer iter.Close()
	return readPrimaryKeys(iter)
}

func (s *Store) deviceIDFilterToPrimaryKeys(ctx context.Context, owner string, deviceIDFilter []string) (primaryKeysValues, error) {
	if len(deviceIDFilter) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString(toDeviceIDQueryFilter(owner, deviceIDFilter))
	b.WriteString(" ALLOW FILTERING")

	return s.readPrimaryKeys(ctx, b.String())
}

func (s *Store) ownerFilterToPrimaryKeys(ctx context.Context, owner string) (primaryKeysValues, error) {
	if owner == "" {
		return nil, errors.New("invalid owner")
	}

	var b strings.Builder
	b.WriteString(ownerKey)
	b.WriteString("='")
	b.WriteString(owner)
	b.WriteString("'")

	return s.readPrimaryKeys(ctx, b.String())
}

func (s *Store) idFilterToPrimaryKeys(ctx context.Context, owner string, idFilter []string) (primaryKeysValues, error) {
	if len(idFilter) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString(toIDQueryFilter(owner, idFilter))

	return s.readPrimaryKeys(ctx, b.String())
}

func (s *Store) queryToPrimaryKeys(ctx context.Context, owner string, query *store.DeleteSigningRecordsQuery) (primaryKeysValues, error) {
	var pks primaryKeysValues
	if len(query.GetIdFilter()) > 0 {
		tmpPks, err := s.idFilterToPrimaryKeys(ctx, owner, query.GetIdFilter())
		if err != nil {
			return nil, err
		}
		pks = tmpPks
	}
	if len(query.GetDeviceIdFilter()) > 0 {
		tmpPks, err := s.deviceIDFilterToPrimaryKeys(ctx, owner, query.GetDeviceIdFilter())
		if err != nil {
			return nil, err
		}
		pks = append(pks, tmpPks...)
	}
	if len(query.GetDeviceIdFilter()) == 0 && len(query.GetIdFilter()) == 0 {
		tmpPks, err := s.ownerFilterToPrimaryKeys(ctx, owner)
		if err != nil {
			return nil, err
		}
		pks = append(pks, tmpPks...)
	}
	pks = pks.unique()
	return pks, nil
}

func (s *Store) DeleteSigningRecords(ctx context.Context, owner string, query *store.DeleteSigningRecordsQuery) (int64, error) {
	pks, err := s.queryToPrimaryKeys(ctx, owner, query)
	if err != nil {
		return 0, err
	}
	if len(pks) == 0 {
		return 0, nil
	}

	var b strings.Builder
	b.WriteString("DELETE FROM ")
	b.WriteString(s.Table())
	b.WriteString(" " + cqldb.WhereClause + " ")
	b.WriteString(ownerKey)
	b.WriteString("='")
	b.WriteString(owner)
	b.WriteString("'")
	b.WriteString(" AND ")
	b.WriteString(pks.toCqlFilterWithoutOwner())
	b.WriteString(" IF EXISTS")

	var count int64
	applied, err := s.Session().Query(b.String()).WithContext(ctx).ScanCAS(nil, nil, nil, nil, nil)
	if err == nil {
		if applied {
			count++
		}
	} else if !errors.Is(err, gocql.ErrNotFound) {
		return 0, err
	}
	return count, nil
}

func (s *Store) DeleteNonDeviceExpiredRecords(_ context.Context, _ time.Time) (int64, error) {
	// Cassandra deletes automatically by setting expiration time to the record
	return 0, store.ErrNotSupported
}

func (s *Store) LoadSigningRecords(ctx context.Context, owner string, query *store.SigningRecordsQuery, p store.Process[store.SigningRecord]) error {
	i := SigningRecordsIterator{
		ctx:      ctx,
		s:        s,
		queries:  toSigningRecordsQueryFilter(owner, query, true),
		provided: make(map[string]struct{}, 32),
	}
	var err error
	for {
		var stored store.SigningRecord
		if !i.Next(ctx, &stored) {
			err = i.Err()
			break
		}
		err = p(&stored)
		if err != nil {
			break
		}
	}
	errClose := i.close()
	if err == nil {
		return errClose
	}
	return err
}

func (s *Store) RevokeSigningRecords(ctx context.Context, ownerID string, query *store.RevokeSigningRecordsQuery) (int64, error) {
	// TODO: revocation list not yet supported by cqldb, so for now we just delete the records
	return s.DeleteSigningRecords(ctx, ownerID, &store.DeleteSigningRecordsQuery{
		IdFilter:       query.GetIdFilter(),
		DeviceIdFilter: query.GetDeviceIdFilter(),
	})
}

type SigningRecordsIterator struct {
	ctx     context.Context
	queries []string
	s       *Store

	queriesIdx int
	iter       *gocql.Iter
	err        error
	provided   map[string]struct{}
}

func (i *SigningRecordsIterator) nextQuery() bool {
	if i.queriesIdx >= len(i.queries) {
		return false
	}
	var b strings.Builder
	b.WriteString(cqldb.SelectCommand + " ")
	b.WriteString(dataKey)
	b.WriteString(" " + cqldb.FromClause + " ")
	b.WriteString(i.s.Table())
	b.WriteString(" " + cqldb.WhereClause + " ")
	b.WriteString(i.queries[i.queriesIdx])
	i.iter = i.s.Session().Query(b.String()).WithContext(i.ctx).Iter()
	i.queriesIdx++
	return true
}

func (i *SigningRecordsIterator) close() error {
	if i.iter == nil {
		return nil
	}
	iter := i.iter
	i.iter = nil
	return iter.Close()
}

func (i *SigningRecordsIterator) Next(_ context.Context, s *store.SigningRecord) bool {
	for i.next(s) {
		if _, ok := i.provided[s.GetId()]; !ok {
			// filter duplicated records
			i.provided[s.GetId()] = struct{}{}
			return true
		}
	}
	return false
}

func (i *SigningRecordsIterator) next(s *store.SigningRecord) bool {
	if i.Err() != nil {
		return false
	}
	if i.iter == nil {
		if !i.nextQuery() {
			return false
		}
	}
	var data []byte
	if !i.iter.Scan(&data) {
		err := i.close()
		if err != nil {
			i.err = err
			return false
		}
		if !i.nextQuery() {
			return false
		}
		if !i.iter.Scan(&data) {
			return false
		}
	}
	err := utils.Unmarshal(data, s)
	if err != nil {
		_ = i.close()
		i.err = err
		return false
	}

	return true
}

func (i *SigningRecordsIterator) Err() error {
	return i.err
}
