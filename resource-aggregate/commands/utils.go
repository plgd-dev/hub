package commands

import (
	"net/http"

	"github.com/gofrs/uuid"
	statusResource "github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
)

const ResourceLinksHref = "/plgd/res"

// ToUUID converts resource href and device id to unique resource ID
func (r *ResourceId) ToUUID() string {
	return uuid.NewV5(uuid.NamespaceURL, r.DeviceId+r.Href).String()
}

// ToUUID converts resource href and device id to unique resource ID
func (r *Resource) ToUUID() string {
	return uuid.NewV5(uuid.NamespaceURL, r.DeviceId+r.Href).String()
}

// GetResourceId converts resource href and device id to resource id struct
func (r *Resource) GetResourceId() *ResourceId {
	return &ResourceId{DeviceId: r.DeviceId, Href: r.Href}
}

func MakeLinksResourceUUID(deviceID string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+ResourceLinksHref).String()
}

func MakeStatusResourceUUID(deviceID string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+statusResource.Href).String()
}

func MakeResourceID(deviceID, href string) *ResourceId {
	return &ResourceId{DeviceId: deviceID, Href: href}
}

func MakeAuditContext(deviceID, userID, correlationId string) *AuditContext {
	return &AuditContext{
		UserId:        userID,
		DeviceId:      deviceID,
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
