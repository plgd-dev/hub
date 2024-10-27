package test

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

var (
	serial0 = rand.Int31()
	serials = make(map[int]string)
)

func GetIssuerID(i int) string {
	return fmt.Sprintf("49000000-0000-0000-0000-%012d", i)
}

func getCertificateSerial(i int) string {
	id, ok := serials[i]
	if !ok {
		id = strconv.FormatInt(int64(serial0)+int64(i), 10)
		serials[i] = id
	}
	return id
}

func GetCertificate(c int, rev, exp time.Time) *store.RevocationListCertificate {
	return &store.RevocationListCertificate{
		Serial:     getCertificateSerial(c),
		ValidUntil: pkgTime.UnixNano(exp),
		Revocation: pkgTime.UnixNano(rev),
	}
}

// AddRevocationListToStore creates and stores multiple revocation lists with certificates.
// It returns a map of created revocation lists indexed by their IDs.
func AddRevocationListToStore(ctx context.Context, t *testing.T, s store.Store, expirationStart time.Time) map[string]*store.RevocationList {
	rlm := make(map[string]*store.RevocationList)
	c := 0
	for i := range 10 {
		now := time.Now()
		rlID := GetIssuerID(i)
		actual := &store.RevocationList{
			Id:         rlID,
			IssuedAt:   now.UnixNano(),
			ValidUntil: now.Add(time.Minute * 10).UnixNano(),
			Number:     strconv.Itoa(i),
		}
		exp := expirationStart.Add(time.Duration(i) * time.Hour)
		for range 10 {
			rlc := GetCertificate(c, now, exp)
			actual.Certificates = append(actual.Certificates, rlc)
			c++
		}
		rlm[rlID] = actual
	}

	err := s.InsertRevocationLists(ctx, maps.Values(rlm)...)
	require.NoError(t, err)
	return rlm
}

func CheckRevocationList(t *testing.T, expected, actual *store.RevocationList, ignoreRevocationTime bool) {
	require.Equal(t, expected.Number, actual.Number)
	require.Equal(t, expected.IssuedAt, actual.IssuedAt)
	require.Equal(t, expected.ValidUntil, actual.ValidUntil)
	require.Len(t, actual.Certificates, len(expected.Certificates))
	sort.Slice(actual.Certificates, func(i, j int) bool {
		return actual.Certificates[i].Serial < actual.Certificates[j].Serial
	})
	sort.Slice(expected.Certificates, func(i, j int) bool {
		return expected.Certificates[i].Serial < expected.Certificates[j].Serial
	})
	for i := range actual.Certificates {
		require.Equal(t, expected.Certificates[i].Serial, actual.Certificates[i].Serial)
		require.Equal(t, expected.Certificates[i].ValidUntil, actual.Certificates[i].ValidUntil)
		if !ignoreRevocationTime {
			require.Equal(t, expected.Certificates[i].Revocation, actual.Certificates[i].Revocation)
		}
	}
}
