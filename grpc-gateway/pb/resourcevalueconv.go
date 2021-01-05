package pb

import (
	extCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	"google.golang.org/grpc/codes"
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

var status2grpcCode = map[Status]codes.Code{
	Status_OK:                 codes.OK,
	Status_BAD_REQUEST:        codes.InvalidArgument,
	Status_UNAUTHORIZED:       codes.Unauthenticated,
	Status_FORBIDDEN:          codes.PermissionDenied,
	Status_NOT_FOUND:          codes.NotFound,
	Status_UNAVAILABLE:        codes.Unavailable,
	Status_NOT_IMPLEMENTED:    codes.Unimplemented,
	Status_ACCEPTED:           extCodes.Accepted,
	Status_ERROR:              codes.Internal,
	Status_METHOD_NOT_ALLOWED: extCodes.MethodNotAllowed,
}

func (s Status) ToGrpcCode() codes.Code {
	v, ok := status2grpcCode[s]
	if ok {
		return v
	}
	return codes.Unknown
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
