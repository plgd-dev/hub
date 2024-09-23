package mongodb_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

func constDate() time.Time {
	return time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
}

func constDate1() time.Time {
	return time.Date(2006, time.January, 5, 15, 4, 5, 0, time.UTC)
}

func TestStoreUpdateSigningRecord(t *testing.T) {
	type args struct {
		sub *store.SigningRecord
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				sub: &store.SigningRecord{
					// Id: "deviceIDnotFound",
					Owner:      "owner",
					CommonName: "commonName",
					Credential: &pb.CredentialStatus{
						CertificatePem: "certificate",
						Date:           constDate().UnixNano(),
						ValidUntilDate: constDate().UnixNano(),
						Serial:         big.NewInt(42).String(),
						IssuerId:       "42424242-4242-4242-4242-424242424242",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "first update",
			args: args{
				sub: &store.SigningRecord{
					Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
					Owner:        "owner",
					CommonName:   "commonName",
					PublicKey:    "publicKey",
					CreationDate: constDate().UnixNano(),
					Credential: &pb.CredentialStatus{
						CertificatePem: "certificate",
						Date:           constDate().UnixNano(),
						ValidUntilDate: constDate().UnixNano(),
						Serial:         big.NewInt(42).String(),
						IssuerId:       "42424242-4242-4242-4242-424242424242",
					},
				},
			},
		},
		{
			name: "second update",
			args: args{
				sub: &store.SigningRecord{
					Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
					Owner:        "owner",
					CommonName:   "commonName",
					PublicKey:    "publicKey",
					CreationDate: constDate().UnixNano(),
					Credential: &pb.CredentialStatus{
						CertificatePem: "certificate1",
						Date:           constDate1().UnixNano(),
						ValidUntilDate: constDate1().UnixNano(),
						Serial:         big.NewInt(42).String(),
						IssuerId:       "42424242-4242-4242-4242-424242424242",
					},
				},
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateSigningRecord(ctx, tt.args.sub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStoreDeleteSigningRecords(t *testing.T) {
	records := []struct {
		id       string
		deviceID string
	}{
		{
			id:       "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
			deviceID: hubTest.GenerateDeviceIDbyIdx(1),
		},
		{
			id:       "9d017fad-2961-4fcc-94a9-1e1291a88ffd",
			deviceID: hubTest.GenerateDeviceIDbyIdx(2),
		},
		{
			id:       "9d017fad-2961-4fcc-94a9-1e1291a88ffe",
			deviceID: hubTest.GenerateDeviceIDbyIdx(3),
		},
		{
			id:       "9d017fad-2961-4fcc-94a9-1e1291a88fff",
			deviceID: hubTest.GenerateDeviceIDbyIdx(4),
		},
	}
	const owner = "owner"
	type args struct {
		owner string
		query *store.DeleteSigningRecordsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    int64
	}{
		{
			name: "invalid Id",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{
					IdFilter: []string{"invalid"},
				},
			},
			want: 0,
		},
		{
			name: "invalid owner",
			args: args{
				owner: "owner1",
				query: &store.DeleteSigningRecordsQuery{
					IdFilter: []string{records[0].id},
				},
			},
			want: 0,
		},
		{
			name: "valid - by deviceID",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{
					DeviceIdFilter: []string{records[0].deviceID},
				},
			},
			want: 1,
		},
		{
			name: "valid - by id",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{
					IdFilter: []string{records[1].id},
				},
			},
			want: 1,
		},
		{
			name: "multiple ids",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{
					IdFilter: []string{records[2].id, records[3].id},
				},
			},
			want: 2,
		},
		{
			name: "valid - empty",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{},
			},
			want: 0,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, r := range records {
		err := s.CreateSigningRecord(ctx, &store.SigningRecord{
			Id:           r.id,
			Owner:        owner,
			CommonName:   "commonName",
			PublicKey:    "publicKey",
			DeviceId:     r.deviceID,
			CreationDate: constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
				Serial:         big.NewInt(42).String(),
				IssuerId:       "42424242-4242-4242-4242-424242424242",
			},
		})
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := s.DeleteSigningRecords(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, n)
			}
		})
	}
}

func TestStoreDeleteExpiredRecords(t *testing.T) {
	type args struct {
		now time.Time
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "nothing to delete",
			args: args{
				now: constDate().Add(-time.Hour),
			},
			want: 0,
		},
		{
			name: "delete one",
			args: args{
				now: constDate().Add(time.Hour),
			},
			want: 1,
		},
		{
			name: "delete but db is empty",
			args: args{
				now: constDate().Add(time.Hour),
			},
			want: 0,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
		Owner:        "owner",
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		CreationDate: constDate().UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           constDate().UnixNano(),
			ValidUntilDate: constDate().UnixNano(),
			Serial:         big.NewInt(42).String(),
			IssuerId:       "42424242-4242-4242-4242-424242424242",
		},
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.DeleteNonDeviceExpiredRecords(ctx, tt.args.now)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

type testSigningRecordHandler struct {
	lcs pb.SigningRecords
}

func (h *testSigningRecordHandler) process(sr *store.SigningRecord) (err error) {
	h.lcs = append(h.lcs, sr)
	return nil
}

func TestStoreLoadSigningRecords(t *testing.T) {
	const id = "9d017fad-2961-4fcc-94a9-1e1291a88ffc"
	const id1 = "9d017fad-2961-4fcc-94a9-1e1291a88ffd"
	const id2 = "9d017fad-2961-4fcc-94a9-1e1291a88ffe"
	const owner = "owner"
	const differentOwner = "owner2"
	const differentOwnerRecordId = "9d017fad-2961-4fcc-94a9-1e1291a88fff"
	upds := pb.SigningRecords{
		{
			Id:           id,
			Owner:        owner,
			CommonName:   "commonName",
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(0),
			CreationDate: constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
				Serial:         big.NewInt(42).String(),
				IssuerId:       "42424242-4242-4242-4242-424242424242",
			},
		},
		{
			Id:           id1,
			Owner:        owner,
			CommonName:   "commonName1",
			CreationDate: constDate().UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(1),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
				Serial:         big.NewInt(42).String(),
				IssuerId:       "42424242-4242-4242-4242-424242424242",
			},
		},
		{
			Id:           id2,
			Owner:        owner,
			CommonName:   "commonName2",
			CreationDate: constDate().UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(2),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
				Serial:         big.NewInt(42).String(),
				IssuerId:       "42424242-4242-4242-4242-424242424242",
			},
		},
		{
			Id:           differentOwnerRecordId,
			Owner:        differentOwner,
			CommonName:   "commonName2",
			CreationDate: constDate().UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(3),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
				Serial:         big.NewInt(42).String(),
				IssuerId:       "42424242-4242-4242-4242-424242424242",
			},
		},
	}

	lcs := upds

	type args struct {
		owner string
		query *store.SigningRecordsQuery
	}
	tests := []struct {
		name string
		args args
		want pb.SigningRecords
	}{
		{
			name: "all",
			args: args{
				query: nil,
			},
			want: lcs,
		},
		{
			name: "id",
			args: args{
				owner: owner,
				query: &store.SigningRecordsQuery{IdFilter: []string{lcs[1].GetId()}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "commonName",
			args: args{
				owner: owner,
				query: &store.SigningRecordsQuery{CommonNameFilter: []string{lcs[1].GetCommonName()}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "DeviceID",
			args: args{
				owner: owner,
				query: &store.SigningRecordsQuery{DeviceIdFilter: []string{lcs[1].GetDeviceId()}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "multiple queries",
			args: args{
				owner: owner,
				query: &store.SigningRecordsQuery{IdFilter: []string{lcs[0].GetId(), lcs[2].GetId()}},
			},
			want: []*store.SigningRecord{lcs[0], lcs[2]},
		},
		{
			name: "different owner",
			args: args{
				owner: differentOwner,
			},
			want: []*store.SigningRecord{lcs[3]},
		},
		{
			name: "different owner - id",
			args: args{
				owner: differentOwner,
				query: &store.SigningRecordsQuery{IdFilter: []string{differentOwnerRecordId}},
			},
			want: []*store.SigningRecord{lcs[3]},
		},
		{
			name: "different owner but id belongs to owner",
			args: args{
				owner: differentOwner,
				query: &store.SigningRecordsQuery{IdFilter: []string{lcs[1].GetId()}},
			},
		},
		{
			name: "all records",
			args: args{
				owner: "",
			},
			want: lcs,
		},
		{
			name: "not found",
			args: args{
				owner: owner,
				query: &store.SigningRecordsQuery{IdFilter: []string{"not found"}},
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, l := range upds {
		err := s.CreateSigningRecord(ctx, l)
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testSigningRecordHandler
			err := s.LoadSigningRecords(ctx, tt.args.owner, tt.args.query, h.process)
			require.NoError(t, err)
			require.Len(t, h.lcs, len(tt.want))
			h.lcs.Sort()
			tt.want.Sort()

			for i := range h.lcs {
				hubTest.CheckProtobufs(t, tt.want[i], h.lcs[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
