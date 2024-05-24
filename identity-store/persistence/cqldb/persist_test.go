package cqldb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/identity-store/persistence"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func newTestPersistence(t *testing.T) *cqldb.Store {
	ctx := context.Background()
	cfg := config.MakeCqlDBConfig()

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
	}()
	p, err := cqldb.New(ctx, &cqldb.Config{
		Embedded: cfg,
		Table:    "testDeviceOwnership",
	}, fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)

	p.AddCloseFunc(func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	})
	return p
}

func TestPersistenceTxRetrieve(t *testing.T) {
	ctx := context.Background()
	p := newTestPersistence(t)
	defer func() {
		err := p.Clear(ctx)
		require.NoError(t, err)
		err = p.Close(ctx)
		require.NoError(t, err)
	}()

	tx := p.NewTransaction(ctx)
	defer tx.Close()

	const owner = "test-owner"

	device := &persistence.AuthorizedDevice{
		DeviceID: test.GenerateDeviceIDbyIdx(1),
		Owner:    owner,
	}

	err := tx.Persist(device)
	require.NoError(t, err)

	tests := []struct {
		name     string
		deviceID string
		owner    string
		want     *persistence.AuthorizedDevice
		wantOk   bool
		wantErr  bool
	}{
		{
			name:     "valid",
			deviceID: device.DeviceID,
			owner:    owner,
			want:     device,
			wantOk:   true,
			wantErr:  false,
		},
		{
			name:     "invalid owner",
			deviceID: device.DeviceID,
			owner:    "invalid-owner",
			want:     nil,
			wantOk:   false,
			wantErr:  false,
		},
		{
			name:     "invalid deviceID",
			deviceID: uuid.Nil.String(),
			owner:    owner,
			want:     nil,
			wantOk:   false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := tx.Retrieve(tt.deviceID, tt.owner)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestPersistenceTxRetrieveByDevice(t *testing.T) {
	ctx := context.Background()
	p := newTestPersistence(t)
	defer func() {
		err := p.Clear(ctx)
		require.NoError(t, err)
		err = p.Close(ctx)
		require.NoError(t, err)
	}()

	tx := p.NewTransaction(ctx)
	defer tx.Close()

	const owner = "test-owner"

	device := &persistence.AuthorizedDevice{
		DeviceID: test.GenerateDeviceIDbyIdx(1),
		Owner:    owner,
	}

	err := tx.Persist(device)
	require.NoError(t, err)

	tests := []struct {
		name     string
		deviceID string
		want     *persistence.AuthorizedDevice
		wantOk   bool
		wantErr  bool
	}{
		{
			name:     "valid",
			deviceID: device.DeviceID,
			want:     device,
			wantOk:   true,
			wantErr:  false,
		},
		{
			name:     "invalid deviceID",
			deviceID: uuid.Nil.String(),
			want:     nil,
			wantOk:   false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := tx.RetrieveByDevice(tt.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestPersistenceTxRetrieveByOwner(t *testing.T) {
	ctx := context.Background()
	p := newTestPersistence(t)
	defer func() {
		err := p.Clear(ctx)
		require.NoError(t, err)
		err = p.Close(ctx)
		require.NoError(t, err)
	}()

	tx := p.NewTransaction(ctx)
	defer tx.Close()

	const owner = "test-owner"

	device1 := &persistence.AuthorizedDevice{
		DeviceID: test.GenerateDeviceIDbyIdx(1),
		Owner:    owner,
	}

	device2 := &persistence.AuthorizedDevice{
		DeviceID: test.GenerateDeviceIDbyIdx(2),
		Owner:    owner,
	}

	err := tx.Persist(device1)
	require.NoError(t, err)

	err = tx.Persist(device2)
	require.NoError(t, err)

	tests := []struct {
		name    string
		owner   string
		devices []*persistence.AuthorizedDevice
	}{
		{
			name:    "valid",
			owner:   owner,
			devices: []*persistence.AuthorizedDevice{device1, device2},
		},
		{
			name:    "invalid owner",
			owner:   "invalid-owner",
			devices: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := tx.RetrieveByOwner(tt.owner)
			defer iter.Close()
			func() {
				errC := iter.Err()
				require.NoError(t, errC)
			}()

			var retrievedDevices []*persistence.AuthorizedDevice
			for {
				var device persistence.AuthorizedDevice
				if !iter.Next(&device) {
					break
				}
				require.NoError(t, err)
				retrievedDevices = append(retrievedDevices, &device)
			}

			require.Equal(t, tt.devices, retrievedDevices)
		})
	}
}

func TestPersistenceTxPersist(t *testing.T) {
	ctx := context.Background()
	p := newTestPersistence(t)
	defer func() {
		err := p.Clear(ctx)
		require.NoError(t, err)
		err = p.Close(ctx)
		require.NoError(t, err)
	}()

	tx := p.NewTransaction(ctx)
	defer tx.Close()

	const owner = "test-owner"

	device := &persistence.AuthorizedDevice{
		DeviceID: test.GenerateDeviceIDbyIdx(1),
		Owner:    owner,
	}

	tests := []struct {
		name                string
		device              *persistence.AuthorizedDevice
		actionBeforePersist func(t *testing.T, tx persistence.PersistenceTx)
		wantErr             bool
	}{
		{
			name:    "valid",
			device:  device,
			wantErr: false,
		},
		{
			name: "duplicity",
			actionBeforePersist: func(t *testing.T, tx persistence.PersistenceTx) {
				err := tx.Persist(device)
				require.NoError(t, err)
			},
			device:  device,
			wantErr: false,
		},
		{
			name: "device with another owner",
			actionBeforePersist: func(t *testing.T, tx persistence.PersistenceTx) {
				err := tx.Persist(device)
				require.NoError(t, err)
			},
			device: &persistence.AuthorizedDevice{
				DeviceID: device.DeviceID,
				Owner:    "another-owner",
			},
			wantErr: true,
		},
		{
			name: "immediately after delete the device",
			actionBeforePersist: func(t *testing.T, tx persistence.PersistenceTx) {
				err := tx.Persist(device)
				require.NoError(t, err)
				err = tx.Delete(device.DeviceID, device.Owner)
				require.NoError(t, err)
			},
			device: &persistence.AuthorizedDevice{
				DeviceID: device.DeviceID,
				Owner:    "another-owner",
			},
			wantErr: false,
		},
		{
			name:    "nil device",
			device:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ClearTable(ctx)
			require.NoError(t, err)
			if tt.actionBeforePersist != nil {
				tt.actionBeforePersist(t, tx)
			}
			err = tx.Persist(tt.device)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
