package mongodb_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

const anotherOwner = "anotherOwner"

func createCACertificate(t require.TestingT, commonName string) string {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = commonName
	cfg.ValidFor = time.Hour * 24
	cfg.BasicConstraints.MaxPathLen = 1000
	rootCA, err := generateCertificate.GenerateRootCA(cfg, priv)
	require.NoError(t, err)

	return "data:;base64," + base64.StdEncoding.EncodeToString(rootCA)
}

func TestStoreCreateEnrollmentGroup(t *testing.T) {
	id := uuid.NewString()
	eg := test.NewEnrollmentGroup(t, id, test.DPSOwner)
	eg1 := test.NewEnrollmentGroup(t, id, anotherOwner)
	type args struct {
		owner string
		eg    *store.EnrollmentGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				eg: &store.EnrollmentGroup{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				owner: eg.GetOwner(),
				eg:    eg,
			},
		},
		{
			name: "duplicity",
			args: args{
				owner: eg.GetOwner(),
				eg:    eg,
			},
			wantErr: true,
		},
		{
			name: anotherOwner,
			args: args{
				owner: eg1.GetOwner(),
				eg:    eg1,
			},
			wantErr: true,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreateEnrollmentGroup(ctx, tt.args.owner, tt.args.eg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStoreUpdateEnrollmentGroup(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()
	ctx := context.Background()
	id := uuid.NewString()
	eg := test.NewEnrollmentGroup(t, id, test.DPSOwner)
	eg1 := test.NewEnrollmentGroup(t, id, anotherOwner)
	err := s.CreateEnrollmentGroup(ctx, eg.GetOwner(), eg)
	require.NoError(t, err)
	eg.Name = "abcd"
	type args struct {
		owner string
		eg    *store.EnrollmentGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				owner: eg.GetOwner(),
				eg:    &store.EnrollmentGroup{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				owner: eg.GetOwner(),
				eg:    eg,
			},
		},
		{
			name: "duplicity",
			args: args{
				owner: eg.GetOwner(),
				eg:    eg,
			},
		},
		{
			name: "not exist",
			args: args{
				owner: eg.GetOwner(),
				eg:    &store.EnrollmentGroup{Id: "notExist"},
			},
			wantErr: true,
		},
		{
			name: "another user tries to replace original owner",
			args: args{
				owner: eg1.GetOwner(),
				eg:    eg1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateEnrollmentGroup(ctx, tt.args.owner, tt.args.eg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStoreWatchEnrollmentGroup(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()
	ctx := context.Background()
	watcher, err := s.WatchEnrollmentGroups(ctx)
	require.NoError(t, err)
	eg := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	err = s.CreateEnrollmentGroup(ctx, eg.GetOwner(), eg)
	require.NoError(t, err)
	egUpd := pb.EnrollmentGroup{
		Id:                   eg.GetId(),
		AttestationMechanism: eg.GetAttestationMechanism(),
		HubIds:               eg.GetHubIds(),
		Name:                 "myName",
		Owner:                eg.GetOwner(),
	}
	err = s.UpdateEnrollmentGroup(ctx, egUpd.GetOwner(), &egUpd)
	require.NoError(t, err)
	_, err = s.DeleteEnrollmentGroups(ctx, eg.GetOwner(), &pb.GetEnrollmentGroupsRequest{
		IdFilter: []string{eg.GetId()},
	})
	require.NoError(t, err)
	expectedEvents := []struct {
		event store.Event
		id    string
	}{
		{
			event: store.EventUpdate,
			id:    eg.GetId(),
		},
		{
			event: store.EventDelete,
			id:    eg.GetId(),
		},
	}
	for _, expectedEvent := range expectedEvents {
		event, id, ok := watcher.Next(ctx)
		require.True(t, ok)
		require.Equal(t, expectedEvent.id, id)
		require.Equal(t, expectedEvent.event, event)
	}
	require.NoError(t, watcher.Close())
}

func TestStoreDeleteEnrollmentGroup(t *testing.T) {
	egs := []*store.EnrollmentGroup{
		test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner),
		test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner),
		test.NewEnrollmentGroup(t, uuid.NewString(), anotherOwner),
	}
	type args struct {
		owner string
		query *store.EnrollmentGroupsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		count   int64
	}{
		{
			name: "invalid cloudId",
			args: args{
				owner: egs[0].GetOwner(),
				query: &store.EnrollmentGroupsQuery{
					IdFilter: []string{"notFound"},
				},
			},
			wantErr: true,
		},

		{
			name: "valid",
			args: args{
				owner: egs[0].GetOwner(),
				query: &store.EnrollmentGroupsQuery{
					IdFilter: []string{egs[0].GetId()},
				},
			},
			count: 1,
		},
		{
			name: "valid multiple",
			args: args{
				owner: egs[0].GetOwner(),
				query: &store.EnrollmentGroupsQuery{
					IdFilter: []string{egs[0].GetId(), egs[1].GetId()},
				},
			},
			count: 1,
		},
		{
			name: "delete all owner",
			args: args{
				owner: egs[0].GetOwner(),
			},
			wantErr: true,
		},
		{
			name: "delete all another owner",
			args: args{
				owner: egs[2].GetOwner(),
			},
			count: 1,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, eg := range egs {
		err := s.UpsertEnrollmentGroup(ctx, eg.GetOwner(), eg)
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := s.DeleteEnrollmentGroups(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.count, count)
		})
	}
}

type testEnrollmentGroupHandler struct {
	lcs pb.EnrollmentGroups
}

func (h *testEnrollmentGroupHandler) Handle(ctx context.Context, iter store.EnrollmentGroupIter) (err error) {
	for {
		var eg store.EnrollmentGroup
		if !iter.Next(ctx, &eg) {
			break
		}
		h.lcs = append(h.lcs, &eg)
	}
	return iter.Err()
}

func TestStoreLoadEnrollmentGroups(t *testing.T) {
	eg0 := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	eg0.AttestationMechanism.X509.LeadCertificateName = "a"
	eg0.HubIds = []string{"75eacc2f-ac28-4a42-a155-164393970ba4"}
	eg0.AttestationMechanism.X509.CertificateChain = createCACertificate(t, eg0.GetAttestationMechanism().GetX509().GetLeadCertificateName())
	eg1 := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	eg1.AttestationMechanism.X509.LeadCertificateName = "b"
	eg1.HubIds = []string{"75eacc2f-ac28-4a42-a155-164393970ba4"}
	eg1.AttestationMechanism.X509.CertificateChain = createCACertificate(t, eg1.GetAttestationMechanism().GetX509().GetLeadCertificateName())
	eg2 := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	eg2.AttestationMechanism.X509.LeadCertificateName = "b"
	eg2.AttestationMechanism.X509.CertificateChain = createCACertificate(t, eg2.GetAttestationMechanism().GetX509().GetLeadCertificateName())
	eg3 := test.NewEnrollmentGroup(t, uuid.NewString(), test.DPSOwner)
	eg3.AttestationMechanism.X509.LeadCertificateName = "c"
	eg3.AttestationMechanism.X509.CertificateChain = createCACertificate(t, eg3.GetAttestationMechanism().GetX509().GetLeadCertificateName())
	eg4 := test.NewEnrollmentGroup(t, uuid.NewString(), anotherOwner)
	enrollmentGroups := pb.EnrollmentGroups{
		eg0,
		eg1,
		eg2,
		eg3,
		eg4,
	}

	type args struct {
		owner string
		query *store.EnrollmentGroupsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    pb.EnrollmentGroups
	}{
		{
			name: "all",
			args: args{
				query: nil,
			},
			want: enrollmentGroups,
		},
		{
			name: "all owner",
			args: args{
				owner: enrollmentGroups[0].GetOwner(),
				query: nil,
			},
			want: enrollmentGroups[:4],
		},

		{
			name: "all another owner",
			args: args{
				owner: enrollmentGroups[4].GetOwner(),
				query: nil,
			},
			want: enrollmentGroups[4:],
		},
		{
			name: "id",
			args: args{
				owner: enrollmentGroups[1].GetOwner(),
				query: &store.EnrollmentGroupsQuery{IdFilter: []string{enrollmentGroups[1].GetId()}},
			},
			want: []*store.EnrollmentGroup{enrollmentGroups[1]},
		},
		{
			name: "hubids",
			args: args{
				owner: enrollmentGroups[0].GetOwner(),
				query: &store.EnrollmentGroupsQuery{HubIdFilter: enrollmentGroups[0].GetHubIds()},
			},
			want: []*store.EnrollmentGroup{enrollmentGroups[0], enrollmentGroups[1]},
		},
		{
			name: "certificateName",
			args: args{
				owner: enrollmentGroups[1].GetOwner(),
				query: &store.EnrollmentGroupsQuery{AttestationMechanismX509CertificateNames: []string{enrollmentGroups[1].GetAttestationMechanism().GetX509().GetLeadCertificateName()}},
			},
			want: []*store.EnrollmentGroup{enrollmentGroups[1], enrollmentGroups[2]},
		},
		{
			name: "multiple queries",
			args: args{
				owner: enrollmentGroups[0].GetOwner(),
				query: &store.EnrollmentGroupsQuery{AttestationMechanismX509CertificateNames: []string{enrollmentGroups[0].GetAttestationMechanism().GetX509().GetLeadCertificateName(), enrollmentGroups[3].GetAttestationMechanism().GetX509().GetLeadCertificateName()}},
			},
			want: []*store.EnrollmentGroup{enrollmentGroups[0], enrollmentGroups[3]},
		},
		{
			name: "not found",
			args: args{
				owner: enrollmentGroups[0].GetOwner(),
				query: &store.EnrollmentGroupsQuery{IdFilter: []string{"not found"}},
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, l := range enrollmentGroups {
		err := s.CreateEnrollmentGroup(ctx, l.GetOwner(), l)
		require.NoError(t, err)
	}

	for _, v := range tests {
		tt := v
		t.Run(tt.name, func(t *testing.T) {
			var h testEnrollmentGroupHandler
			err := s.LoadEnrollmentGroups(ctx, tt.args.owner, tt.args.query, h.Handle)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, h.lcs, len(tt.want))
			h.lcs.Sort()
			want := make(pb.EnrollmentGroups, len(tt.want))
			copy(want, tt.want)
			want.Sort()

			for i := range h.lcs {
				hubTest.CheckProtobufs(t, want[i], h.lcs[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}

func BenchmarkEnrollmentGroups(b *testing.B) {
	numEnrollmentGroups := 1000
	ctx := context.Background()
	b.ResetTimer()
	s, cleanUpStore := test.NewMongoStore(b)
	defer cleanUpStore()
	ratioSharedCertificateNames := numEnrollmentGroups / 1000
	if ratioSharedCertificateNames < 1 {
		ratioSharedCertificateNames = 1
	}
	owner := uuid.NewString()
	for i := range numEnrollmentGroups {
		commonName := strconv.Itoa(i % ratioSharedCertificateNames)
		err := s.CreateEnrollmentGroup(ctx, owner, &store.EnrollmentGroup{
			Id: uuid.NewString(),
			AttestationMechanism: &pb.AttestationMechanism{
				X509: &pb.X509Configuration{
					CertificateChain:    createCACertificate(b, commonName),
					LeadCertificateName: commonName,
				},
			},
			Owner:  owner,
			HubIds: []string{uuid.NewString()},
		})
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := range b.N {
		var h testEnrollmentGroupHandler
		b.StartTimer()
		err := s.LoadEnrollmentGroups(ctx, owner, &pb.GetEnrollmentGroupsRequest{AttestationMechanismX509CertificateNames: []string{strconv.Itoa(i % ratioSharedCertificateNames)}}, h.Handle)
		b.StopTimer()
		require.NoError(b, err)
		require.NotEmpty(b, h.lcs)
	}
}
