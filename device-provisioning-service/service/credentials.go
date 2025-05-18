package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	dpsPb "github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/security/oauth/clientcredentials"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/security"
)

type CSR struct {
	Encoding csr.CertificateEncoding `json:"encoding"`
	Data     string                  `json:"data"`
}

type CredentialsRequest struct {
	CSR             CSR            `json:"csr"`
	SelectedGateway cloud.Endpoint `json:"selectedGateway"`
}

func (RequestHandle) ProcessCredentials(ctx context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		msg, deviceID, identityCertPem, psk, credentials, err := provisionCredentials(ctx, req, session, linkedHubs, group)
		session.updateProvisioningRecord(&store.ProvisioningRecord{
			DeviceId: deviceID,
			Credential: &dpsPb.CredentialStatus{
				Status: &dpsPb.ProvisionStatus{
					Date:         time.Now().UnixNano(),
					CoapCode:     toCoapCode(msg),
					ErrorMessage: toErrorStr(err),
				},
				IdentityCertificatePem: identityCertPem,
				PreSharedKey:           psk,
				Credentials:            credentials,
			},
		})
		return msg, err
	default:
		return nil, statusErrorf(coapCodes.Forbidden, "unsupported command(%v)", req.Code())
	}
}

func (s *Session) signCertificate(ctx context.Context, linkedHub *LinkedHub, csrReq *CredentialsRequest) ([]*x509.Certificate, error) {
	if len(csrReq.CSR.Data) == 0 {
		// csr property was not set so we will send only CAs
		return nil, nil
	}
	if csrReq.CSR.Encoding != csr.CertificateEncoding_PEM {
		return nil, fmt.Errorf("unsupported encoding (%v)", csrReq.CSR.Encoding)
	}
	resp, err := linkedHub.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: []byte(csrReq.CSR.Data)})
	if err != nil {
		return nil, fmt.Errorf("cannot sign identity certificate: %w", err)
	}
	identityChanCert := resp.GetCertificate()

	certsFromChain, err := security.ParseX509FromPEM(identityChanCert)
	if err != nil {
		return nil, fmt.Errorf("cannot parse chain of X509 certs: %w", err)
	}
	return certsFromChain, nil
}

func encodeResponse(resp interface{}, options message.Options) (message.MediaType, []byte, error) {
	accept := coapconv.GetAccept(options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return 0, nil, err
	}
	out, err := encode(resp)
	if err != nil {
		return 0, nil, err
	}
	return accept, out, nil
}

func certsToPem(certs []*x509.Certificate) string {
	certificateChain := make([]byte, 0, 4096)
	for _, cert := range certs {
		certificateChain = append(certificateChain, pem.EncodeToMemory(&pem.Block{
			Type: "CERTIFICATE", Bytes: cert.Raw,
		})...)
	}
	return string(certificateChain)
}

func certToPem(c *x509.Certificate) string {
	return certsToPem([]*x509.Certificate{c})
}

func processErrClaimNotFound(err error, logger log.Logger, linkedHub *LinkedHub, group *EnrollmentGroup) error {
	var claimNotFound clientcredentials.ClaimNotFoundError
	if errors.As(err, &claimNotFound) && claimNotFound.Claim == linkedHub.cfg.GetAuthorization().GetOwnerClaim() {
		err = fmt.Errorf("configured OAuth client for the hub %v used in enrollment group %v returned owner id claim %v which doesn't match the expected value %v",
			linkedHub.cfg.GetId(), group.GetId(), claimNotFound.Claim, group.GetOwner())
		logger.Warn(err)
	}
	return err
}

func parseProvisionCredentialsReq(req *mux.Message) (*CredentialsRequest, error) {
	if req.Body() == nil {
		// body is empty we will send only CAs from configuration.
		return nil, errors.New("empty body")
	}
	ct, err := req.Options().ContentFormat()
	if err != nil {
		return nil, fmt.Errorf("cannot get content type: %w", err)
	}

	var csrReq CredentialsRequest
	switch ct {
	case message.AppCBOR, message.AppOcfCbor:
		err = cbor.ReadFrom(req.Body(), &csrReq)
		if err != nil {
			return nil, fmt.Errorf("cannot decode body: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported content type(%v)", ct)
	}
	return &csrReq, nil
}

func provisionCredentials(ctx context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, group *EnrollmentGroup) (resp *pool.Message, deviceID string, identityCertPem string, psk *dpsPb.PreSharedKey, credentials []*dpsPb.Credential, err error) {
	credReq, err := parseProvisionCredentialsReq(req)
	if err != nil {
		return nil, "", "", nil, nil, statusErrorf(coapCodes.BadRequest, "cannot parse request: %w", err)
	}

	linkedHub, _, err := findLinkedHub(credReq.SelectedGateway, linkedHubs)
	if err != nil {
		return nil, "", "", nil, nil, statusErrorf(coapCodes.BadRequest, "cannot find linked hub: %w", err)
	}

	token, err := linkedHub.GetToken(ctx, group.Owner, map[string]string{
		linkedHub.cfg.GetAuthorization().GetOwnerClaim(): group.Owner,
	}, map[string]interface{}{
		linkedHub.cfg.GetAuthorization().GetOwnerClaim(): group.Owner,
	})
	if err != nil {
		err = processErrClaimNotFound(err, session.getLogger(), linkedHub, group)
		return nil, "", "", nil, nil, statusErrorf(coapCodes.InternalServerError, "cannot get token for enrollment group %v: %w", group.GetId(), err)
	}
	ctx = grpc.CtxWithToken(ctx, token.AccessToken)
	ownerID := events.OwnerToUUID(group.Owner)
	chain, err := session.signCertificate(ctx, linkedHub, credReq)
	if err != nil {
		return nil, "", "", nil, nil, statusErrorf(coapCodes.BadRequest, "%w", err)
	}
	if len(chain) == 0 {
		return nil, "", "", nil, nil, statusErrorf(coapCodes.InternalServerError, "unexpected empty chain")
	}
	var credUpdResp credential.CredentialUpdateRequest
	key, ok, err := group.ResolvePreSharedKey()
	if err == nil && ok {
		psk = &dpsPb.PreSharedKey{
			SubjectId: ownerID,
			Key:       key[:16],
		}
		credUpdResp.Credentials = append(credUpdResp.Credentials, credential.Credential{
			Subject: ownerID,
			Type:    credential.CredentialType_SYMMETRIC_PAIR_WISE,
			PrivateData: &credential.CredentialPrivateData{
				DataInternal: psk.GetKey(),
				Encoding:     credential.CredentialPrivateDataEncoding_RAW,
			},
			Tag: DPSTag,
		})
	}
	if len(chain) > 0 {
		deviceID, err = coap.GetDeviceIDFromIdentityCertificate(chain[0])
		if err != nil {
			return nil, "", "", nil, nil, statusErrorf(coapCodes.BadRequest, "%w", err)
		}
		session.SetDeviceID(deviceID)

		identityCert := make([]byte, 0, 1024)
		for i := range len(chain) - 1 {
			identityCert = append(identityCert, certToPem(chain[i])...)
		}
		identityCertPem = string(identityCert)

		credUpdResp.Credentials = append(credUpdResp.Credentials, credential.Credential{
			Subject: deviceID,
			Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
			Usage:   credential.CredentialUsage_CERT,
			PublicData: &credential.CredentialPublicData{
				DataInternal: identityCertPem,
				Encoding:     credential.CredentialPublicDataEncoding_PEM,
			},
			Tag: DPSTag,
		})
		credUpdResp.Credentials = append(credUpdResp.Credentials, credential.Credential{
			Subject: ownerID,
			Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
			Usage:   credential.CredentialUsage_TRUST_CA,
			PublicData: &credential.CredentialPublicData{
				DataInternal: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: chain[len(chain)-1].Raw})),
				Encoding:     credential.CredentialPublicDataEncoding_PEM,
			},
			Tag: DPSTag,
		})
	}

	msgType, data, err := encodeResponse(credUpdResp, req.Options())
	if err != nil {
		return nil, "", "", nil, nil, statusErrorf(coapCodes.BadRequest, "cannot encode credentials: %w", err)
	}
	return session.createResponse(coapCodes.Changed, req.Token(), msgType, data), deviceID, identityCertPem, psk, dpsPb.CredentialsToPb(credUpdResp.Credentials), nil
}
