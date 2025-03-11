package mongodb_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRevocationList(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	id := uuid.NewString()
	id2 := uuid.NewString()
	id3 := uuid.NewString()
	cert1 := &store.RevocationListCertificate{
		Serial:     "1",
		ValidUntil: time.Now().Add(time.Hour).Unix(),
		Revocation: time.Now().Unix(),
	}
	rl1 := store.RevocationList{
		Id:           id,
		Number:       "1",
		IssuedAt:     time.Now().UnixNano(),
		ValidUntil:   time.Now().Add(time.Minute).UnixNano(),
		Certificates: []*store.RevocationListCertificate{cert1},
	}
	cert2 := &store.RevocationListCertificate{
		Serial:     "2",
		ValidUntil: time.Now().Add(time.Hour).Unix(),
		Revocation: time.Now().Unix(),
	}
	cert3 := &store.RevocationListCertificate{
		Serial:     "2",
		ValidUntil: time.Now().Add(time.Hour).Unix(),
		Revocation: time.Now().Unix(),
	}
	rl3 := store.RevocationList{
		Id:         id3,
		Number:     "1",
		IssuedAt:   time.Now().Add(-time.Minute).UnixNano(),
		ValidUntil: time.Now().UnixNano(),
	}
	type args struct {
		query store.UpdateRevocationListQuery
	}
	tests := []struct {
		name    string
		args    args
		want    *store.RevocationList
		wantErr bool
	}{
		{
			name: "missing ID",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID:            "",
					RevokedCertificates: []*store.RevocationListCertificate{cert1},
				},
			},
			wantErr: true,
		},
		{
			name: "missing serial number",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID: id,
					RevokedCertificates: []*store.RevocationListCertificate{{
						Revocation: time.Now().UnixNano(),
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "missing revocation time",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID: id,
					RevokedCertificates: []*store.RevocationListCertificate{{
						Serial: "1",
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "valid - new document",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID:            rl1.Id,
					RevokedCertificates: rl1.Certificates,
					IssuedAt:            rl1.IssuedAt,
					ValidUntil:          rl1.ValidUntil,
				},
			},
			want: &rl1,
		},
		{
			name: "valid - add to existing document",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID:            id,
					RevokedCertificates: []*store.RevocationListCertificate{cert2},
				},
			},
			want: &store.RevocationList{
				Id:     id,
				Number: "2",
				Certificates: []*store.RevocationListCertificate{
					cert1,
					cert2,
				},
			},
		},
		{
			name: "valid - duplicate serial, noop",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID: id,
					RevokedCertificates: []*store.RevocationListCertificate{{
						Serial:     cert2.Serial,
						ValidUntil: time.Now().Add(time.Hour).Unix(),
						Revocation: time.Now().Unix(),
					}},
				},
			},
			want: &store.RevocationList{
				Id:     id,
				Number: "2",
				Certificates: []*store.RevocationListCertificate{
					cert1,
					cert2,
				},
			},
		},
		{
			name: "valid - different issuer, existing serial",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID:            id2,
					RevokedCertificates: []*store.RevocationListCertificate{cert3},
				},
			},
			want: &store.RevocationList{
				Id:           id2,
				Number:       "1",
				Certificates: []*store.RevocationListCertificate{cert3},
			},
		},
		{
			name: "valid - no certificates, set to expired",
			args: args{
				query: store.UpdateRevocationListQuery{
					IssuerID:   rl3.Id,
					IssuedAt:   rl3.IssuedAt,
					ValidUntil: rl3.ValidUntil,
				},
			},
			want: &rl3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedRL, err := s.UpdateRevocationList(ctx, &tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CheckRevocationList(t, tt.want, updatedRL, false)
		})
	}
}

func TestParallelUpdateRevocationList(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	issuerID := uuid.NewString()
	firstCount := 10
	secondCount := 10
	certificates := make([]*store.RevocationListCertificate, firstCount+secondCount)
	for i := 0; i < firstCount+secondCount; i++ {
		certificates[i] = test.GetCertificate(i, time.Now(), time.Now().Add(time.Hour))
	}

	// create or update
	createOrUpdateRevocationList(ctx, t, 0, firstCount, certificates, issuerID, s)

	rl, err := s.GetLatestIssuedOrIssueRevocationList(ctx, issuerID, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, rl.IssuedAt)
	require.NotEmpty(t, rl.ValidUntil)
	expected := &store.RevocationList{
		Id:           issuerID,
		Number:       "1",
		IssuedAt:     rl.IssuedAt,
		ValidUntil:   rl.ValidUntil,
		Certificates: certificates[:10],
	}
	test.CheckRevocationList(t, expected, rl, true)

	createOrUpdateRevocationList(ctx, t, firstCount, secondCount, certificates, issuerID, s)

	rl, err = s.GetLatestIssuedOrIssueRevocationList(ctx, issuerID, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, rl.IssuedAt)
	require.NotEmpty(t, rl.ValidUntil)
	expected = &store.RevocationList{
		Id:           issuerID,
		Number:       "2",
		IssuedAt:     rl.IssuedAt,
		ValidUntil:   rl.ValidUntil,
		Certificates: certificates,
	}
	test.CheckRevocationList(t, expected, rl, true)
}

func createOrUpdateRevocationList(ctx context.Context, t *testing.T, start, count int, certificates []*store.RevocationListCertificate, issuerID string, s *mongodb.Store) {
	var failed atomic.Bool
	failed.Store(false)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := start; i < start+count; i++ {
		go func(index int) {
			defer wg.Done()
			cert := certificates[index]
			var err error
			// parallel execution should eventually succeed in cases when we get duplicate _id
			// or not found errors
			for range 100 {
				q := &store.UpdateRevocationListQuery{
					IssuerID:            issuerID,
					RevokedCertificates: []*store.RevocationListCertificate{cert},
				}
				_, err = s.UpdateRevocationList(ctx, q)
				if errors.Is(err, store.ErrDuplicateID) || errors.Is(err, store.ErrNotFound) {
					continue
				}
				if err == nil {
					break
				}
				failed.Store(true)
				assert.NoError(t, err)
			}
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
	require.False(t, failed.Load())
}

func TestGetRevocationList(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	stored := test.AddRevocationListToStore(ctx, t, s, time.Now().Add(-2*time.Hour-time.Minute))

	type args struct {
		issuerID       string
		includeExpired bool
	}
	tests := []struct {
		name    string
		args    args
		want    *store.RevocationList
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				issuerID: "not-an-uuid",
			},
			wantErr: true,
		},
		{
			name: "no matching ID",
			args: args{
				issuerID: "00000000-0000-0000-0000-123456789012",
			},
			wantErr: true,
		},
		{
			name: "all from issuer0",
			args: args{
				issuerID:       test.GetIssuerID(0),
				includeExpired: true,
			},
			want: func() *store.RevocationList {
				expected, ok := stored[test.GetIssuerID(0)]
				require.True(t, ok)
				return expected
			}(),
		},
		{
			name: "no valid from issuer0",
			args: args{
				issuerID: test.GetIssuerID(0),
			},
			wantErr: true,
		},
		{
			name: "non-expired from issuer4",
			args: args{
				issuerID: test.GetIssuerID(4),
			},
			want: func() *store.RevocationList {
				expected, ok := stored[test.GetIssuerID(4)]
				require.True(t, ok)
				return expected
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := s.GetRevocationList(ctx, tt.args.issuerID, tt.args.includeExpired)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CheckRevocationList(t, tt.want, retrieved, false)
		})
	}
}
