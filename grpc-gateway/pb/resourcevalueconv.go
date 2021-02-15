package pb

import (
	extCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	"google.golang.org/grpc/codes"
)

var rastatus2status = map[commands.Status]Status{
	commands.Status_OK:                 Status_OK,
	commands.Status_BAD_REQUEST:        Status_BAD_REQUEST,
	commands.Status_UNAUTHORIZED:       Status_UNAUTHORIZED,
	commands.Status_FORBIDDEN:          Status_FORBIDDEN,
	commands.Status_NOT_FOUND:          Status_NOT_FOUND,
	commands.Status_UNAVAILABLE:        Status_UNAVAILABLE,
	commands.Status_NOT_IMPLEMENTED:    Status_NOT_IMPLEMENTED,
	commands.Status_ACCEPTED:           Status_ACCEPTED,
	commands.Status_ERROR:              Status_ERROR,
	commands.Status_METHOD_NOT_ALLOWED: Status_METHOD_NOT_ALLOWED,
	commands.Status_CREATED:            Status_CREATED,
}

func RAStatus2Status(s commands.Status) Status {
	v, ok := rastatus2status[s]
	if ok {
		return v
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
	Status_CREATED:            extCodes.Created,
}

func (s Status) ToGrpcCode() codes.Code {
	v, ok := status2grpcCode[s]
	if ok {
		return v
	}
	return codes.Unknown
}

func (r *ResourceId) ToUUID() string {
	return (&commands.ResourceId{DeviceId: r.GetDeviceId(), Href: r.GetHref()}).ToUUID()
}

func RAContent2Content(s *commands.Content) *Content {
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
