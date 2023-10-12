package mongodb_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/assert"
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			err = s.FlushBulkWriter()
			require.NoError(t, err)
		})
	}
}

func TestStoreDeleteSigningRecord(t *testing.T) {
	const id1 = "9d017fad-2961-4fcc-94a9-1e1291a88ffc"
	deviceID1 := hubTest.GenerateDeviceIDbyIdx(1)
	const id2 = "9d017fad-2961-4fcc-94a9-1e1291a88ffd"
	deviceID2 := hubTest.GenerateDeviceIDbyIdx(2)
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
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           id1,
		Owner:        owner,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		DeviceId:     deviceID1,
		CreationDate: constDate().UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           constDate().UnixNano(),
			ValidUntilDate: constDate().UnixNano(),
		},
	})
	require.NoError(t, err)
	err = s.CreateSigningRecord(ctx, &store.SigningRecord{
		Id:           id2,
		Owner:        owner,
		CommonName:   "commonName",
		PublicKey:    "publicKey",
		DeviceId:     deviceID2,
		CreationDate: constDate().UnixNano(),
		Credential: &pb.CredentialStatus{
			CertificatePem: "certificate",
			Date:           constDate().UnixNano(),
			ValidUntilDate: constDate().UnixNano(),
		},
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := s.DeleteSigningRecords(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, n)
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
		},
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.DeleteNonDeviceExpiredRecords(ctx, tt.args.now)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
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
	upds := pb.SigningRecords{
		{
			Id:           id,
			Owner:        "owner",
			CommonName:   "commonName",
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(0),
			CreationDate: constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
			},
		},
		{
			Id:           id1,
			Owner:        "owner",
			CommonName:   "commonName1",
			CreationDate: constDate().UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(1),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
			},
		},
		{
			Id:           id2,
			Owner:        "owner",
			CommonName:   "commonName2",
			CreationDate: constDate().UnixNano(),
			PublicKey:    "publicKey",
			DeviceId:     hubTest.GenerateDeviceIDbyIdx(2),
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
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
			err := s.LoadSigningRecords(ctx, "owner", tt.args.query, h.Handle)
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

func BenchmarkSigningRecords(b *testing.B) {
	data := make([]*store.SigningRecord, 0, 5001)
	dataCap := cap(data)
	for i := 0; i < dataCap; i++ {
		data = append(data, &store.SigningRecord{
			Id:           hubTest.GenerateDeviceIDbyIdx(i),
			Owner:        "owner",
			CommonName:   "commonName" + strconv.Itoa(i),
			CreationDate: constDate().UnixNano(),
			PublicKey:    "publicKey",
			Credential: &pb.CredentialStatus{
				CertificatePem: "certificate",
				Date:           constDate().UnixNano(),
				ValidUntilDate: constDate().UnixNano(),
			},
		})
	}

	ctx := context.Background()
	b.ResetTimer()
	s, cleanUpStore := test.NewMongoStore(b)
	defer cleanUpStore()
	for i := uint32(0); i < uint32(b.N); i++ {
		b.StopTimer()
		err := s.Clear(ctx)
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
			err := s.FlushBulkWriter()
			require.NoError(b, err)
		}()
	}
}
