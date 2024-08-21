package service

import (
	"context"
	"time"

	"github.com/plgd-dev/device/v2/schema/doxm"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/identity-store/events"
)

func (RequestHandle) ProcessOwnership(_ context.Context, req *mux.Message, session *Session, _ []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.GET:
		msg, err := provisionOwnership(req, session, group)
		session.updateProvisioningRecord(&store.ProvisioningRecord{
			Ownership: &pb.OwnershipStatus{
				Status: &pb.ProvisionStatus{
					Date:         time.Now().UnixNano(),
					CoapCode:     toCoapCode(msg),
					ErrorMessage: toErrorStr(err),
				},
				Owner: group.GetOwner(),
			},
		})
		return msg, err
	default:
		return nil, statusErrorf(coapCodes.Forbidden, "unsupported command(%v)", req.Code())
	}
}

func provisionOwnership(req *mux.Message, session *Session, group *EnrollmentGroup) (*pool.Message, error) {
	owner := events.OwnerToUUID(group.GetOwner())
	resp := doxm.DoxmUpdate{
		OwnerID: &owner,
	}
	msgType, data, err := encodeResponse(resp, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "cannot encode ownership response: %w", err)
	}
	return session.createResponse(coapCodes.Content, req.Token(), msgType, data), nil
}
