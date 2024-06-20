package commands

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	extCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/grpc/codes"
)

const (
	ResourceLinksHref    string = "/plgd/res"
	StatusHref           string = "/plgd/s"
	ServicesResourceHref string = "/plgd/services"
)

// ToUUID converts resource href and device id to unique resource ID
func (r *ResourceId) ToUUID() uuid.UUID {
	if len(r.GetHref()) == 0 {
		return uuid.Nil
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(r.GetDeviceId()+r.GetHref()))
}

// ToUUID converts resource href and device id to unique resource ID
func (r *Resource) ToUUID() uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(r.GetDeviceId()+r.GetHref()))
}

// GetResourceID converts resource href and device id to resource id struct
func (r *Resource) GetResourceID() *ResourceId {
	return &ResourceId{DeviceId: r.GetDeviceId(), Href: r.GetHref()}
}

func MakeLinksResourceUUID(deviceID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(deviceID+ResourceLinksHref))
}

func MakeStatusResourceUUID(deviceID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(deviceID+StatusHref))
}

func MakeServicesResourceUUID(hubID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(hubID+ServicesResourceHref))
}

func NewResourceID(deviceID, href string) *ResourceId {
	return &ResourceId{DeviceId: deviceID, Href: href}
}

func (r *Resource) IsObservable() bool {
	return r.GetPolicy() != nil && r.GetPolicy().GetBitFlags()&int32(schema.Observable) != 0
}

var http2status = map[int]Status{
	http.StatusAccepted:           Status_ACCEPTED,
	http.StatusNotModified:        Status_NOT_MODIFIED,
	http.StatusOK:                 Status_OK,
	http.StatusBadRequest:         Status_BAD_REQUEST,
	http.StatusNotFound:           Status_NOT_FOUND,
	http.StatusNotImplemented:     Status_NOT_IMPLEMENTED,
	http.StatusForbidden:          Status_FORBIDDEN,
	http.StatusUnauthorized:       Status_UNAUTHORIZED,
	http.StatusMethodNotAllowed:   Status_METHOD_NOT_ALLOWED,
	http.StatusCreated:            Status_CREATED,
	http.StatusNoContent:          Status_OK,
	http.StatusServiceUnavailable: Status_UNAVAILABLE,
}

var status2http = map[Status]int{
	Status_ACCEPTED:           http.StatusAccepted,
	Status_OK:                 http.StatusOK,
	Status_NOT_MODIFIED:       http.StatusNotModified,
	Status_BAD_REQUEST:        http.StatusBadRequest,
	Status_NOT_FOUND:          http.StatusNotFound,
	Status_NOT_IMPLEMENTED:    http.StatusNotImplemented,
	Status_FORBIDDEN:          http.StatusForbidden,
	Status_UNAUTHORIZED:       http.StatusUnauthorized,
	Status_METHOD_NOT_ALLOWED: http.StatusMethodNotAllowed,
	Status_CREATED:            http.StatusCreated,
	Status_UNAVAILABLE:        http.StatusServiceUnavailable,
	Status_UNKNOWN:            http.StatusServiceUnavailable,
}

func HTTPStatus2Status(s int) Status {
	v, ok := http2status[s]
	if ok {
		return v
	}
	return Status_UNKNOWN
}

// IsOnline evaluates online state
func (c *Connection) IsOnline() bool {
	if c == nil {
		return false
	}
	return c.GetStatus() == Connection_ONLINE
}

var status2grpcCode = map[Status]codes.Code{
	Status_OK:                 codes.OK,
	Status_BAD_REQUEST:        codes.InvalidArgument,
	Status_UNAUTHORIZED:       codes.Unauthenticated,
	Status_FORBIDDEN:          codes.PermissionDenied,
	Status_NOT_FOUND:          codes.NotFound,
	Status_UNAVAILABLE:        codes.Unavailable,
	Status_NOT_IMPLEMENTED:    codes.Unimplemented,
	Status_ACCEPTED:           codes.Code(extCodes.Accepted),
	Status_ERROR:              codes.Internal,
	Status_METHOD_NOT_ALLOWED: codes.Code(extCodes.MethodNotAllowed),
	Status_CREATED:            codes.Code(extCodes.Created),
	Status_NOT_MODIFIED:       codes.Code(extCodes.Valid),
}

func (s Status) ToGrpcCode() codes.Code {
	v, ok := status2grpcCode[s]
	if ok {
		return v
	}
	return codes.Unknown
}

func (s Status) ToHTTPCode() int {
	v, ok := status2http[s]
	if ok {
		return v
	}
	return http.StatusInternalServerError
}

func (r *ResourceId) Equal(r1 *ResourceId) bool {
	if r == nil && r1 == nil {
		return true
	}
	if r == nil || r1 == nil {
		return false
	}
	return r.GetDeviceId() == r1.GetDeviceId() && r.GetHref() == r1.GetHref()
}

func (r *ResourceId) ToString() string {
	if r == nil {
		return ""
	}
	deviceID := r.GetDeviceId()
	if deviceID == "" {
		return ""
	}
	href := r.GetHref()
	if href == "" {
		return ""
	}
	if href[0] != '/' {
		href = "/" + href
	}
	return deviceID + href
}

func ResourceIdFromString(v string) *ResourceId {
	if len(v) > 0 && v[0] == '/' {
		v = v[1:]
	}
	deviceIDHref := strings.SplitN(v, "/", 2)
	if len(deviceIDHref) != 2 {
		return nil
	}
	return &ResourceId{
		DeviceId: deviceIDHref[0],
		Href:     "/" + deviceIDHref[1],
	}
}

func DecodeContent(content *Content, v interface{}) error {
	if content == nil {
		return errors.New("cannot parse empty content")
	}

	var decode func([]byte, interface{}) error

	switch content.GetContentType() {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		decode = cbor.Decode
	case message.AppJSON.String():
		decode = json.Decode
	case message.TextPlain.String():
		switch out := v.(type) {
		case *string:
			*out = string(content.GetData())
		case *[]byte:
			*out = content.GetData()
		case *interface{}:
			*out = string(content.GetData())
		default:
			return fmt.Errorf("cannot decode content: invalid type (%T)", v)
		}
		return nil
	default:
		return fmt.Errorf("unsupported content type: %v", content.GetContentType())
	}

	return decode(content.GetData(), v)
}
