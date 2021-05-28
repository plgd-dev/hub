package commands

import (
	"net/http"
	"time"

	"github.com/gofrs/uuid"
)

const ResourceLinksHref string = "/plgd/res"
const StatusHref string = "/plgd/s"

// ToUUID converts resource href and device id to unique resource ID
func (r *ResourceId) ToUUID() string {
	if len(r.Href) == 0 {
		return ""
	}
	return uuid.NewV5(uuid.NamespaceURL, r.DeviceId+r.Href).String()
}

// ToUUID converts resource href and device id to unique resource ID
func (r *Resource) ToUUID() string {
	return uuid.NewV5(uuid.NamespaceURL, r.DeviceId+r.Href).String()
}

// GetResourceID converts resource href and device id to resource id struct
func (r *Resource) GetResourceID() *ResourceId {
	return &ResourceId{DeviceId: r.DeviceId, Href: r.Href}
}

func MakeLinksResourceUUID(deviceID string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+ResourceLinksHref).String()
}

func MakeStatusResourceUUID(deviceID string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+StatusHref).String()
}

func NewResourceID(deviceID, href string) *ResourceId {
	return &ResourceId{DeviceId: deviceID, Href: href}
}

func (r *Resource) IsObservable() bool {
	return r.GetPolicies() != nil && r.GetPolicies().GetBitFlags()&2 != 0
}

func NewAuditContext(userID, correlationId string) *AuditContext {
	return &AuditContext{
		UserId:        userID,
		CorrelationId: correlationId,
	}
}

var http2status = map[int]Status{
	http.StatusAccepted:         Status_ACCEPTED,
	http.StatusOK:               Status_OK,
	http.StatusBadRequest:       Status_BAD_REQUEST,
	http.StatusNotFound:         Status_NOT_FOUND,
	http.StatusNotImplemented:   Status_NOT_IMPLEMENTED,
	http.StatusForbidden:        Status_FORBIDDEN,
	http.StatusUnauthorized:     Status_UNAUTHORIZED,
	http.StatusMethodNotAllowed: Status_METHOD_NOT_ALLOWED,
	http.StatusCreated:          Status_CREATED,
	http.StatusNoContent:        Status_OK,
}

func HTTPStatus2Status(s int) Status {
	v, ok := http2status[s]
	if ok {
		return v
	}
	return Status_UNKNOWN
}

// IsOnline evaluate online state
func (s *OnlineStatus) IsOnline() bool {
	if !s.Value {
		return false
	}
	if s.ValidUntil <= 0 {
		return s.Value
	}
	return time.Now().Before(time.Unix(0, s.ValidUntil))
}
