package service

import (
	"context"
	"time"

	"github.com/plgd-dev/device/v2/schema/plgdtime"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
)

func (RequestHandle) ProcessPlgdTime(_ context.Context, req *mux.Message, session *Session, _ []*LinkedHub, _ *EnrollmentGroup) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.GET:
		msg, err := provisionPlgdTime(req, session)
		session.updateProvisioningRecord(&store.ProvisioningRecord{
			PlgdTime: &pb.ProvisionStatus{
				Date:         time.Now().UnixNano(),
				CoapCode:     toCoapCode(msg),
				ErrorMessage: toErrorStr(err),
			},
		})
		return msg, err
	default:
		return nil, statusErrorf(coapCodes.Forbidden, "unsupported command(%v)", req.Code())
	}
}

func provisionPlgdTime(req *mux.Message, session *Session) (*pool.Message, error) {
	resp := plgdtime.PlgdTimeUpdate{
		Time: time.Now().Format(time.RFC3339Nano),
	}
	msgType, data, err := encodeResponse(resp, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "cannot encode plgdTime response: %w", err)
	}
	return session.createResponse(coapCodes.Content, req.Token(), msgType, data), nil
}
