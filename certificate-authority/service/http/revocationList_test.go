package http_test

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	certAuthURI "github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func checkRevocationList(t *testing.T, crl *x509.RevocationList, certificates []*store.RevocationListCertificate) {
	require.NotEmpty(t, crl.ThisUpdate)
	require.NotEmpty(t, crl.NextUpdate)
	expected := make([]x509.RevocationListEntry, 0, len(certificates))
	for _, cert := range certificates {
		serial, err := store.ParseBigInt(cert.Serial)
		require.NoError(t, err)
		expected = append(expected, x509.RevocationListEntry{
			SerialNumber:   serial,
			RevocationTime: pkgTime.Unix(pkgTime.Unix(0, cert.Revocation).Unix(), 0).UTC(),
		})
	}
	actual := make([]x509.RevocationListEntry, 0, len(crl.RevokedCertificateEntries))
	for _, cert := range crl.RevokedCertificateEntries {
		newCert := cert
		newCert.Raw = nil
		actual = append(actual, newCert)
	}
	require.Equal(t, expected, actual)
}

func addRevocationLists(ctx context.Context, t *testing.T, s store.Store) map[string]*store.RevocationList {
	rlm := make(map[string]*store.RevocationList)
	// valid
	now := time.Now()
	rl1 := &store.RevocationList{
		Id:         test.GetIssuerID(0),
		IssuedAt:   now.Add(-time.Minute).UnixNano(),
		ValidUntil: now.Add(time.Minute * 10).UnixNano(),
		Number:     "1",
	}
	for i := range 10 {
		rlc := test.GetCertificate(i, now, now.Add(time.Hour))
		rl1.Certificates = append(rl1.Certificates, rlc)
	}
	rlm[rl1.Id] = rl1

	// not issued
	rl2 := &store.RevocationList{
		Id:     test.GetIssuerID(1),
		Number: "2",
	}
	for i := range 10 {
		rlc := test.GetCertificate(i, now, now.Add(time.Hour))
		rl2.Certificates = append(rl2.Certificates, rlc)
	}
	rlm[rl2.Id] = rl2

	// expired
	rl3 := &store.RevocationList{
		Id:         test.GetIssuerID(2),
		IssuedAt:   now.Add(-time.Hour).UnixNano(),
		ValidUntil: now.Add(-time.Minute).UnixNano(),
		Number:     "3",
	}
	for i := range 10 {
		rlc := test.GetCertificate(i, now.Add(-time.Hour), now.Add(-time.Minute))
		rl3.Certificates = append(rl3.Certificates, rlc)
	}
	rlm[rl3.Id] = rl3

	err := s.InsertRevocationLists(ctx, maps.Values(rlm)...)
	require.NoError(t, err)
	return rlm
}

func TestRevocationList(t *testing.T) {
	if config.ACTIVE_DATABASE() == database.CqlDB {
		t.Skip("revocation list not supported for CqlDB")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	shutDown := testService.SetUpServices(context.Background(), t, testService.SetUpServicesOAuth|testService.SetUpServicesMachine2MachineOAuth)
	defer shutDown()
	caShutdown := test.New(t, test.MakeConfig(t))
	defer caShutdown()
	s, cleanUpStore := test.NewStore(t)
	defer cleanUpStore()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	stored := addRevocationLists(ctx, t, s)

	type args struct {
		issuer string
	}
	tests := []struct {
		name      string
		args      args
		verifyCRL func(crl *x509.RevocationList)
		wantErr   bool
	}{
		{
			name: "invalid issuerID",
			args: args{
				issuer: "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				issuer: test.GetIssuerID(0),
			},
			verifyCRL: func(crl *x509.RevocationList) {
				var certificates []*store.RevocationListCertificate
				for _, issuerCerts := range stored {
					if issuerCerts.Id != test.GetIssuerID(0) {
						continue
					}
					certificates = append(certificates, issuerCerts.Certificates...)
				}
				checkRevocationList(t, crl, certificates)
			},
		},
		{
			name: "valid - not issued",
			args: args{
				issuer: test.GetIssuerID(1),
			},
			verifyCRL: func(crl *x509.RevocationList) {
				var certificates []*store.RevocationListCertificate
				for _, issuerCerts := range stored {
					if issuerCerts.Id != test.GetIssuerID(1) {
						continue
					}
					certificates = append(certificates, issuerCerts.Certificates...)
				}
				checkRevocationList(t, crl, certificates)
			},
		},
		{
			name: "expired",
			args: args{
				issuer: test.GetIssuerID(2),
			},
			verifyCRL: func(crl *x509.RevocationList) {
				var certificates []*store.RevocationListCertificate
				for _, issuerCerts := range stored {
					if issuerCerts.Id != test.GetIssuerID(2) {
						continue
					}
					certificates = append(certificates, issuerCerts.Certificates...)
				}
				checkRevocationList(t, crl, certificates)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, certAuthURI.SigningRevocationList, nil).Host(config.CERTIFICATE_AUTHORITY_HTTP_HOST).AddIssuerID(tt.args.issuer).Build()
			httpResp := httpgwTest.HTTPDo(t, request)
			respBody, err := io.ReadAll(httpResp.Body)
			require.NoError(t, err)
			err = httpResp.Body.Close()
			require.NoError(t, err)
			crl, err := x509.ParseRevocationList(respBody)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			tt.verifyCRL(crl)
		})
	}
}

func TestParallelIssueAndUpdateRevocationList(t *testing.T) {
	if config.ACTIVE_DATABASE() == database.CqlDB {
		t.Skip("revocation list not supported for CqlDB")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	shutDown := testService.SetUpServices(context.Background(), t, testService.SetUpServicesOAuth|testService.SetUpServicesMachine2MachineOAuth)
	defer shutDown()
	caShutdown := test.New(t, test.MakeConfig(t))
	defer caShutdown()
	s, cleanUpStore := test.NewStore(t)
	defer cleanUpStore()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	issuerID := uuid.NewString()
	const iterations = 100
	const iterationsPerHttpRequest = 10
	certificates := make([]*store.RevocationListCertificate, iterations)
	for i := 0; i < len(certificates); i++ {
		certificates[i] = test.GetCertificate(i, time.Now(), time.Now().Add(time.Hour))
	}

	q := &store.UpdateRevocationListQuery{
		IssuerID: issuerID,
	}
	_, err := s.UpdateRevocationList(ctx, q)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(iterations)
	doUpdate := func(i int) {
		go func(index int) {
			defer wg.Done()
			for {
				cert := certificates[index]
				q := &store.UpdateRevocationListQuery{
					IssuerID:            issuerID,
					RevokedCertificates: []*store.RevocationListCertificate{cert},
				}
				_, err := s.UpdateRevocationList(ctx, q)
				if errors.Is(err, store.ErrDuplicateID) || errors.Is(err, store.ErrNotFound) {
					continue
				}
				if err == nil {
					break
				}
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Add(iterations)
	doIssue := func() {
		go func() {
			defer wg.Done()
			for {
				_, err := s.GetLatestIssuedOrIssueRevocationList(ctx, issuerID, time.Second)
				if errors.Is(err, store.ErrDuplicateID) || errors.Is(err, store.ErrNotFound) {
					continue
				}
				if err == nil {
					break
				}
				assert.NoError(t, err)
			}
		}()
	}

	wg.Add(iterations / iterationsPerHttpRequest)
	doIssueByHttp := func() {
		go func() {
			defer wg.Done()
			request := httpgwTest.NewRequest(http.MethodGet, certAuthURI.SigningRevocationList, nil).Host(config.CERTIFICATE_AUTHORITY_HTTP_HOST).AddIssuerID(issuerID).Build()
			httpResp := httpgwTest.HTTPDo(t, request)
			_ = httpResp.Body.Close()
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		}()
	}

	for iter := range iterations {
		doIssue()
		doUpdate(iter)
		if iter%iterationsPerHttpRequest == 0 {
			doIssueByHttp()
		}
	}
	wg.Wait()
}
