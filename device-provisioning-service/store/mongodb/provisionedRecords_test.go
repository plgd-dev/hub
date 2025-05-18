package mongodb_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
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

func TestStoreUpdateProvisioningRecord(t *testing.T) {
	const owner = "owner"
	type args struct {
		owner string
		sub   *store.ProvisioningRecord
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				sub: &store.ProvisioningRecord{
					Owner: owner,
					// Id: "deviceIDnotFound",
					EnrollmentGroupId: "enrollmentGroupID",
					Credential: &pb.CredentialStatus{
						IdentityCertificatePem: "certificate",
					},
					Attestation: &pb.Attestation{
						Date: constDate().UnixNano(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				owner: owner,
				sub: &store.ProvisioningRecord{
					Owner:             owner,
					Id:                "mfgID",
					EnrollmentGroupId: "enrollmentGroupID",
					Attestation: &pb.Attestation{
						Date: constDate().UnixNano(),
					},
				},
			},
		},
		{
			name: "valid - owner is not set in filter",
			args: args{
				sub: &store.ProvisioningRecord{
					Owner:             anotherOwner,
					Id:                "mfgID",
					EnrollmentGroupId: "enrollmentGroupID",
					Attestation: &pb.Attestation{
						Date: constDate().UnixNano(),
					},
				},
			},
		},
		{
			name: "invalid owner",
			args: args{
				owner: anotherOwner,
				sub: &store.ProvisioningRecord{
					Owner: anotherOwner,
					// Id: "deviceIDnotFound",
					EnrollmentGroupId: "enrollmentGroupID",
					Credential: &pb.CredentialStatus{
						IdentityCertificatePem: "certificate",
					},
					Attestation: &pb.Attestation{
						Date: constDate().UnixNano(),
					},
				},
			},
			wantErr: true,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateProvisioningRecord(ctx, tt.args.owner, tt.args.sub)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			err = s.FlushBulkWriter()
			require.NoError(t, err)
		})
	}
}

func TestStoreDeleteProvisioningRecords(t *testing.T) {
	const owner = "owner"
	type args struct {
		query *store.ProvisioningRecordsQuery
		owner string
	}
	tests := []struct {
		name      string
		args      args
		wantCount int64
	}{
		{
			name: "invalid cloudId",
			args: args{
				owner: owner,
				query: &store.ProvisioningRecordsQuery{
					IdFilter: []string{"notFound"},
				},
			},
			wantCount: 0,
		},
		{
			name: "another owner",
			args: args{
				owner: anotherOwner,
				query: &store.ProvisioningRecordsQuery{
					IdFilter: []string{"mfgID"},
				},
			},
			wantCount: 0,
		},
		{
			name: "valid",
			args: args{
				owner: owner,
				query: &store.ProvisioningRecordsQuery{
					IdFilter: []string{"mfgID"},
				},
			},
			wantCount: 1,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.UpdateProvisioningRecord(ctx, owner, &store.ProvisioningRecord{
		Owner:             owner,
		Id:                "mfgID",
		EnrollmentGroupId: "enrollmentGroupID",
		Attestation: &pb.Attestation{
			Date: constDate().UnixNano(),
		},
	})
	require.NoError(t, err)
	err = s.FlushBulkWriter()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.DeleteProvisioningRecords(ctx, tt.args.owner, tt.args.query)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, got)
		})
	}
}

type testProvisioningRecordHandler struct {
	lcs pb.ProvisioningRecords
}

func (h *testProvisioningRecordHandler) Handle(ctx context.Context, iter store.ProvisioningRecordIter) (err error) {
	for {
		var sub store.ProvisioningRecord
		if !iter.Next(ctx, &sub) {
			break
		}
		h.lcs = append(h.lcs, &sub)
	}
	return iter.Err()
}

func TestStoreLoadProvisioningRecords(t *testing.T) {
	const owner = "owner"
	upds := pb.ProvisioningRecords{
		{
			Id:                "mfgID",
			EnrollmentGroupId: "enrollmentGroupID",
			CreationDate:      constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate",
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Acl: &pb.ACLStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Cloud: &pb.CloudStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Attestation: &pb.Attestation{
				Date: constDate1().UnixNano(),
			},
			Owner: owner,
		},
		{
			Id:                "mfgID",
			EnrollmentGroupId: "enrollmentGroupID",
			CreationDate:      constDate1().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate1",
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Owner: owner,
		},
		{
			Id:                "mfgID",
			EnrollmentGroupId: "enrollmentGroupID",
			CreationDate:      constDate1().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate",
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Attestation: &pb.Attestation{
				Date: constDate().UnixNano(),
			},
			Acl: &pb.ACLStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Cloud: &pb.CloudStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Owner: owner,
		},
		{
			Id:                "mfgID2",
			EnrollmentGroupId: "enrollmentGroupID",
			CreationDate:      constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate",
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Attestation: &pb.Attestation{
				Date: constDate().UnixNano(),
			},
			Owner: owner,
		},
		{
			Id:                "mfgID3",
			EnrollmentGroupId: "enrollmentGroupID1",
			CreationDate:      constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate",
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Attestation: &pb.Attestation{
				Date: constDate().UnixNano(),
			},
			Owner: owner,
		},
	}

	lcs := pb.ProvisioningRecords{
		{
			Id:                "mfgID",
			EnrollmentGroupId: "enrollmentGroupID",
			CreationDate:      constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate1",
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Attestation: &pb.Attestation{
				Date: constDate1().UnixNano(),
			},
			Acl: &pb.ACLStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Cloud: &pb.CloudStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Owner: owner,
		},
		upds[len(upds)-2],
		upds[len(upds)-1],
	}

	type args struct {
		owner string
		query *store.ProvisioningRecordsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    pb.ProvisioningRecords
	}{
		{
			name: "all",
			args: args{
				owner: owner,
				query: nil,
			},
			want: lcs,
		},
		{
			name: "all - another owner",
			args: args{
				owner: anotherOwner,
				query: nil,
			},
		},
		{
			name: "id",
			args: args{
				owner: owner,
				query: &store.ProvisioningRecordsQuery{IdFilter: []string{lcs[1].GetId()}},
			},
			want: []*store.ProvisioningRecord{lcs[1]},
		},
		{
			name: "enrollmentGroupID",
			args: args{
				owner: owner,
				query: &store.ProvisioningRecordsQuery{EnrollmentGroupIdFilter: []string{lcs[0].GetEnrollmentGroupId()}},
			},
			want: []*store.ProvisioningRecord{lcs[0], lcs[1]},
		},
		{
			name: "multiple queries",
			args: args{
				owner: owner,
				query: &store.ProvisioningRecordsQuery{EnrollmentGroupIdFilter: []string{lcs[0].GetEnrollmentGroupId(), lcs[2].GetEnrollmentGroupId()}},
			},
			want: lcs,
		},
		{
			name: "not found",
			args: args{
				owner: owner,
				query: &store.ProvisioningRecordsQuery{IdFilter: []string{"not found"}},
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for idx, l := range upds {
		err := s.UpdateProvisioningRecord(ctx, l.GetOwner(), l)
		require.NoError(t, err)
		if idx%2 == 0 {
			time.Sleep(time.Second * 2)
		}
	}
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testProvisioningRecordHandler
			err := s.LoadProvisioningRecords(ctx, tt.args.owner, tt.args.query, h.Handle)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
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

func BenchmarkProvisioningRecords(b *testing.B) {
	data := make([]*store.ProvisioningRecord, 0, 5001)
	const owner = "owner"
	dataCap := cap(data)
	for i := range dataCap {
		data = append(data, &store.ProvisioningRecord{
			Id:                "mfgID" + strconv.Itoa(i),
			EnrollmentGroupId: "enrollmentGroupID",
			CreationDate:      constDate().UnixNano(),
			Credential: &pb.CredentialStatus{
				IdentityCertificatePem: "certificate",
				Status: &pb.ProvisionStatus{
					Date: constDate().UnixNano(),
				},
			},
			Acl: &pb.ACLStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Cloud: &pb.CloudStatus{
				Status: &pb.ProvisionStatus{
					Date: constDate1().UnixNano(),
				},
			},
			Attestation: &pb.Attestation{
				Date: constDate1().UnixNano(),
			},
			Owner: owner,
		})
	}

	ctx := context.Background()
	b.ResetTimer()
	s, cleanUpStore := test.NewMongoStore(b)
	defer cleanUpStore()
	for range uint32(b.N) {
		b.StopTimer()
		err := s.Clear(ctx)
		require.NoError(b, err)
		b.StartTimer()
		func() {
			var wg sync.WaitGroup
			wg.Add(len(data))
			for _, l := range data {
				go func(l *pb.ProvisioningRecord) {
					defer wg.Done()
					err := s.UpdateProvisioningRecord(ctx, l.GetOwner(), l)
					assert.NoError(b, err)
				}(l)
			}
			wg.Wait()
			err := s.FlushBulkWriter()
			require.NoError(b, err)
		}()
	}
}
