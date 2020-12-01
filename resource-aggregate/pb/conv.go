package pb

import "net/http"

var http2status = map[int]Status{
	http.StatusAccepted:         Status_ACCEPTED,
	http.StatusOK:               Status_OK,
	http.StatusBadRequest:       Status_BAD_REQUEST,
	http.StatusNotFound:         Status_NOT_FOUND,
	http.StatusNotImplemented:   Status_NOT_IMPLEMENTED,
	http.StatusForbidden:        Status_FORBIDDEN,
	http.StatusUnauthorized:     Status_UNAUTHORIZED,
	http.StatusMethodNotAllowed: Status_METHOD_NOT_ALLOWED,
}

func HTTPStatus2Status(s int) Status {
	v, ok := http2status[s]
	if ok {
		return v
	}
	return Status_UNKNOWN
}
