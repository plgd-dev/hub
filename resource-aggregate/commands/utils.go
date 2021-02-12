package commands

import (
	"net/http"

	"github.com/gofrs/uuid"
)

const ResourceLinksHref = "/plgd/res"

// ToUUID converts resource href and device id to unique resource ID
func (r *ResourceId) ToUUID() string {
	return uuid.NewV5(uuid.NamespaceURL, r.DeviceId+r.Href).String()
}

func MakeLinksResourceUUID(deviceID string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+ResourceLinksHref).String()
}

func MakeAuditContext(deviceID, userID, correlationId string) AuditContext {
	return AuditContext{
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
