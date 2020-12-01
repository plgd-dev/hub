package pb

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/go-coap/v2/message"
)

func RAStatus2Status(s pbRA.Status) Status {
	switch s {
	case pbRA.Status_OK:
		return Status_OK
	case pbRA.Status_BAD_REQUEST:
		return Status_BAD_REQUEST
	case pbRA.Status_UNAUTHORIZED:
		return Status_UNAUTHORIZED
	case pbRA.Status_FORBIDDEN:
		return Status_FORBIDDEN
	case pbRA.Status_NOT_FOUND:
		return Status_NOT_FOUND
	case pbRA.Status_UNAVAILABLE:
		return Status_UNAVAILABLE
	case pbRA.Status_NOT_IMPLEMENTED:
		return Status_NOT_IMPLEMENTED
	case pbRA.Status_ACCEPTED:
		return Status_ACCEPTED
	case pbRA.Status_ERROR:
		return Status_ERROR
	case pbRA.Status_METHOD_NOT_ALLOWED:
		return Status_METHOD_NOT_ALLOWED
	}
	return Status_UNKNOWN
}

func (r *ResourceId) ID() string {
	return cqrs.MakeResourceId(r.GetDeviceId(), r.GetHref())
}

func RAContent2Content(s *pbRA.Content) *Content {
	if s == nil {
		return nil
	}
	contentType := s.GetContentType()
	if contentType == "" {
		if s.GetCoapContentFormat() < 0 {
			return nil
		}
		contentType = message.MediaType(s.GetCoapContentFormat()).String()
	}

	return &Content{
		Data:        s.GetData(),
		ContentType: contentType,
	}
}
