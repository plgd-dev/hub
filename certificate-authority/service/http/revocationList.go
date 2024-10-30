package http

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"google.golang.org/grpc/codes"
)

func errCannotGetRevocationList(err error) error {
	return pkgGrpc.ForwardErrorf(codes.Internal, "cannot get revocation list: %v", err)
}

func createCRL(rl *store.RevocationList, issuer *x509.Certificate, priv crypto.Signer) ([]byte, error) {
	number, err := store.ParseBigInt(rl.Number)
	if err != nil {
		return nil, err
	}
	entries := make([]x509.RevocationListEntry, 0, len(rl.Certificates))
	for _, c := range rl.Certificates {
		sn, errP := store.ParseBigInt(c.Serial)
		if errP != nil {
			return nil, errP
		}
		entries = append(entries, x509.RevocationListEntry{
			SerialNumber:   sn,
			RevocationTime: pkgTime.Unix(0, c.Revocation),
		})
	}
	template := &x509.RevocationList{
		Number:                    number,
		ThisUpdate:                pkgTime.Unix(0, rl.IssuedAt),
		NextUpdate:                pkgTime.Unix(0, rl.ValidUntil),
		RevokedCertificateEntries: entries,
	}
	return x509.CreateRevocationList(rand.Reader, template, issuer, priv)
}

func (requestHandler *requestHandler) tryGetRevocationList(ctx context.Context, issuerID string, validFor time.Duration) (*store.RevocationList, error) {
	rl, err := requestHandler.store.GetLatestIssuedOrIssueRevocationList(ctx, issuerID, validFor)
	if err == nil {
		return rl, nil
	}
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrDuplicateID) {
		// this only occurs if some parallel request updated the revocation list first, in that case we should
		// just retrieve the updated one
		return requestHandler.store.GetRevocationList(ctx, issuerID, true)
	}
	return nil, err
}

func (requestHandler *requestHandler) writeRevocationList(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	issuerID := vars[uri.IssuerIDKey]
	if _, err := uuid.Parse(issuerID); err != nil {
		return err
	}
	signer := requestHandler.cas.GetSigner()
	_, validFor := signer.GetCRLConfiguration()
	rl, err := requestHandler.tryGetRevocationList(r.Context(), issuerID, validFor)
	if err != nil {
		return err
	}
	crl, err := createCRL(rl, signer.GetCertificate(), signer.GetPrivateKey())
	if err != nil {
		return err
	}
	w.Header().Set(pkgHttp.ContentTypeHeaderKey, "application/pkix-crl")
	_, err = w.Write(crl)
	return err
}

func (requestHandler *requestHandler) revocationList(w http.ResponseWriter, r *http.Request) {
	if err := requestHandler.writeRevocationList(w, r); err != nil {
		serverMux.WriteError(w, errCannotGetRevocationList(err))
	}
}
