package cqldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/identity-store/persistence"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
)

// PersistenceTx prevents data race for a sequence of read and write operations.
type PersistenceTx struct {
	tx    *gocql.Session
	table string
	err   error
	ctx   context.Context
}

// NewTransaction creates a new transaction.
// A transaction must always be closed:
//
//	tx := s.persistence.NewTransaction()
//	defer tx.Close()
func (s *Store) NewTransaction(ctx context.Context) persistence.PersistenceTx {
	return &PersistenceTx{tx: s.client.Session(), table: s.Table(), err: nil, ctx: ctx}
}

func (p *PersistenceTx) retrieveDeviceByQuery(whereCondition string) (_ *persistence.AuthorizedDevice, ok bool, err error) {
	var b strings.Builder
	b.WriteString(cqldb.SelectCommand + " ")
	b.WriteString(ownerKey)
	b.WriteString(",")
	b.WriteString(deviceIDKey)
	b.WriteString(" " + cqldb.FromClause + " ")
	b.WriteString(p.table)
	b.WriteString(" " + cqldb.WhereClause + " ")
	b.WriteString(whereCondition)

	var d persistence.AuthorizedDevice
	err = p.tx.Query(b.String()).WithContext(p.ctx).Scan(&d.Owner, &d.DeviceID)
	if err != nil && errors.Is(err, gocql.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &d, true, nil
}

// Retrieve device's authorization details.
func (p *PersistenceTx) Retrieve(deviceID, owner string) (_ *persistence.AuthorizedDevice, ok bool, err error) {
	if p.err != nil {
		return nil, false, p.err
	}

	var b strings.Builder
	b.WriteString(deviceIDKey)
	b.WriteString("=")
	b.WriteString(deviceID)
	b.WriteString(" AND ")
	b.WriteString(ownerKey)
	b.WriteString("='")
	b.WriteString(owner)
	b.WriteString("'")
	b.WriteString(" ALLOW FILTERING")

	return p.retrieveDeviceByQuery(b.String())
}

// RetrieveByDevice device's authorization details.
func (p *PersistenceTx) RetrieveByDevice(deviceID string) (_ *persistence.AuthorizedDevice, ok bool, err error) {
	if p.err != nil {
		err = p.err
		return
	}

	var b strings.Builder
	b.WriteString(deviceIDKey)
	b.WriteString("=")
	b.WriteString(deviceID)

	return p.retrieveDeviceByQuery(b.String())
}

// RetrieveAll retrieves all owner's authorized devices.
func (p *PersistenceTx) RetrieveByOwner(owner string) persistence.Iterator {
	if p.err != nil {
		return &iterator{err: p.err}
	}

	var b strings.Builder
	b.WriteString(cqldb.SelectCommand + " ")
	b.WriteString(ownerKey)
	b.WriteString(",")
	b.WriteString(deviceIDKey)
	b.WriteString(" " + cqldb.FromClause + " ")
	b.WriteString(p.table)
	b.WriteString(" " + cqldb.WhereClause + " ")
	b.WriteString(ownerKey)
	b.WriteString("='")
	b.WriteString(owner)
	b.WriteString("'")

	iter := p.tx.Query(b.String()).WithContext(p.ctx).Iter()

	return &iterator{
		iter: iter,
		ctx:  p.ctx,
	}
}

type iterator struct {
	err  error
	iter *gocql.Iter
	ctx  context.Context
}

func (i *iterator) Next(s *persistence.AuthorizedDevice) bool {
	if i.err != nil {
		return false
	}
	return i.iter.Scan(&s.Owner, &s.DeviceID)
}

func (i *iterator) Err() error {
	return i.err
}

func (i *iterator) Close() {
	if i.iter != nil {
		i.err = i.iter.Close()
	}
}

func (p *PersistenceTx) Persist(d *persistence.AuthorizedDevice) error {
	if d == nil {
		return errors.New("cannot persist nil device")
	}
	if p.err != nil {
		return p.err
	}
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(p.table)
	b.WriteString(" (")
	b.WriteString(ownerKey)
	b.WriteString(",")
	b.WriteString(deviceIDKey)
	b.WriteString(") VALUES ('")
	b.WriteString(d.Owner)
	b.WriteString("',")
	b.WriteString(d.DeviceID)
	b.WriteString(")")
	b.WriteString(" IF NOT EXISTS")

	var upd persistence.AuthorizedDevice
	applied, err := p.tx.Query(b.String()).WithContext(p.ctx).ScanCAS(&upd.DeviceID, &upd.Owner)
	if err != nil {
		return err
	}
	if !applied {
		if d.Owner == upd.Owner {
			return nil
		}
		return fmt.Errorf("device %v already exists", upd.DeviceID)
	}
	return nil
}

// Delete removes the device's authorization record.
func (p *PersistenceTx) Delete(deviceID, owner string) error {
	if p.err != nil {
		return p.err
	}
	var b strings.Builder
	b.WriteString("DELETE FROM ")
	b.WriteString(p.table)
	b.WriteString(" " + cqldb.WhereClause + " ")
	b.WriteString(deviceIDKey)
	b.WriteString("=")
	b.WriteString(deviceID)
	b.WriteString(" IF ")
	b.WriteString(ownerKey)
	b.WriteString("='")
	b.WriteString(owner)
	b.WriteString("'")
	applied, err := p.tx.Query(b.String()).WithContext(p.ctx).ScanCAS(nil)
	if err != nil {
		return err
	}
	if !applied {
		return fmt.Errorf("device %v is not found", deviceID)
	}
	return nil
}

// Close must always be called (use defer immediately after NewTransaction).
func (p *PersistenceTx) Close() {
	// do nothing - transaction is not supported by cql, but the implementation does not require it
}
