package service

import (
	"fmt"
	"net/http"

	pbCA "github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/kit/codec/json"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type signCertificateRequest struct {
	CSR string
}

type signCertificateResponse struct {
	CertificateChain string
}

func (requestHandler *RequestHandler) signCertificate(w http.ResponseWriter, r *http.Request) {
	if requestHandler.caClient == nil {
		writeError(w, status.Errorf(codes.Unimplemented, "certificate authority address is not set"))
		return
	}

	var body signCertificateRequest
	if err := json.ReadFrom(r.Body, &body); err != nil {
		writeError(w, fmt.Errorf("invalid json body: %w", err))
		return
	}

	ctx := kitNetGrpc.CtxWithToken(r.Context(), getAccessToken(r.Header))
	response, err := requestHandler.caClient.SignCertificate(ctx, &pbCA.SignCertificateRequest{
		CertificateSigningRequest: []byte(body.CSR),
	})
	if err != nil {
		writeError(w, fmt.Errorf("cannot sign certificate: %w", err))
		return
	}

	jsonResponseWriter(w, signCertificateResponse{
		CertificateChain: string(response.GetCertificate()),
	})
}
