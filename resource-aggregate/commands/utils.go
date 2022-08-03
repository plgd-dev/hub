package commands

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema"
	extCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	"google.golang.org/grpc/codes"
)

const (
	ResourceLinksHref string = "/plgd/res"
	StatusHref        string = "/plgd/s"
)

// ToUUID converts resource href and device id to unique resource ID
func (r *ResourceId) ToUUID() string {
	if len(r.Href) == 0 {
		return ""
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(r.DeviceId+r.Href)).String()
}

// ToUUID converts resource href and device id to unique resource ID
func (r *Resource) ToUUID() string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(r.DeviceId+r.Href)).String()
}

// GetResourceID converts resource href and device id to resource id struct
func (r *Resource) GetResourceID() *ResourceId {
	return &ResourceId{DeviceId: r.DeviceId, Href: r.Href}
}

func MakeLinksResourceUUID(deviceID string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(deviceID+ResourceLinksHref)).String()
}

func MakeStatusResourceUUID(deviceID string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(deviceID+StatusHref)).String()
}

func NewResourceID(deviceID, href string) *ResourceId {
	return &ResourceId{DeviceId: deviceID, Href: href}
}

func (r *Resource) IsObservable() bool {
	return r.GetPolicy() != nil && r.GetPolicy().GetBitFlags()&int32(schema.Observable) != 0
}

var http2status = map[int]Status{
	http.StatusAccepted:           Status_ACCEPTED,
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
func (c *ConnectionStatus) IsOnline() bool {
	if c == nil {
		return false
	}
	if c.Value == ConnectionStatus_OFFLINE {
		return false
	}
	if c.ValidUntil <= 0 {
		// s.ValidUntil <= 0 means infinite
		return c.Value == ConnectionStatus_ONLINE
	}
	return time.Now().UnixNano() < c.ValidUntil
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

func (r *ResourceId) ToString() string {
	if r == nil {
		return ""
	}
	if r.DeviceId == "" {
		return ""
	}
	if r.Href == "" {
		return ""
	}
	href := r.Href
	if href[0] != '/' {
		href = "/" + href
	}
	return r.DeviceId + href
}

func ResourceIdFromString(v string) *ResourceId {
	val := strings.SplitN(v, "/", 2)
	if len(val) != 2 {
		return nil
	}
	return &ResourceId{
		DeviceId: val[0],
		Href:     "/" + val[1],
	}
}
