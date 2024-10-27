package grpc

import (
	"context"
	"crypto/x509"
	"errors"
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

func toSigningRecord(owner, issuerID string, template *x509.Certificate) (*pb.SigningRecord, error) {
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
			Serial:         template.SerialNumber.String(),
			IssuerId:       issuerID,
		},
	}, nil
}

func (s *CertificateAuthorityServer) getSigningRecord(ctx context.Context, signingRecord *pb.SigningRecord) (*pb.SigningRecord, error) {
	checkForIdentity := signingRecord.GetDeviceId() != "" && signingRecord.GetDeviceId() != signingRecord.GetOwner()
	var err error
	var originalSr *store.SigningRecord
	if checkForIdentity {
		now := time.Now().UnixNano()
		err = s.store.LoadSigningRecords(ctx, signingRecord.GetOwner(), &store.SigningRecordsQuery{
			CommonNameFilter: []string{signingRecord.GetCommonName()},
		}, func(sr *store.SigningRecord) (err error) {
			// _id is calculated as uuid.NewSHA1(uuid.NameSpaceX500, CommonName + PublicKey) -> thus same CommonName and PublicKey == same _id
			if signingRecord.GetPublicKey() != sr.GetPublicKey() &&
				sr.GetCredential().GetValidUntilDate() > now {
				return fmt.Errorf("common name %v with different public key fingerprint exists", sr.GetCommonName())
			}
			if signingRecord.GetId() == sr.GetId() {
				originalSr = sr
			}
			return nil
		})
	} else {
		err = s.store.LoadSigningRecords(ctx, signingRecord.GetOwner(), &store.SigningRecordsQuery{
			IdFilter: []string{signingRecord.GetId()},
		}, func(sr *store.SigningRecord) (err error) {
			originalSr = sr
			return nil
		})
	}
	if err != nil {
		return nil, err
	}
	return originalSr, nil
}

func (s *CertificateAuthorityServer) updateRevocationListForSigningRecord(ctx context.Context, sr, prevSr *pb.SigningRecord) error {
	if prevSr != nil {
		// revoke previous signing record
		prevCred := prevSr.GetCredential()
		if prevCred != nil {
			query := store.UpdateRevocationListQuery{
				IssuerID: prevCred.GetIssuerId(),
				RevokedCertificates: []*store.RevocationListCertificate{
					{
						Serial:     prevCred.GetSerial(),
						ValidUntil: prevCred.GetValidUntilDate(),
						Revocation: time.Now().UnixNano(),
					},
				},
			}
			_, err := s.store.UpdateRevocationList(ctx, &query)
			if err != nil {
				return fmt.Errorf("failed to update revocation list: %w", err)
			}
		}
		return nil
	}
	cred := sr.GetCredential()
	if cred != nil {
		// create new RevocationList if it doesn't exist
		err := s.store.InsertRevocationLists(ctx, &store.RevocationList{
			Id:     cred.GetIssuerId(),
			Number: "1",
		})
		if errors.Is(err, store.ErrDuplicateID) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to create revocation list: %w", err)
		}
	}
	return nil
}

func (s *CertificateAuthorityServer) updateSigningRecord(ctx context.Context, signingRecord *pb.SigningRecord) error {
	// try to get previous signing record
	prevSr, err := s.getSigningRecord(ctx, signingRecord)
	if err != nil {
		return err
	}
	if s.store.SupportsRevocationList() {
		err = s.updateRevocationListForSigningRecord(ctx, signingRecord, prevSr)
		if err != nil {
			return err
		}
	}
	// upsert new one
	err = s.store.UpdateSigningRecord(ctx, signingRecord)
	return err
}

func (s *CertificateAuthorityServer) SignCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	const fmtError = "cannot sign certificate: %v"
	logger := s.logger.With("csr", string(req.GetCertificateSigningRequest()))
	if err := s.validateRequest(req.GetCertificateSigningRequest()); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	signer := s.GetSigner()
	if signer == nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, errors.New("signer is empty")))
	}
	cert, signingRecord, err := signer.Sign(ctx, req.GetCertificateSigningRequest())
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	credential := signingRecord.GetCredential()
	if credential == nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, errors.New("cannot create signing record")))
	}
	credential.CertificatePem = string(cert)
	if err := s.updateSigningRecord(ctx, signingRecord); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	logger.With("crt", string(cert)).Debugf("CertificateAuthorityServer.SignCertificate")

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
