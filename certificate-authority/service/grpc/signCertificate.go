package grpc

import (
	"context"
	"crypto/x509"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *CertificateAuthorityServer) validateRequest(csr []byte) error {
	infoData, err := getInfoData(csr)
	if err != nil {
		return err
	}
	if infoData.CertificateCommonNameID == s.hubID {
		return fmt.Errorf("common name contains same value as hub id(%v)", s.hubID)
	}
	return nil
}

func (s *CertificateAuthorityServer) updateSigningIdentityCertificateRecord(ctx context.Context, updateSigningRecord *pb.SigningRecord) error {
	var found bool
	now := time.Now().UnixNano()
	err := s.store.LoadSigningRecords(ctx, updateSigningRecord.GetOwner(), &store.SigningRecordsQuery{
		CommonNameFilter: []string{updateSigningRecord.GetCommonName()},
	}, func(ctx context.Context, iter store.SigningRecordIter) (err error) {
		for {
			var signingRecord pb.SigningRecord
			ok := iter.Next(ctx, &signingRecord)
			if !ok {
				break
			}
			if updateSigningRecord.GetPublicKey() != signingRecord.GetPublicKey() && signingRecord.GetCredential().GetValidUntilDate() > now {
				return fmt.Errorf("common name %v with different public key fingerprint exist", signingRecord.GetCommonName())
			}
			found = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if found {
		return s.store.UpdateSigningRecord(ctx, updateSigningRecord)
	}
	return s.store.CreateSigningRecord(ctx, updateSigningRecord)
}

func toSigningRecord(owner string, template *x509.Certificate) (*pb.SigningRecord, error) {
	publicKeyRaw, err := x509.MarshalPKIXPublicKey(template.PublicKey)
	if err != nil {
		return nil, err
	}

	publicKey := uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw).String()
	id := uuid.NewSHA1(uuid.NameSpaceX500, append([]byte(template.Subject.CommonName), publicKeyRaw...)).String()
	now := time.Now().UnixNano()

	m := regexp.MustCompile("uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")
	deviceIDCommonName := m.FindString(template.Subject.CommonName)
	deviceID := ""
	if deviceIDCommonName != "" {
		deviceID = deviceIDCommonName[5:]
	}

	return &pb.SigningRecord{
		Id:           id,
		Owner:        owner,
		CommonName:   template.Subject.CommonName,
		PublicKey:    publicKey,
		CreationDate: now,
		DeviceId:     deviceID,
		Credential: &pb.CredentialStatus{
			CertificatePem: "",
			Date:           now,
			ValidUntilDate: template.NotAfter.UnixNano(),
		},
	}, nil
}

func (s *CertificateAuthorityServer) updateSigningRecord(ctx context.Context, signingRecord *pb.SigningRecord) error {
	var checkForIdentity bool
	if signingRecord.GetDeviceId() != "" && signingRecord.GetDeviceId() != signingRecord.GetOwner() {
		checkForIdentity = true
	}
	if checkForIdentity {
		return s.updateSigningIdentityCertificateRecord(ctx, signingRecord)
	}
	return s.store.UpdateSigningRecord(ctx, signingRecord)
}

func (s *CertificateAuthorityServer) SignCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	const fmtError = "cannot sign certificate: %v"
	logger := s.logger.With("csr", string(req.GetCertificateSigningRequest()))
	if err := s.validateRequest(req.GetCertificateSigningRequest()); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	signer := s.GetSigner()
	if signer == nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, fmt.Errorf("signer is empty")))
	}
	cert, signingRecord, err := signer.Sign(ctx, req.GetCertificateSigningRequest())
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	if signingRecord.GetCredential() == nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: cannot create signing record"))
	}
	signingRecord.Credential.CertificatePem = string(cert)
	if err := s.updateSigningRecord(ctx, signingRecord); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	logger.With("crt", string(cert)).Debugf("CertificateAuthorityServer.SignCertificate")

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
