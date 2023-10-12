package cqldb_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreInsertSigningRecord(t *testing.T) {
	date := time.Now().Add(time.Hour)
	type args struct {
		sub *store.SigningRecord
	}
	createSigningRecord := &store.SigningRecord{
		Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
		Owner:        "owner",
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           date.UnixNano() - 1,
			ValidUntilDate: date.UnixNano() - 1,
		},
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
						Date:           date.UnixNano(),
						ValidUntilDate: date.UnixNano(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "first insert",
			args: args{
				sub: createSigningRecord,
			},
		},
		{
			name: "second insert - fails",
			args: args{
				createSigningRecord,
			},
			wantErr: true,
		},
	}

	s, cleanUpStore := test.NewCQLStore(t)
	defer cleanUpStore()

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreateSigningRecord(ctx, tt.args.sub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStoreUpdateSigningRecord(t *testing.T) {
	date := time.Now().Add(time.Hour)
	date1 := date.Add(time.Hour)
	type args struct {
		sub *store.SigningRecord
	}
	createSigningRecord := &store.SigningRecord{
		Id:           "9d017fad-2961-4fcc-94a9-1e1291a88ffc",
		Owner:        "owner",
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           date.UnixNano() - 1,
			ValidUntilDate: date.UnixNano() - 1,
		},
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
						Date:           date.UnixNano(),
						ValidUntilDate: date.UnixNano(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "first update",
			args: args{
				sub: &store.SigningRecord{
					Id:           createSigningRecord.GetId(),
					Owner:        createSigningRecord.GetOwner(),
					CommonName:   createSigningRecord.GetCommonName(),
					PublicKey:    createSigningRecord.GetPublicKey(),
					CreationDate: date.UnixNano(),
					Credential: &pb.CredentialStatus{
						CertificatePem: "certificate",
						Date:           date.UnixNano(),
						ValidUntilDate: date.UnixNano(),
					},
				},
			},
		},
		{
			name: "second update",
			args: args{
				sub: &store.SigningRecord{
					Id:           createSigningRecord.GetId(),
					Owner:        createSigningRecord.GetOwner(),
					CommonName:   createSigningRecord.GetCommonName(),
					PublicKey:    createSigningRecord.GetPublicKey(),
					CreationDate: date.UnixNano(),
					Credential: &pb.CredentialStatus{
						CertificatePem: "certificate1",
						Date:           date1.UnixNano(),
						ValidUntilDate: date1.UnixNano(),
					},
				},
			},
		},
	}

	s, cleanUpStore := test.NewCQLStore(t)
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
			var h testSigningRecordHandler
			err = s.LoadSigningRecords(ctx, tt.args.sub.Owner, &pb.GetSigningRecordsRequest{
				IdFilter: []string{tt.args.sub.Id},
			}, h.Handle)
			require.NoError(t, err)
			require.Len(t, h.lcs, 1)
			hubTest.CheckProtobufs(t, tt.args.sub, h.lcs[0], hubTest.RequireToCheckFunc(require.Equal))
		})
	}
}

func TestStoreDeleteSigningRecord(t *testing.T) {
	const id1 = "9d017fad-2961-4fcc-94a9-1e1291a88ffc"
	deviceID1 := hubTest.GenerateDeviceIDbyIdx(1)
	const id2 = "9d017fad-2961-4fcc-94a9-1e1291a88ffd"
	deviceID2 := hubTest.GenerateDeviceIDbyIdx(2)
	const id3 = "9d017fad-2961-4fcc-94a9-1e1291a88ffe"
	deviceID3 := hubTest.GenerateDeviceIDbyIdx(3)
	const owner = "owner"
	const owner1 = "owner1"
	date := time.Now().Add(time.Hour)
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
					IdFilter: []string{uuid.Nil.String()},
				},
			},
			want: 0,
		},
		{
			name: "invalid owner",
			args: args{
				owner: "owner1",
				query: &store.DeleteSigningRecordsQuery{
					IdFilter: []string{id1},
				},
			},
			want: 0,
		},
		{
			name: "valid - by deviceID",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{
					DeviceIdFilter: []string{deviceID1},
				},
			},
			want: 1,
		},
		{
			name: "valid - by id",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{
					IdFilter: []string{id2},
				},
			},
			want: 1,
		},
		{
			name: "valid - empty",
			args: args{
				owner: owner,
				query: &store.DeleteSigningRecordsQuery{},
			},
			want: 0,
		},
		{
			name: "valid - by owner",
			args: args{
				owner: owner1,
				query: &store.DeleteSigningRecordsQuery{},
			},
			want: 1,
		},
	}

	s, cleanUpStore := test.NewCQLStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           id1,
		Owner:        owner,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		DeviceId:     deviceID1,
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           date.UnixNano(),
			ValidUntilDate: date.UnixNano(),
		},
	})
	require.NoError(t, err)
	err = s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           id2,
		Owner:        owner,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		DeviceId:     deviceID2,
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           date.UnixNano(),
			ValidUntilDate: date.UnixNano(),
		},
	})
	require.NoError(t, err)
	err = s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           id3,
		Owner:        owner1,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		DeviceId:     deviceID3,
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           date.UnixNano(),
			ValidUntilDate: date.UnixNano(),
		},
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := s.DeleteSigningRecords(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, n)
		})
	}
}

func TestStoreDeleteExpiredRecords(t *testing.T) {
	const id = "9d017fad-2961-4fcc-94a9-1e1291a88ffc"
	date := time.Now().Add(time.Second * 2)

	s, cleanUpStore := test.NewCQLStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           id,
		Owner:        "owner",
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		CreationDate: date.UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           date.UnixNano(),
			ValidUntilDate: date.UnixNano(),
		},
	})
	require.NoError(t, err)
	var h testSigningRecordHandler
	err = s.LoadSigningRecords(ctx, "owner", nil, h.Handle)
	require.NoError(t, err)
	require.Len(t, h.lcs, 1)
	time.Sleep(time.Second * 3)
	_, err = s.DeleteNonDeviceExpiredRecords(ctx, time.Now())
	require.Error(t, err)
	require.Equal(t, store.ErrNotSupported, err)
	var h1 testSigningRecordHandler
	err = s.LoadSigningRecords(ctx, "owner", nil, h1.Handle)
	require.NoError(t, err)
	require.Len(t, h1.lcs, 0)
}

type testSigningRecordHandler struct {
	lcs pb.SigningRecords
}

func (h *testSigningRecordHandler) Handle(ctx context.Context, iter store.SigningRecordIter) (err error) {
	for {
		var sub store.SigningRecord
		if !iter.Next(ctx, &sub) {
			break
		}
		h.lcs = append(h.lcs, &sub)
	}
	return iter.Err()
}

func TestStoreLoadSigningRecords(t *testing.T) {
	const id = "9d017fad-2961-4fcc-94a9-1e1291a88ffc"
	const id1 = "9d017fad-2961-4fcc-94a9-1e1291a88ffd"
	const id2 = "9d017fad-2961-4fcc-94a9-1e1291a88ffe"
	date := time.Now().Add(time.Hour)
	upds := pb.SigningRecords{
		{
			Id:           id,
			Owner:        "owner",
			CommonName:   "commonName",
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(1),
			CreationDate: date.UnixNano(),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           date.UnixNano(),
				ValidUntilDate: date.UnixNano(),
			},
		},
		{
			Id:           id1,
			Owner:        "owner",
			CommonName:   "commonName1",
			CreationDate: date.UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(2),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           date.UnixNano(),
				ValidUntilDate: date.UnixNano(),
			},
		},
		{
			Id:           id2,
			Owner:        "owner",
			CommonName:   "commonName2",
			CreationDate: date.UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(3),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           date.UnixNano(),
				ValidUntilDate: date.UnixNano(),
			},
		},
	}

	lcs := upds

	type args struct {
		owner string
		query *store.SigningRecordsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    pb.SigningRecords
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
				owner: "owner",
				query: &store.SigningRecordsQuery{IdFilter: []string{lcs[1].GetId()}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "commonName",
			args: args{
				owner: "owner",
				query: &store.SigningRecordsQuery{CommonNameFilter: []string{lcs[1].GetCommonName()}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "DeviceID",
			args: args{
				owner: "owner",
				query: &store.SigningRecordsQuery{DeviceIdFilter: []string{lcs[1].GetDeviceId()}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "id - another owner",
			args: args{
				owner: "another owner",
				query: &store.SigningRecordsQuery{IdFilter: []string{lcs[1].Id}},
			},
			want: []*store.SigningRecord{lcs[1]},
		},
		{
			name: "multiple queries",
			args: args{
				owner: "owner",
				query: &store.SigningRecordsQuery{IdFilter: []string{lcs[0].Id, lcs[2].Id}},
			},
			want: []*store.SigningRecord{lcs[0], lcs[2]},
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
				owner: "owner",
				query: &store.SigningRecordsQuery{IdFilter: []string{uuid.Nil.String()}},
			},
		},
		{
			name: "multiple queries with crossing data",
			args: args{
				owner: "owner",
				query: &store.SigningRecordsQuery{
					IdFilter:       []string{lcs[0].Id, lcs[2].Id},
					DeviceIdFilter: []string{lcs[0].GetDeviceId()},
				},
			},
			want: []*store.SigningRecord{lcs[0], lcs[2]},
		},
	}

	s, cleanUpStore := test.NewCQLStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, l := range upds {
		err := s.CreateSigningRecord(ctx, l)
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testSigningRecordHandler
			err := s.LoadSigningRecords(ctx, "owner", tt.args.query, h.Handle)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			require.Len(t, h.lcs, len(tt.want))
			h.lcs.Sort()
			tt.want.Sort()

			for i := range h.lcs {
				hubTest.CheckProtobufs(t, tt.want[i], h.lcs[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}

func BenchmarkSigningRecords(b *testing.B) {
	data := make([]*store.SigningRecord, 0, 5001)
	date := time.Now().Add(time.Hour)
	dataCap := cap(data)
	for i := 0; i < dataCap; i++ {
		data = append(data, &store.SigningRecord{
			Id:           hubTest.GenerateDeviceIDbyIdx(i),
			Owner:        "owner",
			CommonName:   "commonName" + strconv.Itoa(i),
			CreationDate: date.UnixNano(),
			PublicKey:    "publicKey",
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           date.UnixNano(),
				ValidUntilDate: date.UnixNano(),
			},
		})
	}

	ctx := context.Background()
	b.ResetTimer()
	s, cleanUpStore := test.NewCQLStore(b)
	defer cleanUpStore()
	for i := uint32(0); i < uint32(b.N); i++ {
		b.StopTimer()
		err := s.ClearTable(ctx)
		require.NoError(b, err)
		b.StartTimer()
		func() {
			var wg sync.WaitGroup
			wg.Add(len(data))
			for _, l := range data {
				go func(l *pb.SigningRecord) {
					defer wg.Done()
					err := s.UpdateSigningRecord(ctx, l)
					require.NoError(b, err)
				}(l)
			}
			wg.Wait()
		}()
	}
}
