package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/coap"
	"go.uber.org/atomic"
)

// Session represents a setup of connection
type Session struct {
	server                    *Service
	coapConn                  mux.Conn
	chains                    [][]*x509.Certificate
	deviceID                  atomic.String
	err                       error
	enrollmentGroup           *EnrollmentGroup
	manufacturerCertificateID string
	localEndpoints            atomic.Pointer[[]string]
}

func toManufacturerCertificateID(publicKey any) (string, error) {
	publicKeyRaw, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	return uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw).String(), nil
}

// newSession creates and initializes a session
func newSession(server *Service, coapConn mux.Conn, chains [][]*x509.Certificate) *Session {
	// enrollmentGroup can be nil - any request to from this session ends with error and enrollmentGroup is nil
	var enrollmentGroup *EnrollmentGroup
	var manufacturerCertificateID string
	var err error
	var deviceID string
	if len(chains) == 0 || len(chains[0]) == 0 {
		err = errors.New("unable to find enrollment group for empty certificate chain")
	} else {
		ctx, cancel := context.WithTimeout(coapConn.Context(), server.config.APIs.COAP.InactivityMonitor.Timeout)
		defer cancel()
		group, ok, err1 := server.enrollmentGroupsCache.GetEnrollmentGroup(ctx, chains)
		switch {
		case err1 != nil:
			err = fmt.Errorf("unable to find enrollment group for the manufacturer certificate %v: %w", chains[0][0].Subject.CommonName, err1)
		case ok:
			manufacturerCertificateID, err1 = toManufacturerCertificateID(chains[0][0].PublicKey)
			if err1 != nil {
				err = fmt.Errorf("unable to marshal public key of manufacturer certificate %v: %w", chains[0][0].Subject.CommonName, err1)
			} else {
				enrollmentGroup = group
			}
		default:
			err = fmt.Errorf("unable to find enrollment group for the manufacturer certificate %v", chains[0][0].Subject.CommonName)
		}
	}
	s := Session{
		server:                    server,
		coapConn:                  coapConn,
		chains:                    chains,
		err:                       err,
		enrollmentGroup:           enrollmentGroup,
		manufacturerCertificateID: manufacturerCertificateID,
	}
	s.deviceID.Store(deviceID)
	if len(chains) > 0 && len(chains[0]) > 0 {
		s.updateProvisioningRecord(&store.ProvisioningRecord{
			Attestation: &pb.Attestation{
				Date: time.Now().UnixNano(),
				X509: &pb.X509Attestation{
					CertificatePem: certToPem(chains[0][0]),
					CommonName:     chains[0][0].Subject.CommonName,
				},
			},
		})
	}
	return &s
}

func (s *Session) RemoteAddr() net.Addr {
	return s.coapConn.RemoteAddr()
}

func (s *Session) DeviceID() string {
	return s.deviceID.Load()
}

func (s *Session) SetDeviceID(deviceID string) {
	s.deviceID.Store(deviceID)
}

func (s *Session) String() string {
	if len(s.chains) == 0 {
		return ""
	}
	if len(s.chains[0]) == 0 {
		return ""
	}
	return s.chains[0][0].Subject.CommonName
}

func (s *Session) Context() context.Context {
	return s.coapConn.Context()
}

// Close closes coap connection
func (s *Session) Close() error {
	// wait one second for send response
	time.Sleep(time.Second)
	if err := s.coapConn.Close(); err != nil {
		return fmt.Errorf("cannot close session: %w", err)
	}
	return nil
}

// OnClose action when coap connection was closed.
func (s *Session) OnClose() {
	s.Debugf("session was closed")
}

func (s *Session) WriteMessage(m *pool.Message) error {
	return s.coapConn.WriteMessage(m)
}

func (s *Session) createResponse(code codes.Code, token message.Token, contentFormat message.MediaType, payload []byte) *pool.Message {
	msg := s.server.messagePool.AcquireMessage(s.coapConn.Context())
	msg.SetCode(code)
	msg.SetToken(token)
	if len(payload) > 0 {
		msg.SetContentFormat(contentFormat)
		msg.SetBody(bytes.NewReader(payload))
	}
	return msg
}

func (s *Session) Errorf(fmt string, args ...interface{}) {
	logger := s.getLogger()
	logger.Errorf(fmt, args...)
}

func (s *Session) Debugf(fmt string, args ...interface{}) {
	logger := s.getLogger()
	logger.Debugf(fmt, args...)
}

func (s *Session) getGroupAndLinkedHubs(ctx context.Context) ([]*LinkedHub, *EnrollmentGroup, error) {
	if s.enrollmentGroup == nil {
		return nil, nil, errors.New("cannot get enrollment group: not found")
	}
	linkedHub, err := s.server.linkedHubCache.GetHubs(ctx, s.enrollmentGroup)
	if err != nil {
		return nil, s.enrollmentGroup, fmt.Errorf("cannot get linked hub for enrollment group for %v: %w", s.enrollmentGroup, err)
	}
	return linkedHub, s.enrollmentGroup, err
}

func (s *Session) checkForError() error {
	return s.err
}

func (s *Session) updateProvisioningRecord(provisionedDevice *store.ProvisioningRecord) {
	if s.err != nil {
		return
	}
	provisionedDevice.EnrollmentGroupId = s.enrollmentGroup.GetId()
	provisionedDevice.Id = s.manufacturerCertificateID
	provisionedDevice.LocalEndpoints = s.getLocalEndpoints()
	provisionedDevice.Owner = s.enrollmentGroup.GetOwner()

	err := s.server.store.UpdateProvisioningRecord(s.Context(), provisionedDevice.GetOwner(), provisionedDevice)
	if err != nil {
		s.Errorf("cannot update record about provisioned device at DB: %v", err)
	}
}

func (s *Session) resolveLocalEndpoints() {
	v := []string{}
	if !s.localEndpoints.CompareAndSwap(nil, &v) {
		return
	}
	endpoints, err := coap.GetEndpointsFromDeviceResource(s.coapConn.Context(), s.coapConn)
	if err != nil {
		s.Errorf("cannot get local endpoints: %w", err)
		return
	}
	s.localEndpoints.Store(&endpoints)
	s.getLogger().With(log.LocalEndpointsKey, endpoints).Debug("local endpoints retrieval successful.")
}

func (s *Session) getLocalEndpoints() []string {
	localEndpoints := s.localEndpoints.Load()
	if localEndpoints == nil {
		return nil
	}
	if len(*localEndpoints) == 0 {
		return nil
	}
	return *localEndpoints
}
